package directory

type Reader struct {
	RootDir string
}

func (r *Reader) GetDescription() string {
	//todo
	return ""
}

func (r *Reader) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	//todo

	return nil
}
