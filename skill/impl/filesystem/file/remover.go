package file

import (
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

func (r *Remover) Do(cmdCtx interface{}, callback func(output interface{}) (interface{}, error)) error {
	params, isValidParams := cmdCtx.(map[string]interface{})
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
