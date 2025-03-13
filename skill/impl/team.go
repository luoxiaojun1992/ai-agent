package impl

import ai_agent "github.com/luoxiaojun1992/ai-agent"

type Team struct {
	Members map[string]*ai_agent.AgentDouble
}

func (t *Team) GetDescription() string {
	//todo
	return ""
}

func (t *Team) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	//todo
	return nil
}
