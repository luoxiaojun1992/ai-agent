package file

import (
	"context"
	"errors"
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

func TestValidateRemovePath_RejectDangerousPath(t *testing.T) {
	err := validateRemovePath("/")
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

func TestReader_Do_MissingPath(t *testing.T) {
	r := &Reader{}
	if err := r.Do(context.Background(), map[string]any{}, nil); err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestReader_Do_PathNotString(t *testing.T) {
	r := &Reader{}
	if err := r.Do(context.Background(), map[string]any{"path": 1}, nil); err == nil {
		t.Fatalf("expected path type error")
	}
}

func TestReader_Do_ReadFileError(t *testing.T) {
	r := &Reader{}
	err := r.Do(context.Background(), map[string]any{"path": filepath.Join(t.TempDir(), "missing.txt")}, func(any) (any, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected read error for nonexistent file")
	}
}

func TestReader_Do_CallbackError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	expected := errors.New("cb error")
	r := &Reader{}
	err := r.Do(context.Background(), map[string]any{"path": path}, func(any) (any, error) { return nil, expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestWriter_Do_MissingPath(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	if err := w.Do(context.Background(), map[string]any{"content": "x"}, nil); err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestWriter_Do_PathNotString(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	if err := w.Do(context.Background(), map[string]any{"path": 1, "content": "x"}, nil); err == nil {
		t.Fatalf("expected path type error")
	}
}

func TestWriter_Do_InvalidParamsAndMissingContent(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	if err := w.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
	if err := w.Do(context.Background(), map[string]any{"path": "x"}, nil); err == nil {
		t.Fatalf("expected missing content error")
	}
}

func TestRemover_Do_InvalidParams(t *testing.T) {
	r := &Remover{}
	if err := r.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestRemover_Do_PathNotString(t *testing.T) {
	r := &Remover{}
	if err := r.Do(context.Background(), map[string]any{"path": 1}, nil); err == nil {
		t.Fatalf("expected path type error")
	}
}

func TestRemover_Do_EmptyPath(t *testing.T) {
	r := &Remover{}
	if err := r.Do(context.Background(), map[string]any{"path": ""}, nil); err == nil {
		t.Fatalf("expected empty path error")
	}
}

func TestValidateRemovePath_CurrentDir(t *testing.T) {
	err := validateRemovePath(".")
	if err == nil {
		t.Fatalf("expected current directory rejection error")
	}
	if !strings.Contains(err.Error(), "current directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRemovePath_SystemDirPrefix(t *testing.T) {
	err := validateRemovePath("/dev/null")
	if err == nil {
		t.Fatalf("expected system directory rejection error")
	}
	if !strings.Contains(err.Error(), "system") {
		t.Fatalf("unexpected error: %v", err)
	}
}
