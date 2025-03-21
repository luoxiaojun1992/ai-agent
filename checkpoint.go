package ai_agent

type Checkpoint interface {
	Do(agentDouble *AgentDouble) error
}
