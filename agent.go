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

type AgentOption struct {
	config    *Config
	ollamaCli ollama.IClient
	milvusCli milvus.IClient
	character string
	role      string
	skillSet  map[string]skill.Skill
}

func (ao *AgentOption) SetConfig(config *Config) *AgentOption {
	ao.config = config
	return ao
}

func (ao *AgentOption) SetOllamaCli(ollamaCli ollama.IClient) *AgentOption {
	ao.ollamaCli = ollamaCli
	return ao
}

func (ao *AgentOption) SetMilvusCli(milvusCli milvus.IClient) *AgentOption {
	ao.milvusCli = milvusCli
	return ao
}

func (ao *AgentOption) SetCharacter(character string) *AgentOption {
	ao.character = character
	return ao
}

func (ao *AgentOption) SetRole(role string) *AgentOption {
	ao.role = role
	return ao
}

func (ao *AgentOption) AddSkill(name string, processor skill.Skill) *AgentOption {
	ao.skillSet[name] = processor
	return ao
}

type Agent struct {
	//todo search engine client
	config *Config

	personalInfo *personalInfo
	skillSet     map[string]skill.Skill

	ollamaCli ollama.IClient
	milvusCli milvus.IClient
}

func NewAgent(ctx context.Context, optionFuncs ...func(option *AgentOption)) (*Agent, error) {
	option := &AgentOption{
		skillSet: make(map[string]skill.Skill),
	}
	for _, optionFunc := range optionFuncs {
		optionFunc(option)
	}
	if option.config == nil {
		return nil, errors.New("invalid agent config")
	}
	if option.ollamaCli == nil {
		option.SetOllamaCli(ollama.NewClient(&ollama.Config{
			Host: option.config.OllamaHost,
		}))
	}
	if option.milvusCli == nil {
		milvusCli, err := milvus.NewClient(ctx, &milvus.Config{
			Host: option.config.MilvusHost,
		})
		if err != nil {
			return nil, err
		}
		option.SetMilvusCli(milvusCli)
	}

	return &Agent{
		config: option.config,
		personalInfo: &personalInfo{
			character: option.character,
			role:      option.role,
		},
		skillSet:  option.skillSet,
		ollamaCli: option.ollamaCli,
		milvusCli: option.milvusCli,
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
  "context": {
    "parameter1": "value1",
    "parameter2": "value2"
  },
  "abort_on_error": true
}
</tool>
The value of context might be JSON object or other data structure according to the payload definition of specific function.
For example, if you need to call a 'search' function to look up information about the weather in New York, you should include this JSON in your response:
<tool>
{
  "function": "search",
  "context": {
    "query\": "weather in New York"
  },
  "abort_on_error": true
}
</tool>
Here is a list of supported functions (might also be called as skill or tool) you can call when needed:
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

type AgentDoubleOption struct {
	config     *Config
	agent      *Agent
	checkpoint Checkpoint
}

func (ado *AgentDoubleOption) SetConfig(config *Config) *AgentDoubleOption {
	ado.config = config
	return ado
}

func (ado *AgentDoubleOption) SetAgent(agent *Agent) *AgentDoubleOption {
	ado.agent = agent
	return ado
}

func (ado *AgentDoubleOption) SetCheckpoint(checkpoint Checkpoint) *AgentDoubleOption {
	ado.checkpoint = checkpoint
	return ado
}

type AgentDouble struct {
	config *Config

	Agent      *Agent
	memory     *Memory
	checkpoint Checkpoint
}

func NewAgentDouble(ctx context.Context, optionFuncs ...func(option *AgentDoubleOption)) (*AgentDouble, error) {
	doubleOption := &AgentDoubleOption{}
	for _, optionFunc := range optionFuncs {
		optionFunc(doubleOption)
	}
	if doubleOption.config == nil {
		return nil, errors.New("invalid agent double config")
	}
	if doubleOption.agent == nil {
		agent, err := NewAgent(ctx, func(option *AgentOption) {
			option.SetConfig(doubleOption.config)
		})
		if err != nil {
			return nil, err
		}
		doubleOption.SetAgent(agent)
	}

	return &AgentDouble{
		config:     doubleOption.config,
		Agent:      doubleOption.agent,
		memory:     NewMemory(),
		checkpoint: doubleOption.checkpoint,
	}, nil
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
	//todo add prompt for milvus collection name for milvus skill
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
		if err := ad.Agent.Command(ctx, functionCall.Function, functionCall.Context, func(output any) (any, error) {
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

	if ad.config.AgentMode == AgentModeLoop && !prompt.ParseLoopEnd(responseContentStr) {
		time.Sleep(ad.config.AgentLoopDuration)
		return ad.talkToOllamaWithMemory(ctx, callback)
	}

	return nil
}

func (ad *AgentDouble) ListenAndWatch(ctx context.Context, message string, images []string, callback func(response string) error) error {
	//Search context
	ctxVectors, err := ad.Recall(ctx, message)
	if err != nil {
		return err
	}
	if len(ctxVectors) > 0 {
		ad.AddSystemMemory("Context: \n"+strings.Join(ctxVectors, "\n"), nil)
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

func (ad *AgentDouble) Remember(ctx context.Context, info string) error {
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: info,
	})
	if err != nil {
		return err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		return ad.Agent.milvusCli.InsertVector(ctx, ad.config.MilvusCollection, info, embeddingResponse.Embeddings[0])
	}
	return nil
}

func (ad *AgentDouble) Recall(ctx context.Context, prompt string) ([]string, error) {
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: prompt,
	})
	if err != nil {
		return nil, err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		ctxVectors, err := ad.Agent.milvusCli.SearchVector(ctx, ad.config.MilvusCollection, embeddingResponse.Embeddings[0])
		if err != nil {
			return nil, err
		}
		if len(ctxVectors) > 0 {
			return ctxVectors, nil
		}
	}
	return nil, nil
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
