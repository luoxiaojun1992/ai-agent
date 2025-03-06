package ai_agent

import (
	"context"
	"errors"
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

type PersonalInfo struct {
	character string
	role      string
}

func (pi *PersonalInfo) characterPrompt() string {
	return "Personality: \n" + "You are " + pi.character
}

func (pi *PersonalInfo) rolePrompt() string {
	return "Role: \n" + "You are " + pi.role
}

func (pi *PersonalInfo) prompt() string {
	return pi.characterPrompt() + "\n" + pi.rolePrompt()
}

type Agent struct {
	//todo search engine client
	config *Config

	personalInfo *PersonalInfo
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
		personalInfo: &PersonalInfo{},
		skillSet:     make(map[string]skill),
		ollamaCli: ollama.NewClient(&ollama.Config{
			Host: config.OllamaHost,
		}),
		milvusCli: milvusCli,
		//todo init ollama and milvus clis
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

func (sa *Agent) Command(skillName string, callback func(output interface{}) error) error {
	processor, existed := sa.skillSet[skillName]
	if !existed {
		return errors.New("skill hasn't been learned")
	}
	return processor.do(callback)
}

type Memory struct {
	contexts []*ollama.Message
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

func (ad *AgentDouble) InitMemory() *AgentDouble {
	personalInfoPrompt := ad.Agent.personalInfo.prompt()
	ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
		Role:    "system",
		Content: personalInfoPrompt,
	})
	return ad
}

func (ad *AgentDouble) talkToOllama(callback func(response string) error) error {
	var responseContent strings.Builder
	if err := ad.Agent.ollamaCli.Talk(&ollama.ChatRequest{
		Model:    ad.config.ChatModel,
		Messages: ad.memory.contexts,
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

	ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
		Role:    "assistant",
		Content: responseContent.String(),
	})

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
			ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
				Role:    "system",
				Content: "Context: \n" + strings.Join(ctxVectors, "\n"),
			})
		}
	}

	//Generate response
	ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
		Role:    "user",
		Content: message,
	})
	return ad.talkToOllama(callback)
}

func (ad *AgentDouble) Think(callback func(output interface{}) error) error {
	ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
		Role:    "assistant",
		Content: "Let me think and output something",
	})
	return ad.talkToOllama(func(response string) error {
		return callback(response)
	})
}

func (ad *AgentDouble) Learn(info string) error {
	ad.memory.contexts = append(ad.memory.contexts, &ollama.Message{
		Role:    "system",
		Content: info,
	})
	return nil
}

func (ad *AgentDouble) Read(url string) error {
	//todo
	return nil
}
