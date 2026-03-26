package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(rootDir, pathStr string) (string, error) {
	if strings.TrimSpace(rootDir) == "" {
		return "", errors.New("root dir is required")
	}
	if strings.TrimSpace(pathStr) == "" {
		return "", errors.New("path cannot be empty")
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("resolve root dir: %w", err)
	}

	cleanPath := filepath.Clean(pathStr)
	fullPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		fullPath = filepath.Join(absRoot, cleanPath)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes root dir")
	}

	return absPath, nil
}
