package schemaimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tblsconfig "github.com/k1LoW/tbls/config"
	snapsql "github.com/shibukawa/snapsql"
)

func TestNewConfigDefaults(t *testing.T) {
	opts := Options{
		TblsConfigPath: "./db/.tbls.yml",
		SchemaJSONPath: "./db/schema.json",
		OutputDir:      "./schema",
		Include:        []string{"public.*"},
		Exclude:        []string{"internal.*"},
	}

	cfg := NewConfig(opts)

	if cfg.TblsConfigPath != opts.TblsConfigPath {
		t.Fatalf("expected TblsConfigPath %q, got %q", opts.TblsConfigPath, cfg.TblsConfigPath)
	}

	if cfg.SchemaJSONPath != opts.SchemaJSONPath {
		t.Fatalf("expected SchemaJSONPath %q, got %q", opts.SchemaJSONPath, cfg.SchemaJSONPath)
	}

	if cfg.OutputDir != opts.OutputDir {
		t.Fatalf("expected OutputDir %q, got %q", opts.OutputDir, cfg.OutputDir)
	}

	if !cfg.IncludeViews {
		t.Fatalf("expected IncludeViews default true")
	}

	if !cfg.IncludeIndexes {
		t.Fatalf("expected IncludeIndexes default true")
	}

	if !cfg.SchemaAware {
		t.Fatalf("expected SchemaAware default true")
	}

	if &cfg.Include == &opts.Include {
		t.Fatalf("Include slice should be copied, not aliased")
	}

	if &cfg.Exclude == &opts.Exclude {
		t.Fatalf("Exclude slice should be copied, not aliased")
	}
}

func TestNewImporterInitialState(t *testing.T) {
	cfg := NewConfig(Options{TblsConfigPath: "./.tbls.yml", SchemaJSONPath: "./schema.json", OutputDir: "./schema"})

	importer := NewImporter(cfg)
	if importer == nil {
		t.Fatalf("expected importer instance")
	}

	if importer.Config().TblsConfigPath != cfg.TblsConfigPath {
		t.Fatalf("importer config mismatch")
	}

	if importer.hasLoadedSchema() {
		t.Fatalf("schema should not be loaded initially")
	}
}

