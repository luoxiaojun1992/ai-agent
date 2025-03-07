package ai_agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
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
	skillSet     map[string]skill

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
		skillSet:     make(map[string]skill),
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

func (sa *Agent) LearnSkill(name string, processor skill) *Agent {
	sa.skillSet[name] = processor
	return sa
}

func (sa *Agent) Command(skillName string, cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	processor, existed := sa.skillSet[skillName]
	if !existed {
		return errors.New("skill hasn't been learned")
	}
	return processor.do(cmdCtx, callback)
}

type memoryCtx struct {
	role    string
	content string
	epoch   int
}

func (mc *memoryCtx) toOllamaMessage() *ollama.Message {
	return &ollama.Message{
		Role:    mc.role,
		Content: fmt.Sprintf("[Message has been recalled %d times] ", mc.epoch) + mc.content,
	}
}

type memory struct {
	contexts []*memoryCtx
}

func NewMemory() *memory {
	return &memory{}
}

type AgentDouble struct {
	config *Config

	Agent  *Agent
	memory *memory
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

func (ad *AgentDouble) AddMemory(role, content string) error {
	ad.memory.contexts = append(ad.memory.contexts, &memoryCtx{
		role:    role,
		content: content,
	})
	return nil
}

func (ad *AgentDouble) InitMemory() *AgentDouble {
	personalInfoPrompt := ad.Agent.personalInfo.prompt()
	ad.AddMemory("system", personalInfoPrompt)
	return ad
}

func (ad *AgentDouble) talkToOllama(callback func(response string) error) error {
	var responseContent strings.Builder
	ollamaMessages := make([]*ollama.Message, 0, len(ad.memory.contexts))
	for _, memCtx := range ad.memory.contexts {
		memCtx.epoch++
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

	ad.AddMemory("assistant", responseContent.String())

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
			ad.AddMemory("system", "Context: \n"+strings.Join(ctxVectors, "\n"))
		}
	}

	//Generate response
	ad.AddMemory("user", message)
	return ad.talkToOllama(callback)
}

func (ad *AgentDouble) Think(callback func(output interface{}) error) error {
	ad.AddMemory("assistant", "Let me think and output something")
	return ad.talkToOllama(func(response string) error {
		return callback(response)
	})
}

func (ad *AgentDouble) Learn(info string) error {
	ad.AddMemory("system", info)
	return nil
}

func (ad *AgentDouble) Read(url string) error {
	//todo
	return nil
}

func (ad *AgentDouble) Forget(number int) error {
	memoryLen := len(ad.memory.contexts)
	if number < 0 {
		number = memoryLen
	}
	if number > memoryLen {
		number = memoryLen
	}

	leftMemoryLen := memoryLen - number

	if leftMemoryLen <= 0 {
		ad.memory.contexts = nil
		return nil
	}

	ad.memory.contexts = ad.memory.contexts[:leftMemoryLen]
	return nil
}
