package main

import (
	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/query"
	"strings"
	"testing"
)

func TestDryRun_MissingRequiredParams_ShowsFriendlyError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sql := "" +
		"/*#\n" +
		"function_name: pg_items_by_id\n" +
		"parameters:\n" +
		"  pid: int\n" +
		"*/\n" +
		"SELECT id FROM items WHERE id = /*= pid */0;\n"
	path := writeTemp(t, dir, "missing_param.snap.sql", sql)

	// Prepare command
	q := &QueryCmd{TemplateFile: path, DryRun: true, Dialect: "postgresql"}
	// Execute dry-run with empty params
	err := q.executeDryRun(&Context{Quiet: true}, map[string]any{}, query.QueryOptions{})
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "missing required parameter")
	assert.Contains(t, err.Error(), "pid (int)")
}
