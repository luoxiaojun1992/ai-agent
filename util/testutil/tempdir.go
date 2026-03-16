package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func CreateRepoScopedTempDir(t *testing.T, prefix string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller path")
	}

	dir := filepath.Dir(currentFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to locate repository root")
		}
		dir = parent
	}

	base := filepath.Join(dir, ".tmp-testdata")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("failed to create temp data base directory: %v", err)
	}

	tempDir, err := os.MkdirTemp(base, prefix)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	return tempDir
}
