package directory

import (
	"errors"
	"os"
)

type Writer struct {
	RootDir string
}

func (w *Writer) GetDescription() string {
	//todo
	return ""
}

func (w *Writer) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	params, isValidParams := cmdCtx.(map[string]interface{})
	if !isValidParams {
		return errors.New("error converting params for filesystem/directory/writer skill")
	}

	path, hasPath := params["path"]
	if !hasPath {
		return errors.New("not found path from params")
	}
	pathStr, isValidPath := path.(string)
	if !isValidPath {
		return errors.New("error converting path from params")
	}

	return os.MkdirAll(w.RootDir+"/"+pathStr, os.ModeDir)
}
