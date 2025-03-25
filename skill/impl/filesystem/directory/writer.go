package directory

import (
	"context"
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

func (w *Writer) Do(_ context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
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

	//todo test carefully
	return os.MkdirAll(w.RootDir+"/"+pathStr, os.ModeDir)
}
