package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/pathutil"
)

type Writer struct {
	RootDir string
}

func (w *Writer) GetDescription() (string, error) {
	return `Write content to a file at the specified path. This skill creates new files or overwrites existing ones with the provided content.
Parameters:
- path: string - The relative path where the file should be written (relative to RootDir)
- content: string - The text content to write to the file
Returns: Success status`, nil
}

func (w *Writer) ShortDescription() string {
	return "Write content to file on disk"
}

func (w *Writer) Do(_ context.Context, cmdCtx any, _ func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
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

	fullPath, err := pathutil.ResolvePathWithinRoot(w.RootDir, pathStr)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(contentStr), 0644)
}
