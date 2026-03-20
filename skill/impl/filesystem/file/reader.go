package file

import (
	"context"
	"errors"
	"os"

	"github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/pathutil"
)

type Reader struct {
	RootDir string
}

func (r *Reader) GetDescription() (string, error) {
	return `Read file content from specified path. This skill allows reading text files and returns the content as bytes.
Parameters:
- path: string - The full path to the file to be read
Returns: File content as byte array`, nil
}

func (r *Reader) ShortDescription() string {
	return "Read file content from disk"
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

	fullPath, err := pathutil.ResolvePathWithinRoot(r.RootDir, pathStr)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	_, err = callback(content)
	return err
}