func TestLoadSchemaJSONAndConvertSuccess(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.json")

	json := `{"driver":{"name":"postgres","database":"app","database_version":"16"},"tables":[{"name":"public.users","type":"TABLE","columns":[{"name":"id","type":"int","pk":true},{"name":"email","type":"text","nullable":false}],"constraints":[{"name":"users_pkey","type":"PRIMARY KEY","columns":["id"]}],"indexes":[{"name":"users_email_idx","def":"CREATE UNIQUE INDEX users_email_idx ON public.users (email)","columns":["email"]}]}]}`
	if err := os.WriteFile(schemaPath, []byte(json), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	cfg := NewConfig(Options{WorkingDir: tmp, SchemaJSONPath: schemaPath})
	importer := NewImporter(cfg)
	importer.cfg.TblsConfig = &tblsconfig.Config{
		DSN: tblsconfig.DSN{URL: "postgres://localhost/app"},
	}

	if err := importer.LoadSchemaJSON(context.Background()); err != nil {
		t.Fatalf("LoadSchemaJSON returned error: %v", err)
	}

	if !importer.hasLoadedSchema() {
		t.Fatalf("expected schema to be marked as loaded")
	}

	if importer.schema == nil || importer.schema.Driver == nil || importer.schema.Driver.Name != "postgres" {
		t.Fatalf("unexpected schema driver: %#v", importer.schema)
	}

	converted, err := importer.Convert(context.Background())
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if len(converted) != 1 {
		t.Fatalf("expected one database schema, got %d", len(converted))
	}

	db := converted[0]
	if db.DatabaseInfo.Type != "postgres" || db.DatabaseInfo.Name != "public" {
		t.Fatalf("unexpected database info: %+v", db.DatabaseInfo)
	}

	if len(db.Tables) != 1 {
		t.Fatalf("expected single table, got %d", len(db.Tables))
	}

	table := db.Tables[0]
	if table.Name != "users" || table.Schema != "public" {
		t.Fatalf("unexpected table: %+v", table)
	}

	if len(table.ColumnOrder) != 2 || table.ColumnOrder[0] != "id" || table.ColumnOrder[1] != "email" {
		t.Fatalf("unexpected column order: %v", table.ColumnOrder)
	}

	if len(table.Constraints) != 1 || table.Constraints[0].Type != "PRIMARY KEY" {
		t.Fatalf("unexpected constraints: %+v", table.Constraints)
	}

	if len(table.Indexes) != 1 || !table.Indexes[0].IsUnique {
		t.Fatalf("unexpected indexes: %+v", table.Indexes)
	}

	if got := table.Columns["id"].DataType; got != snapTypeInt {
		t.Fatalf("expected id column type int, got %s", got)
	}
}

func TestLoadSchemaJSONMissingFile(t *testing.T) {
	t.Parallel()

	cfg := NewConfig(Options{SchemaJSONPath: "./missing.json"})
	importer := NewImporter(cfg)

	if err := importer.LoadSchemaJSON(context.Background()); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestLoadSchemaJSONValidationFailure(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.json")

	json := `{"tables":[]}`
	if err := os.WriteFile(schemaPath, []byte(json), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	cfg := NewConfig(Options{SchemaJSONPath: schemaPath})
	importer := NewImporter(cfg)

	if err := importer.LoadSchemaJSON(context.Background()); err == nil {
		t.Fatalf("expected validation error for schema without driver and tables")
	}
}

func TestConvertAppliesDriverSpecificTypeMapping(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.json")

	json := `{"driver":{"name":"mysql"},"tables":[{"name":"users","type":"TABLE","columns":[{"name":"id","type":"BIGINT"},{"name":"created_at","type":"datetime"},{"name":"flags","type":"json"}]}]}`
	if err := os.WriteFile(schemaPath, []byte(json), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	cfg := NewConfig(Options{SchemaJSONPath: schemaPath})
	importer := NewImporter(cfg)

	if err := importer.LoadSchemaJSON(context.Background()); err != nil {
		t.Fatalf("LoadSchemaJSON returned error: %v", err)
	}

	schemas, err := importer.Convert(context.Background())
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected one schema, got %d", len(schemas))
	}

	tables := schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected one table, got %d", len(tables))
	}

	cols := tables[0].Columns
	if cols["id"].DataType != snapTypeInt {
		t.Fatalf("expected id to map to int, got %s", cols["id"].DataType)
	}

	if cols["created_at"].DataType != snapTypeDateTime {
		t.Fatalf("expected created_at to map to datetime, got %s", cols["created_at"].DataType)
	}

	if cols["flags"].DataType != snapTypeJSON {
		t.Fatalf("expected flags to map to json, got %s", cols["flags"].DataType)
	}
}

func TestConvertMarksPrimaryKeysFromConstraints(t *testing.T) {
	t.Parallel()

	cfg := NewConfig(Options{
		WorkingDir:     ".",
		SchemaJSONPath: filepath.Join("..", "testdata", "tbls", "kanban_schema.json"),
	})

	importer := NewImporter(cfg)

	if err := importer.LoadSchemaJSON(context.Background()); err != nil {
		t.Fatalf("LoadSchemaJSON returned error: %v", err)
	}

	schemas, err := importer.Convert(context.Background())
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	var boards *snapsql.TableInfo

	for _, s := range schemas {
		for _, tbl := range s.Tables {
			if tbl.Name == "boards" {
				boards = tbl
			}
		}
	}

	if boards == nil {
		t.Fatalf("expected boards table to be present in schema")
	}

	idColumn, ok := boards.Columns["id"]
	if !ok {
		t.Fatalf("boards table missing id column")
	}

	if !idColumn.IsPrimaryKey {
		t.Fatalf("expected boards.id to be marked as primary key")
	}
}
