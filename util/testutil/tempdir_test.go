package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateRepoScopedTempDir(t *testing.T) {
	dir := CreateRepoScopedTempDir(t, "ut-")
	if dir == "" {
		t.Fatalf("expected temp dir path")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected temp dir to exist: %v", err)
	}
	if filepath.Base(filepath.Dir(dir)) != ".tmp-testdata" {
		t.Fatalf("expected parent temp base to be .tmp-testdata, got %s", filepath.Base(filepath.Dir(dir)))
	}
}
