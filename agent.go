package ai_agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
	"github.com/luoxiaojun1992/ai-agent/skill"
	"github.com/luoxiaojun1992/ai-agent/util/prompt"
)

type AgentMode string

const (
	AgentModeChat AgentMode = "chat"
	AgentModeLoop AgentMode = "loop"
)

type Config struct {
	ChatModel        string
	EmbeddingModel   string
	SupervisorModel  string
	OllamaHost       string
	MilvusHost       string
	MilvusCollection string

	AgentMode         AgentMode
	AgentLoopDuration time.Duration
}

type personalInfo struct {
	character string
	role      string
}

func (pi *personalInfo) characterPrompt() string {
	return "Personality: \n" + "You are " + pi.character
}

func (pi *personalInfo) rolePrompt() string {
	return "Role: \n" + "You are " + pi.role
}

func (pi *personalInfo) prompt() string {
	return pi.characterPrompt() + "\n" + pi.rolePrompt()
}

func (pi *personalInfo) setCharacter(character string) *personalInfo {
	pi.character = character
	return pi
}

func (pi *personalInfo) SetRole(role string) *personalInfo {
	pi.role = role
	return pi
}

type Agent struct {
	//todo search engine client
	config *Config

	personalInfo *personalInfo
	skillSet     map[string]skill.Skill

	ollamaCli ollama.IClient
	milvusCli milvus.IClient
}

func NewAgent(ctx context.Context, config *Config) (*Agent, error) {
	milvusCli, err := milvus.NewClient(ctx, &milvus.Config{
		Host: config.MilvusHost,
	})
	if err != nil {
		return nil, err
	}

	return &Agent{
		config:       config,
		personalInfo: &personalInfo{},
		skillSet:     make(map[string]skill.Skill),
		ollamaCli: ollama.NewClient(&ollama.Config{
			Host: config.OllamaHost,
		}),
		milvusCli: milvusCli,
	}, nil
}

func (a *Agent) SetCharacter(character string) *Agent {
	a.personalInfo.setCharacter(character)
	return a
}

func (a *Agent) SetRole(role string) *Agent {
	a.personalInfo.SetRole(role)
	return a
}

func (a *Agent) GetDescription() string {
	return a.personalInfo.prompt()
}

func (a *Agent) toolPrompt() string {
	functionPromptList := make([]string, 0, len(a.skillSet))
	for skillName, skill := range a.skillSet {
		functionPromptList = append(functionPromptList, fmt.Sprintf("%s: %s", skillName, skill.GetDescription()))
	}
	allFunctionPrompt := strings.Join(functionPromptList, "\n\n")
	return fmt.Sprintf(`
When answering questions, if you need to call external tools or resources, please return the function call in JSON format embedded within your response. The JSON should follow this structure:
<tool>{
  "function": "function_name",
  "parameters": {
    "argument1": "value1",
    "argument2": "value2"
  },
  "abort_on_error": true
}
</tool>
For example, if you need to call a 'search' function to look up information about the weather in New York, you should include this JSON in your response:
<tool>
{
  "function": "search",
  "parameters": {
    "query\": "weather in New York"
  },
  "abort_on_error": true
}
</tool>
Here is a list of supported functions you can call when needed:
%s
`, allFunctionPrompt)
}

func (a *Agent) LearnSkill(name string, processor skill.Skill) *Agent {
	a.skillSet[name] = processor
	return a
}

func (a *Agent) Command(ctx context.Context, skillName string, cmdCtx any, callback func(output any) (any, error)) error {
	processor, existed := a.skillSet[skillName]
	if !existed {
		return fmt.Errorf("skill [%s] hasn't been learned", skillName)
	}
	return processor.Do(ctx, cmdCtx, callback)
}

func (a *Agent) talkToOllama(model string, messages []*ollama.Message, callback func(response string) error) (string, error) {
	var responseContent strings.Builder
	if err := a.ollamaCli.Talk(&ollama.ChatRequest{
		Model:    model,
		Messages: messages,
	}, func(response string) error {
		if _, err := responseContent.WriteString(response); err != nil {
			return err
		}
		return callback(response)
	}); err != nil {
		return "", err
	}

	return responseContent.String(), nil
}

