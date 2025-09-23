package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/shibukawa/snapsql"
)

func TestSQLiteEphemeralWorkflow(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()

	schemaFile := filepath.Join(tempDir, "schema.sql")
	if err := os.WriteFile(schemaFile, []byte(`CREATE TABLE users (
        id INTEGER PRIMARY KEY,
        email TEXT NOT NULL
    );`), 0o644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}

	cmd := &TestCmd{
		Schema:       []string{schemaFile},
		SchemaOutput: filepath.Join(tempDir, "schema-out"),
	}

	config := &snapsql.Config{
		Dialect: "sqlite",
		Schema: snapsql.SchemaExtractionConfig{
			IncludeViews:   false,
			IncludeIndexes: true,
			TablePatterns: snapsql.TablePatterns{
				Include: []string{"*"},
			},
		},
	}

	provisioned, err := cmd.provisionSQLiteDatabase(ctx, false)
	if err != nil {
		t.Fatalf("failed to provision sqlite database: %v", err)
	}

	t.Cleanup(func() {
		if cerr := provisioned.Close(ctx); cerr != nil {
			t.Logf("cleanup error: %v", cerr)
		}
	})

	if err := cmd.applySchema(ctx, provisioned.DB, cmd.Schema, false); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	if err := cmd.executeSchemaPull(ctx, config, provisioned, false); err != nil {
		t.Fatalf("schema pull failed: %v", err)
	}

	tableInfo, err := cmd.loadTableInfo(cmd.SchemaOutput, false)
	if err != nil {
		t.Fatalf("failed to load table info: %v", err)
	}

	if _, ok := tableInfo["users"]; !ok {
		t.Fatalf("expected table 'users' in table info map, got keys %v", sortedKeys(tableInfo))
	}
}

func sortedKeys(m map[string]*snapsql.TableInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

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
