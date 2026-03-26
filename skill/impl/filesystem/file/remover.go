package file

import (
	"context"
	"errors"
	"os"
	"strings"
)

type Remover struct {
	RootDir string
}

func (r *Remover) GetDescription() (string, error) {
	return `Remove a file or directory at the specified path. This skill permanently deletes files or directories recursively.
Parameters:
- path: string - The path to the file or directory to be removed
Returns: Success status
Warning: This operation is irreversible and will delete all contents recursively`, nil
}

func (r *Remover) ShortDescription() string {
	return "Remove file or directory from disk"
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

	fullPath, err := resolvePath(r.RootDir, pathStr)
	if err != nil {
		return err
	}

	if err := validateRemovePath(fullPath); err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}


func validateRemovePath(pathStr string) error {
	// Security check: ensure we're not deleting system directories
	if pathStr == "/" || pathStr == "" || pathStr == "." {
		return errors.New("cannot delete root directory, current directory, or empty path")
	}

	// Check if trying to delete system directories
	systemDirs := []string{"/bin", "/etc", "/usr", "/var", "/sys", "/proc", "/dev"}
	for _, sysDir := range systemDirs {
		if pathStr == sysDir || strings.HasPrefix(pathStr, sysDir+string(os.PathSeparator)) {
			return errors.New("cannot delete system directories")
		}
	}

	return nil
}
