package ai_agent

type skill interface {
	do(callback func(output interface{}) error) error
}
