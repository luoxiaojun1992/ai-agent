package ai_agent

type Checkpoint interface {
	Do(snapshot *Memory) error
}
