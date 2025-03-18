package file

type Writer struct {
	RootDir string
}

func (w *Writer) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	//todo

	return nil
}
