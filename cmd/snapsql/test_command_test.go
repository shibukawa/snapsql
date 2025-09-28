package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTargetPaths(t *testing.T) {
	projectRoot := t.TempDir()

	subDir := filepath.Join(projectRoot, "cases")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	cmd := &TestCmd{Paths: []string{subDir}}

	paths, err := cmd.resolveTargetPaths(projectRoot)
	if err != nil {
		t.Fatalf("resolveTargetPaths returned error: %v", err)
	}

	if len(paths) != 1 || paths[0] != filepath.Clean(subDir) {
		t.Fatalf("unexpected resolved paths: %v", paths)
	}

	cmd = &TestCmd{Paths: []string{"cases"}}

	paths, err = cmd.resolveTargetPaths(projectRoot)
	if err != nil {
		t.Fatalf("resolveTargetPaths relative path error: %v", err)
	}

	if len(paths) != 1 || paths[0] != filepath.Clean(subDir) {
		t.Fatalf("unexpected resolved paths for relative input: %v", paths)
	}

	outside := filepath.Dir(projectRoot)

	cmd = &TestCmd{Paths: []string{outside}}
	if _, err := cmd.resolveTargetPaths(projectRoot); err == nil {
		t.Fatalf("expected error for path outside project root")
	}
}
