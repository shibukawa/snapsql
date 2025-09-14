package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/query"
)

// helper to write a file
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	assert.NoError(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

func TestQuery_DryRun_SQLTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sql := "SELECT id FROM users WHERE id = 1"
	path := writeTemp(t, dir, "simple.snap.sql", sql)

	// Load intermediate and optimize
	format, err := query.LoadIntermediateFormat(path)
	assert.NoError(t, err)

	optimized, err := intermediate.OptimizeInstructions(format.Instructions, "postgresql")
	assert.NoError(t, err)

	q := &QueryCmd{}
	outSQL, args, err := q.buildSQLFromOptimized(optimized, format, map[string]any{})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(outSQL, "SELECT"))
	assert.False(t, strings.Contains(outSQL, "?"))
	assert.Equal(t, 0, len(args))
}

func TestQuery_DryRun_SQLTemplate_WithParam(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sql := "" +
		"/*#\n" +
		"function_name: select_user\n" +
		"parameters:\n" +
		"  user_id: int\n" +
		"*/\n" +
		"SELECT id FROM users WHERE id = /*= user_id */1"
	path := writeTemp(t, dir, "withparam.snap.sql", sql)

	// Load intermediate and optimize
	format, err := query.LoadIntermediateFormat(path)
	assert.NoError(t, err)

	optimized, err := intermediate.OptimizeInstructions(format.Instructions, "postgresql")
	assert.NoError(t, err)

	q := &QueryCmd{}
	outSQL, args, err := q.buildSQLFromOptimized(optimized, format, map[string]any{"user_id": 1})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(outSQL, "?"), outSQL)
	assert.Equal(t, 1, len(args))
	assert.Equal(t, 1, args[0])
}

func TestQuery_DryRun_MarkdownTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	md := "" +
		"# Title\n\n" +
		"## Description\n\nexample\n\n" +
		"" +
		"## SQL\n\n" +
		"```sql\nSELECT id FROM users WHERE id = 1\n```\n"

	path := writeTemp(t, dir, "simple.snap.md", md)

	format, err := query.LoadIntermediateFormat(path)
	assert.NoError(t, err)

	optimized, err := intermediate.OptimizeInstructions(format.Instructions, "postgresql")
	assert.NoError(t, err)

	q := &QueryCmd{}
	outSQL, args, err := q.buildSQLFromOptimized(optimized, format, map[string]any{})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(outSQL, "SELECT"))
	assert.False(t, strings.Contains(outSQL, "?"))
	assert.Equal(t, 0, len(args))
}

func TestQuery_DryRun_MarkdownTemplate_WithParam(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	md := "" +
		"# Title\n\n" +
		"## Description\n\nexample\n\n" +
		"## Parameters\n\n" +
		"```yaml\nuser_id: int\n```\n\n" +
		"## SQL\n\n" +
		"```sql\nSELECT id FROM users WHERE id = /*= user_id */1\n```\n"

	path := writeTemp(t, dir, "withparam.snap.md", md)

	format, err := query.LoadIntermediateFormat(path)
	assert.NoError(t, err)

	optimized, err := intermediate.OptimizeInstructions(format.Instructions, "postgresql")
	assert.NoError(t, err)

	q := &QueryCmd{}
	outSQL, args, err := q.buildSQLFromOptimized(optimized, format, map[string]any{"user_id": 1})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(outSQL, "?"), outSQL)
	assert.Equal(t, 1, len(args))
	assert.Equal(t, 1, args[0])
}

func TestQuery_IsDangerousQuery(t *testing.T) {
	q := &QueryCmd{}
	assert.True(t, q.isDangerousQuery("DELETE FROM users"))
	assert.True(t, q.isDangerousQuery("update users set name='x'"))
	assert.False(t, q.isDangerousQuery("DELETE FROM users WHERE id = 1"))
}

func TestQuery_InvalidOutputFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sql := "SELECT 1"
	path := writeTemp(t, dir, "format.snap.sql", sql)

	q := &QueryCmd{
		TemplateFile: path,
		DryRun:       true,
		Format:       "invalid",
	}

	// Quiet to avoid stdout noise
	err := q.Run(&Context{Quiet: true})
	assert.Error(t, err)
}
