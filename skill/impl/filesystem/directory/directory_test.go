package directory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestValidateRemovePath_RejectDangerousPath(t *testing.T) {
	err := validateRemovePath("/")
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

func TestDirectorySkill_Descriptions(t *testing.T) {
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

func TestReader_Do_InvalidPathType(t *testing.T) {
	r := &Reader{}
	err := r.Do(context.Background(), map[string]any{"path": 1}, nil)
	if err == nil {
		t.Fatalf("expected invalid path type error")
	}
}

func TestReader_Do_MissingPath(t *testing.T) {
	r := &Reader{}
	if err := r.Do(context.Background(), map[string]any{}, nil); err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestReader_Do_ReadDirError(t *testing.T) {
	r := &Reader{}
	err := r.Do(context.Background(), map[string]any{"path": filepath.Join(t.TempDir(), "missing-dir")}, func(any) (any, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected ReadDir error for nonexistent directory")
	}
}

func TestReader_Do_CallbackError(t *testing.T) {
	dir := t.TempDir()
	expected := errors.New("cb error")
	r := &Reader{RootDir: dir}
	err := r.Do(context.Background(), map[string]any{"path": dir}, func(any) (any, error) { return nil, expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestWriter_Do_MissingPath(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	if err := w.Do(context.Background(), map[string]any{}, nil); err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestWriter_Do_PathNotString(t *testing.T) {
	w := &Writer{RootDir: t.TempDir()}
	if err := w.Do(context.Background(), map[string]any{"path": 1}, nil); err == nil {
		t.Fatalf("expected path type error")
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
	if runtime.GOOS == "windows" {
		t.Skip("system dir prefix check uses Unix-style protected dirs")
	}

	err := validateRemovePath("/dev")
	if err == nil {
		t.Fatalf("expected system directory rejection error")
	}
	if !strings.Contains(err.Error(), "system") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePath_RejectsTraversalOutsideRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := resolvePath(root, "../escape-dir"); err == nil {
		t.Fatalf("expected traversal to be rejected")
	}
}
