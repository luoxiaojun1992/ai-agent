package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luoxiaojun1992/ai-agent/util/testutil"
)

func TestWriterReaderAndRemover_Do(t *testing.T) {
	root := testutil.CreateRepoScopedTempDir(t, "file-skill-test-")

	writer := &Writer{RootDir: root}
	reader := &Reader{RootDir: root}
	remover := &Remover{RootDir: root}

	relPath := filepath.Join("a", "b.txt")
	fullPath := filepath.Join(root, relPath)

	if err := writer.Do(context.Background(), map[string]any{
		"path":    relPath,
		"content": "hello",
	}, nil); err != nil {
		t.Fatalf("writer failed: %v", err)
	}

	var got []byte
	err := reader.Do(context.Background(), map[string]any{
		"path": fullPath,
	}, func(output any) (any, error) {
		content, ok := output.([]byte)
		if !ok {
			t.Fatalf("expected []byte callback output, got %T", output)
		}
		got = content
		return nil, nil
	})
	if err != nil {
		t.Fatalf("reader failed: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected file content: %s", string(got))
	}

	if err := remover.Do(context.Background(), map[string]any{
		"path": fullPath,
	}, nil); err != nil {
		t.Fatalf("remover failed: %v", err)
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed")
	}
}

func TestRemover_Do_RejectDangerousPath(t *testing.T) {
	remover := &Remover{}
	err := remover.Do(context.Background(), map[string]any{
		"path": "/",
	}, nil)
	if err == nil {
		t.Fatalf("expected dangerous path to be rejected")
	}
}

func TestReader_Do_InvalidParams(t *testing.T) {
	reader := &Reader{}
	if err := reader.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestFileSkill_Descriptions(t *testing.T) {
	r := &Reader{}
	w := &Writer{}
	rm := &Remover{}
	rDesc, err := r.GetDescription()
	if err != nil || rDesc == "" || r.ShortDescription() == "" {
		t.Fatalf("reader descriptions should not be empty")
	}
	wDesc, err := w.GetDescription()
	if err != nil || wDesc == "" || w.ShortDescription() == "" {
		t.Fatalf("writer descriptions should not be empty")
	}
	rmDesc, err := rm.GetDescription()
	if err != nil || rmDesc == "" || rm.ShortDescription() == "" {
		t.Fatalf("remover descriptions should not be empty")
	}
	if !strings.Contains(rmDesc, "Warning") {
		t.Fatalf("expected warning in remover description")
	}
}

func TestWriter_Do_InvalidContentType(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	err := w.Do(context.Background(), map[string]any{"path": "x", "content": 1}, nil)
	if err == nil {
		t.Fatalf("expected invalid content type error")
	}
}
