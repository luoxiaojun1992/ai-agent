package main

import (
	"path/filepath"
	"testing"
)

func TestResolveWorkspaceDir(t *testing.T) {
	base := t.TempDir()
	allowedRoot := filepath.Join(base, "workspace-root")
	codeDir := filepath.Join(allowedRoot, "app")

	t.Run("default workspace", func(t *testing.T) {
		got, err := resolveWorkspaceDir(allowedRoot, codeDir, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		want := filepath.Join(allowedRoot, defaultWorkspaceDir)
		if got != want {
			t.Fatalf("unexpected workspace path, got %q want %q", got, want)
		}
	})

	t.Run("relative child workspace", func(t *testing.T) {
		got, err := resolveWorkspaceDir(allowedRoot, codeDir, "project-a")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		want := filepath.Join(allowedRoot, "project-a")
		if got != want {
			t.Fatalf("unexpected workspace path, got %q want %q", got, want)
		}
	})

	t.Run("cannot use root directly", func(t *testing.T) {
		_, err := resolveWorkspaceDir(allowedRoot, codeDir, ".")
		if err == nil {
			t.Fatal("expected error when workspace equals allowed root")
		}
	})

	t.Run("cannot escape allowed root", func(t *testing.T) {
		_, err := resolveWorkspaceDir(allowedRoot, codeDir, "../etc")
		if err == nil {
			t.Fatal("expected error when workspace escapes allowed root")
		}
	})

	t.Run("cannot use code directory", func(t *testing.T) {
		_, err := resolveWorkspaceDir(allowedRoot, codeDir, "app")
		if err == nil {
			t.Fatal("expected error when workspace points to code directory")
		}
	})
}
