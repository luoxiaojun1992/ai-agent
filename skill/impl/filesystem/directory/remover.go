package directory

import (
	"context"
	"errors"
	"os"
)

type Remover struct {
	RootDir string
}

func (r *Remover) GetDescription() string {
	//todo
	return ""
}

func (r *Remover) Do(_ context.Context, cmdCtx any, _ func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for filesystem/file/remover skill")
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
	return os.RemoveAll(pathStr)
}