func (a *Agent) reviewResponse(response string) (bool, error) {
	checkResult, err := a.talkToOllama(a.config.SupervisorModel, []*ollama.Message{
		{
			Role: "user",
			Content: `Analyze the logical coherence of the following content.
If there are logical problems such as contradictions, unreasonable causal relationships, or incomplete reasoning, output 'true';
if there are no logical problems, output 'false'.
Content to be analyzed:` + "\n" + response,
		},
	}, func(_ string) error {
		return nil
	})
	if err != nil {
		return false, err
	}
	return checkResult == "true", nil
}

func (a *Agent) Close() error {
	return a.milvusCli.Close()
}

type MemoryCtx struct {
	Role    string
	Content string
	Images  []string
	Epoch   int
}

func (mc *MemoryCtx) toOllamaMessage() *ollama.Message {
	return &ollama.Message{
		Role:    mc.Role,
		Content: fmt.Sprintf("[This message has been recalled %d times] ", mc.Epoch) + mc.Content,
		Images:  append(make([]string, 0, len(mc.Images)), mc.Images...),
	}
}

type Memory struct {
	Contexts []*MemoryCtx
}

func NewMemory() *Memory {
	return &Memory{}
}

type AgentDouble struct {
	config *Config

	Agent        *Agent
	memory       *Memory
	mode         AgentMode
	loopDuration time.Duration
	checkpoint   Checkpoint
}

func NewDoubleWithAgent(agent *Agent, config *Config, checkpoint Checkpoint) *AgentDouble {
	return &AgentDouble{
		config:     config,
		Agent:      agent,
		memory:     NewMemory(),
		mode:       config.AgentMode,
		checkpoint: checkpoint,
	}
}

func NewAgentDouble(ctx context.Context, config *Config, checkpoint Checkpoint) (*AgentDouble, error) {
	agent, err := NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	return NewDoubleWithAgent(agent, config, checkpoint), nil
}

func (ad *AgentDouble) loopPrompt() string {
	return "Determine if the conversation should continue. If not, include <loop_end/> in your response."
}

func (ad *AgentDouble) AddMemory(role, content string, images []string) *AgentDouble {
	ad.memory.Contexts = append(ad.memory.Contexts, &MemoryCtx{
		Role:    role,
		Content: content,
		Images:  images,
	})
	return ad
}

func (ad *AgentDouble) AddSystemMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("system", content, images)
}

func (ad *AgentDouble) AddAssistantMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("assistant", content, images)
}

func (ad *AgentDouble) AddUserMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("user", content, images)
}

func (ad *AgentDouble) InitMemory() *AgentDouble {
	personalInfoPrompt := ad.Agent.personalInfo.prompt()
	return ad.AddSystemMemory(personalInfoPrompt, nil).
		AddSystemMemory(ad.Agent.toolPrompt(), nil).
		AddSystemMemory(ad.loopPrompt(), nil)
}

