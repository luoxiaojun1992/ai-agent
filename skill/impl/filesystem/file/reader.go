package file

import (
	"context"
	"errors"
	"os"
)

type Reader struct {
	RootDir string
}

func (r *Reader) GetDescription() string {
	//todo
	return ""
}

func (r *Reader) Do(_ context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for filesystem/file/reader skill")
	}

	path, hasPath := params["path"]
	if !hasPath {
		return errors.New("not found path from params")
	}
	pathStr, isValidPath := path.(string)
	if !isValidPath {
		return errors.New("error converting path from params")
	}

	content, err := os.ReadFile(pathStr)
	if err != nil {
		return err
	}
	_, err = callback(content)
	return err
}
