package ai_agent

import (
	"errors"

	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
)

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

	personalInfo *PersonalInfo
	skillSet     map[string]skill

	ollamaCli ollama.IClient
	milvusCli milvus.IClient
}

func NewAgent() *Agent {
	return &Agent{
		personalInfo: &PersonalInfo{},
		skillSet:     make(map[string]skill),
	}
}

func (sa *Agent) SetCharacter(character string) *Agent {
	sa.personalInfo.character = character
	return sa
}

func (sa *Agent) SetRole(role string) *Agent {
	sa.personalInfo.role = role
	return sa
}

func (sa *Agent) Chat(message string) (string, error) {
	//todo
	return "", nil
}

func (sa *Agent) Think(callback func(output interface{}) error) error {
	//todo
	return nil
}

func (sa *Agent) LearnInfo(info string) error {
	//todo
	return nil
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
	Agent  *Agent
	Memory *Memory
}

func NewAgentDouble() *AgentDouble {
	return &AgentDouble{
		Agent:  NewAgent(),
		Memory: NewMemory(),
	}
}

func (ad *AgentDouble) initMemory() *AgentDouble {
	personalInfoPromt := ad.Agent.personalInfo.prompt()
	ad.Memory.contexts = append(ad.Memory.contexts, &ollama.Message{
		Role:    "system",
		Content: personalInfoPromt,
	})
	return ad
}
