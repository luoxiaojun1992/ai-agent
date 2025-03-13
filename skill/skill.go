package skill

type Skill interface {
	GetDescription() string
	Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error
}
