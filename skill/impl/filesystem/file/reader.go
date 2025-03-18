package file

type Reader struct {
	RootDir string
}

func (r *Reader) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	//todo

	return nil
}
