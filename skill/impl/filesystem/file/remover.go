package file

type Remover struct {
	RootDir string
}

func (r *Remover) GetDescription() string {
	//todo
	return ""
}

func (r *Remover) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	//todo

	return nil
}
