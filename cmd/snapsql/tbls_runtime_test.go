package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestResolveDatabaseFromTblsUnavailable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ctx := &Context{Config: filepath.Join(dir, "snapsql.yaml")}

	db, err := resolveDatabaseFromTbls(ctx)
	if !errors.Is(err, ErrTblsDatabaseUnavailable) {
		t.Fatalf("expected ErrTblsDatabaseUnavailable, got %v", err)
	}

	if db != nil {
		t.Fatalf("expected no database, got %+v", db)
	}
}

func TestResolveDatabaseFromTblsSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ctx := &Context{Config: filepath.Join(dir, "snapsql.yaml")}

	content := "dsn: postgres://demo:demo@localhost:5432/demo?sslmode=disable\n"
	assert.NoError(t, os.WriteFile(filepath.Join(dir, ".tbls.yml"), []byte(content), 0o644))

	db, err := resolveDatabaseFromTbls(ctx)
	assert.NoError(t, err)

	if db == nil {
		t.Fatalf("expected database config")
	}

	assert.Equal(t, "pgx", db.Driver)
	assert.Equal(t, "postgres://demo:demo@localhost:5432/demo?sslmode=disable", db.Connection)
}
