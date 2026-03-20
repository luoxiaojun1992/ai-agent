package pathutil

import (
	"errors"
	"path/filepath"
	"strings"
)

func ResolvePathWithinRoot(rootDir, pathStr string) (string, error) {
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

func ValidateNotSystemPath(absPath string) error {
	protectedRoots := []string{"/bin", "/etc", "/usr", "/var", "/sys", "/proc", "/dev"}
	for _, root := range protectedRoots {
		if isPathWithinBase(root, absPath) {
			return errors.New("cannot delete system directories")
		}
	}
	return nil
}

func isPathWithinBase(basePath, targetPath string) bool {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
