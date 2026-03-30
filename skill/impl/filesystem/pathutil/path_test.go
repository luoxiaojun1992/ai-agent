package pathutil

import (
	"path/filepath"
	"testing"
)

func TestResolvePath_ValidationErrors(t *testing.T) {
	t.Parallel()

	if _, err := ResolvePath("   ", "a.txt"); err == nil {
		t.Fatalf("expected error when root dir is empty")
	}

	if _, err := ResolvePath(t.TempDir(), "   "); err == nil {
		t.Fatalf("expected error when path is empty")
	}
}

func TestResolvePath_ResolvesPathsWithinRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	expected := filepath.Join(root, "a", "b.txt")

	got, err := ResolvePath(root, filepath.Join("a", "b.txt"))
	if err != nil {
		t.Fatalf("expected relative path to resolve, got error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected resolved path: got %q, want %q", got, expected)
	}

	got, err = ResolvePath(root, filepath.Join("a", "..", "b.txt"))
	if err != nil {
		t.Fatalf("expected cleaned relative path to resolve, got error: %v", err)
	}
	if got != filepath.Join(root, "b.txt") {
		t.Fatalf("unexpected cleaned path: got %q, want %q", got, filepath.Join(root, "b.txt"))
	}
}

func TestResolvePath_AbsolutePathHandling(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	inside := filepath.Join(root, "inside.txt")
	got, err := ResolvePath(root, inside)
	if err != nil {
		t.Fatalf("expected absolute path within root to resolve, got error: %v", err)
	}
	if got != inside {
		t.Fatalf("unexpected resolved absolute path: got %q, want %q", got, inside)
	}

	outside := filepath.Join(t.TempDir(), "outside.txt")
	if _, err := ResolvePath(root, outside); err == nil {
		t.Fatalf("expected absolute path outside root to be rejected")
	}
}

func TestResolvePath_RejectsTraversalOutsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := ResolvePath(root, filepath.Join("..", "escape.txt")); err == nil {
		t.Fatalf("expected traversal path to be rejected")
	}
}
