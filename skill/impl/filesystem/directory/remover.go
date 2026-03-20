package directory

import (
	"context"
	"errors"
	"os"
	"runtime"

	"github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/pathutil"
)

type Remover struct {
	RootDir string
}

func (r *Remover) GetDescription() (string, error) {
	return `Remove a directory and all its contents recursively. This skill permanently deletes directories and all files within them.
Parameters:
- path: string - The path to the directory to be removed
Returns: Success status
Warning: This operation is irreversible and will delete all contents recursively`, nil
}

func (r *Remover) ShortDescription() string {
	return "Remove directory and all contents"
}

func (r *Remover) Do(_ context.Context, cmdCtx any, _ func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for filesystem/directory/remover skill")
	}

	path, hasPath := params["path"]
	if !hasPath {
		return errors.New("not found path from params")
	}
	pathStr, isValidPath := path.(string)
	if !isValidPath {
		return errors.New("error converting path from params")
	}

	fullPath, err := pathutil.ResolvePathWithinRoot(r.RootDir, pathStr)
	if err != nil {
		return err
	}
	if err := validateRemovePath(fullPath); err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}

func validateRemovePath(absPath string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return pathutil.ValidateNotSystemPath(absPath)
}
