package ai_agent

type skill interface {
	getDescription() string
	do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error
}
