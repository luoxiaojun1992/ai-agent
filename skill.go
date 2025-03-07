package ai_agent

type skill interface {
	do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error
}