func (ad *AgentDouble) talkToOllamaWithMemory(ctx context.Context, callback func(response string) error) error {
	ollamaMessages := make([]*ollama.Message, 0, len(ad.memory.Contexts))
	for _, memCtx := range ad.memory.Contexts {
		memCtx.Epoch++
		ollamaMessages = append(ollamaMessages, memCtx.toOllamaMessage())
	}
	responseContentStr, err := ad.Agent.talkToOllama(ad.config.ChatModel, ollamaMessages, callback)
	if err != nil {
		return err
	}

	//todo test if output only contains true or false
	isCompliant, err := ad.Agent.reviewResponse(responseContentStr)
	if err != nil {
		return err
	}
	if !isCompliant {
		return errors.New("response from model is non-compliant")
	}

	ad.AddAssistantMemory(responseContentStr, nil)

	functionCallList, err := prompt.ParseFunctionCalling(responseContentStr)
	if err != nil {
		return err
	}
	for _, functionCall := range functionCallList {
		if err := ad.Agent.Command(ctx, functionCall.Function, functionCall.Parameters, func(output any) (any, error) {
			resultOfFunCall := fmt.Sprintf("The result of function [%s]: %v", functionCall.Function, output)
			ad.AddSystemMemory(resultOfFunCall, nil)
			callback(resultOfFunCall)
			return nil, nil
		}); err != nil {
			errorOfFuncCall := fmt.Sprintf("The error [%s] happened during executing the function [%s].",
				err.Error(),
				functionCall.Function)
			ad.AddSystemMemory(errorOfFuncCall, nil)
			callback(errorOfFuncCall)
			if functionCall.AbortOnError {
				break
			}
			continue
		}

		successOfFuncCall := fmt.Sprintf("The function [%s] has been executed successfully.", functionCall.Function)
		ad.AddSystemMemory(successOfFuncCall, nil)
		callback(successOfFuncCall)
	}

	if ad.checkpoint != nil {
		if err := ad.checkpoint.Do(ad); err != nil {
			return err
		}
	}

	if ad.mode == AgentModeLoop && !prompt.ParseLoopEnd(responseContentStr) {
		time.Sleep(ad.loopDuration)
		return ad.talkToOllamaWithMemory(ctx, callback)
	}

	return nil
}

func (ad *AgentDouble) ListenAndWatch(ctx context.Context, message string, images []string, callback func(response string) error) error {
	//Search context
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: message,
	})
	if err != nil {
		return err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		ctxVectors, err := ad.Agent.milvusCli.SearchVector(ctx, ad.config.MilvusCollection, embeddingResponse.Embeddings[0])
		if err != nil {
			return err
		}
		if len(ctxVectors) > 0 {
			ad.AddSystemMemory("Context: \n"+strings.Join(ctxVectors, "\n"), nil)
		}
	}

	//Generate response
	ad.AddUserMemory(message, images)
	return ad.talkToOllamaWithMemory(ctx, callback)
}

func (ad *AgentDouble) Think(ctx context.Context, callback func(output any) error) error {
	ad.AddAssistantMemory("Let me think and output something", nil)
	return ad.talkToOllamaWithMemory(ctx, func(response string) error {
		return callback(response)
	})
}

func (ad *AgentDouble) Learn(info string) *AgentDouble {
	return ad.AddSystemMemory(info, nil)
}

func (ad *AgentDouble) Read(url string) error {
	//todo
	return nil
}

func (ad *AgentDouble) Write() error {
	//todo write something to milvus
	return nil
}

func (ad *AgentDouble) Forget(number int) *AgentDouble {
	memoryLen := len(ad.memory.Contexts)
	if number < 0 {
		number = memoryLen
	}
	if number > memoryLen {
		number = memoryLen
	}

	leftMemoryLen := memoryLen - number

	if leftMemoryLen <= 0 {
		ad.memory.Contexts = nil
		return nil
	}

	ad.memory.Contexts = ad.memory.Contexts[:leftMemoryLen]
	return ad
}

func (ad *AgentDouble) ResetMemory() *AgentDouble {
	return ad.Forget(-1).InitMemory()
}

func (ad *AgentDouble) MemorySnapshot() *Memory {
	memorySnapshot := &Memory{}
	for _, memoryCtx := range ad.memory.Contexts {
		memorySnapshot.Contexts = append(memorySnapshot.Contexts, &MemoryCtx{
			Role:    memoryCtx.Role,
			Content: memoryCtx.Content,
			Images:  append(make([]string, 0, len(memoryCtx.Images)), memoryCtx.Images...),
			Epoch:   memoryCtx.Epoch,
		})
	}
	return memorySnapshot
}

func (ad *AgentDouble) LoadMemory(snapshot *Memory) *AgentDouble {
	newMemory := &Memory{}
	for _, memoryCtx := range snapshot.Contexts {
		newMemory.Contexts = append(newMemory.Contexts, &MemoryCtx{
			Role:    memoryCtx.Role,
			Content: memoryCtx.Content,
			Images:  append(make([]string, 0, len(memoryCtx.Images)), memoryCtx.Images...),
			Epoch:   memoryCtx.Epoch,
		})
	}
	ad.memory = newMemory
	return ad
}
