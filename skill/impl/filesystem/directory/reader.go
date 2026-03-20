package directory

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
	return `Read directory contents and list all entries. This skill reads a directory and returns all files and subdirectories within it.
Parameters:
- path: string - The path to the directory to be read
Returns: Array of directory entries (files and subdirectories)`, nil
}

func (r *Reader) ShortDescription() string {
	return "List directory contents"
}

func (r *Reader) Do(_ context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for filesystem/directory/reader skill")
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

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}
	_, err = callback(entries)
	return err
}
