package ai_agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
	"github.com/luoxiaojun1992/ai-agent/skill"
	"github.com/luoxiaojun1992/ai-agent/util/prompt"
)

type Config struct {
	ChatModel        string
	EmbeddingModel   string
	OllamaHost       string
	MilvusHost       string
	MilvusCollection string
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

func (sa *Agent) SetCharacter(character string) *Agent {
	sa.personalInfo.character = character
	return sa
}

func (sa *Agent) SetRole(role string) *Agent {
	sa.personalInfo.role = role
	return sa
}

func (sa *Agent) toolPrompt() string {
	functionPromptList := make([]string, 0, len(sa.skillSet))
	for skillName, skill := range sa.skillSet {
		functionPromptList = append(functionPromptList, fmt.Sprintf("%s: %s", skillName, skill.GetDescription()))
	}
	allFunctionPrompt := strings.Join(functionPromptList, "\n")
	return fmt.Sprintf(`
When answering questions, if you need to call external tools or resources, please return the function call in JSON format embedded within your response. The JSON should follow this structure:
<tool>{
  "function": "function_name",
  "parameters": {
    "argument1": "value1",
    "argument2": "value2"
  }
}
</tool>
For example, if you need to call a 'search' function to look up information about the weather in New York, you should include this JSON in your response:
<tool>
{
  "function": "search",
  "parameters": {
    "query\": "weather in New York"
  }
}
</tool>
Here is a list of supported functions you can call when needed:
%s
`, allFunctionPrompt)
}

func (sa *Agent) LearnSkill(name string, processor skill.Skill) *Agent {
	sa.skillSet[name] = processor
	return sa
}

func (sa *Agent) Command(skillName string, cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	processor, existed := sa.skillSet[skillName]
	if !existed {
		return errors.New(fmt.Sprintf("skill [%s] hasn't been learned", skillName))
	}
	return processor.Do(cmdCtx, callback)
}

type MemoryCtx struct {
	Role    string
	Content string
	Epoch   int
}

func (mc *MemoryCtx) toOllamaMessage() *ollama.Message {
	return &ollama.Message{
		Role:    mc.Role,
		Content: fmt.Sprintf("[This message has been recalled %d times] ", mc.Epoch) + mc.Content,
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

	Agent  *Agent
	memory *Memory
}

func NewAgentDouble(ctx context.Context, config *Config) (*AgentDouble, error) {
	agent, err := NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	return &AgentDouble{
		config: config,
		Agent:  agent,
		memory: NewMemory(),
	}, nil
}

func (ad *AgentDouble) AddMemory(role, content string) *AgentDouble {
	ad.memory.Contexts = append(ad.memory.Contexts, &MemoryCtx{
		Role:    role,
		Content: content,
	})
	return ad
}

func (ad *AgentDouble) AddSystemMemory(content string) *AgentDouble {
	return ad.AddMemory("system", content)
}

func (ad *AgentDouble) AddAssistantMemory(content string) *AgentDouble {
	return ad.AddMemory("assistant", content)
}

func (ad *AgentDouble) AddUserMemory(content string) *AgentDouble {
	return ad.AddMemory("user", content)
}

func (ad *AgentDouble) InitMemory() *AgentDouble {
	personalInfoPrompt := ad.Agent.personalInfo.prompt()
	return ad.AddSystemMemory(personalInfoPrompt).
		AddSystemMemory(ad.Agent.toolPrompt())
}

func (ad *AgentDouble) talkToOllama(callback func(response string) error) error {
	var responseContent strings.Builder
	ollamaMessages := make([]*ollama.Message, 0, len(ad.memory.Contexts))
	for _, memCtx := range ad.memory.Contexts {
		memCtx.Epoch++
		ollamaMessages = append(ollamaMessages, memCtx.toOllamaMessage())
	}
	if err := ad.Agent.ollamaCli.Talk(&ollama.ChatRequest{
		Model:    ad.config.ChatModel,
		Messages: ollamaMessages,
	}, func(response string) error {
		if _, err := responseContent.WriteString(response); err != nil {
			return err
		}
		if err := callback(response); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	responseContentStr := responseContent.String()
	ad.AddAssistantMemory(responseContentStr)

	functionCallList, err := prompt.ParseFunctionCalling(responseContentStr)
	if err != nil {
		return err
	}
	for _, functionCall := range functionCallList {
		if err := ad.Agent.Command(functionCall.Function, functionCall.Parameters, func(output interface{}) (interface{}, error) {
			ad.AddSystemMemory(fmt.Sprintf("The result of function [%s]: %v", functionCall.Function, output))
			return nil, nil
		}); err != nil {
			ad.AddSystemMemory(fmt.Sprintf("The error [%s] happened during executing the function [%s].",
				err.Error(),
				functionCall.Function))
			return nil
		}
		ad.AddSystemMemory(
			fmt.Sprintf("The function [%s] has been executed successfully.", functionCall.Function))
	}

	return nil
}

func (ad *AgentDouble) Listen(message string, callback func(response string) error) error {
	//Search context
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: message,
	})
	if err != nil {
		return err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		ctxVectors, err := ad.Agent.milvusCli.SearchVector(ad.config.MilvusCollection, embeddingResponse.Embeddings[0])
		if err != nil {
			return err
		}
		if len(ctxVectors) > 0 {
			ad.AddSystemMemory("Context: \n" + strings.Join(ctxVectors, "\n"))
		}
	}

	//Generate response
	ad.AddUserMemory(message)
	return ad.talkToOllama(callback)
}

func (ad *AgentDouble) Think(callback func(output interface{}) error) error {
	ad.AddAssistantMemory("Let me think and output something")
	return ad.talkToOllama(func(response string) error {
		return callback(response)
	})
}

func (ad *AgentDouble) Learn(info string) *AgentDouble {
	return ad.AddSystemMemory(info)
}

func (ad *AgentDouble) Read(url string) error {
	//todo
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
			Epoch:   memoryCtx.Epoch,
		})
	}
	return memorySnapshot
}
