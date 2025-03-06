package ai_agent

type skill interface {
	do(callback func(ctx interface{}) error) error
}
