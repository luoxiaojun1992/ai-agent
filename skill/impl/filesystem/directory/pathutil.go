package directory

import (
	"errors"
	"path/filepath"
	"strings"
)

func resolvePathWithinRoot(rootDir, pathStr string) (string, error) {
	if strings.TrimSpace(pathStr) == "" {
		return "", errors.New("path cannot be empty")
	}
	if rootDir == "" {
		rootDir = "."
	}

	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(pathStr)
	fullPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		fullPath = filepath.Join(rootAbs, cleanPath)
	}

	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(rootAbs, fullAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes root directory")
	}

	return fullAbs, nil
}
