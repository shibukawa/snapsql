package schemaimport

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

func TestResolveConfigLoadsTblsConfigAndDerivesSchemaPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	yaml := "dsn: postgres://user:pass@localhost:5432/app\ndocPath: dbdoc\n"
	configPath := writeFile(t, tmp, ".tbls.yml", yaml)

	opts := Options{
		WorkingDir: tmp,
		OutputDir:  "schema-out",
	}

	cfg, err := ResolveConfig(context.Background(), opts)
	if err != nil {
		t.Fatalf("ResolveConfig returned error: %v", err)
	}

	if cfg.TblsConfigPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, cfg.TblsConfigPath)
	}

	expectedDoc := filepath.Join(tmp, "dbdoc")

	expectedJSON := filepath.Join(expectedDoc, "schema.json")
	if cfg.SchemaJSONPath != expectedJSON {
		t.Fatalf("expected schema JSON %q, got %q", expectedJSON, cfg.SchemaJSONPath)
	}

	if cfg.DocPath != expectedDoc {
		t.Fatalf("expected doc path %q, got %q", expectedDoc, cfg.DocPath)
	}

	if cfg.OutputDir != filepath.Join(tmp, "schema-out") {
		t.Fatalf("expected output dir to be absolutised, got %q", cfg.OutputDir)
	}

	if cfg.TblsConfig == nil {
		t.Fatalf("expected tbls config to be loaded")
	}

	if gotDSN := cfg.TblsConfig.DSN.URL; gotDSN != "postgres://user:pass@localhost:5432/app" {
		t.Fatalf("unexpected DSN %q", gotDSN)
	}
}

func TestResolveConfigHonoursExplicitPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	configPath := writeFile(t, tmp, "tbls.yml", "docPath: docs\n")

	opts := Options{
		WorkingDir:     tmp,
		TblsConfigPath: "tbls.yml",
		SchemaJSONPath: "fixtures/schema.json",
	}

	cfg, err := ResolveConfig(context.Background(), opts)
	if err != nil {
		t.Fatalf("ResolveConfig returned error: %v", err)
	}

	if cfg.TblsConfigPath != configPath {
		t.Fatalf("expected resolved config path %q, got %q", configPath, cfg.TblsConfigPath)
	}

	expectedDoc := filepath.Join(tmp, "fixtures")

	expectedJSON := filepath.Join(expectedDoc, "schema.json")
	if cfg.SchemaJSONPath != expectedJSON {
		t.Fatalf("expected schema JSON %q, got %q", expectedJSON, cfg.SchemaJSONPath)
	}

	if cfg.DocPath != expectedDoc {
		t.Fatalf("expected doc path %q, got %q", expectedDoc, cfg.DocPath)
	}
}

func TestResolveConfigMissingConfigReturnsError(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	_, err := ResolveConfig(context.Background(), Options{WorkingDir: tmp})
	if err == nil {
		t.Fatalf("expected error when config is missing")
	}
}

func TestResolveConfigExplicitAbsolutePaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	configPath := writeFile(t, tmp, "tbls.yml", "docPath: docs\n")

	opts := Options{
		WorkingDir:     tmp,
		TblsConfigPath: configPath,
		SchemaJSONPath: filepath.Join(tmp, "schema.json"),
	}

	cfg, err := ResolveConfig(context.Background(), opts)
	if err != nil {
		t.Fatalf("ResolveConfig returned error: %v", err)
	}

	if runtime.GOOS == "windows" {
		// Windows path casing is normalised by filepath.Clean; skip strict equality.
		if filepath.Clean(cfg.TblsConfigPath) != filepath.Clean(configPath) {
			t.Fatalf("expected config path %q, got %q", configPath, cfg.TblsConfigPath)
		}
	} else if cfg.TblsConfigPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, cfg.TblsConfigPath)
	}

	if cfg.SchemaJSONPath != filepath.Join(tmp, "schema.json") {
		t.Fatalf("expected schema path to remain absolute")
	}
}
