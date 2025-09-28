package schemaimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRuntimeSuccess(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	tblsPath := filepath.Join(tmp, ".tbls.yml")
	schemaPath := filepath.Join(tmp, "doc", "schema.json")

	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tblsContent := "dsn: postgres://localhost/app\ndocPath: doc\n"
	if err := os.WriteFile(tblsPath, []byte(tblsContent), 0o644); err != nil {
		t.Fatalf("write tbls: %v", err)
	}

	schemaJSON := `{"driver":{"name":"postgres"},"tables":[{"name":"public.users","type":"TABLE","columns":[{"name":"id","type":"int"}]}]}`
	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	runtime, err := LoadRuntime(ctx, Options{WorkingDir: tmp})
	if err != nil {
		t.Fatalf("LoadRuntime returned error: %v", err)
	}

	if runtime.Config.TblsConfig == nil {
		t.Fatalf("expected tbls config to be loaded")
	}

	if runtime.Config.TblsConfigPath != tblsPath {
		t.Fatalf("unexpected tbls path: %s", runtime.Config.TblsConfigPath)
	}

	if len(runtime.Schemas) != 1 {
		t.Fatalf("expected one schema, got %d", len(runtime.Schemas))
	}

	tables := runtime.TablesByName()
	if tables["users"] == nil {
		t.Fatalf("expected users table in lookup")
	}

	if tables["public.users"] == nil {
		t.Fatalf("expected schema-qualified users key")
	}
}

func TestLoadRuntimePropagatesErrors(t *testing.T) {
	ctx := context.Background()

	if _, err := LoadRuntime(ctx, Options{}); err == nil {
		t.Fatalf("expected error when tbls config missing")
	}
}
