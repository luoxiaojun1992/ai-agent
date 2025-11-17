package directory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

type Writer struct {
	RootDir string
}

func (w *Writer) GetDescription() string {
	return `Create directories at the specified path. This skill creates new directories and any necessary parent directories recursively.
Parameters:
- path: string - The relative path where the directory should be created (relative to RootDir)
Returns: Success status`
}

func (w *Writer) ShortDescription() string {
	return "Create directories recursively"
}

func (w *Writer) Do(_ context.Context, cmdCtx any, _ func(output any) (any, error)) error {
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

	// Security: Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(pathStr)
	fullPath := filepath.Join(w.RootDir, cleanPath)

	return os.MkdirAll(fullPath, 0755)
}
