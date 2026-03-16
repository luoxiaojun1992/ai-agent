package directory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/luoxiaojun1992/ai-agent/util/testutil"
)

func TestWriterReaderAndRemover_Do(t *testing.T) {
	root := testutil.CreateRepoScopedTempDir(t, "dir-skill-test-")

	writer := &Writer{RootDir: root}
	reader := &Reader{RootDir: root}
	remover := &Remover{RootDir: root}

	relPath := filepath.Join("x", "y")
	fullPath := filepath.Join(root, relPath)

	if err := writer.Do(context.Background(), map[string]any{"path": relPath}, nil); err != nil {
		t.Fatalf("writer failed: %v", err)
	}

	testFile := filepath.Join(fullPath, "f.txt")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		t.Fatalf("setup file failed: %v", err)
	}

	count := 0
	err := reader.Do(context.Background(), map[string]any{"path": fullPath}, func(output any) (any, error) {
		entries, ok := output.([]os.DirEntry)
		if !ok {
			t.Fatalf("expected []os.DirEntry callback output, got %T", output)
		}
		count = len(entries)
		return nil, nil
	})
	if err != nil {
		t.Fatalf("reader failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected directory entries")
	}

	if err := remover.Do(context.Background(), map[string]any{"path": fullPath}, nil); err != nil {
		t.Fatalf("remover failed: %v", err)
	}
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be removed")
	}
}

func TestRemover_Do_RejectDangerousPath(t *testing.T) {
	remover := &Remover{}
	err := remover.Do(context.Background(), map[string]any{"path": "/"}, nil)
	if err == nil {
		t.Fatalf("expected dangerous path to be rejected")
	}
}

func TestWriter_Do_InvalidParams(t *testing.T) {
	writer := &Writer{}
	if err := writer.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}
