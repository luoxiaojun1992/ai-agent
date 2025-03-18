package file

import (
	"errors"
	"os"
)

type Writer struct {
	RootDir string
}

func (w *Writer) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	params, isValidParams := cmdCtx.(map[string]interface{})
	if !isValidParams {
		return errors.New("error converting params for filesystem/file/writer skill")
	}

	path, hasPath := params["path"]
	if !hasPath {
		return errors.New("not found path from params")
	}
	pathStr, isValidPath := path.(string)
	if !isValidPath {
		return errors.New("error converting path from params")
	}

	content, hasContent := params["content"]
	if !hasContent {
		return errors.New("not found content from params")
	}
	contentStr, isValidContent := content.(string)
	if !isValidContent {
		return errors.New("error converting content from params")
	}

	return os.WriteFile(w.RootDir+"/"+pathStr, []byte(contentStr), os.ModeAppend)
}
