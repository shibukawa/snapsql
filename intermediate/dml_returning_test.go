package intermediate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/pull"
)

func TestPipelineUpdateReturningGeneratesResponses(t *testing.T) {
	schemaPath := filepath.Join("..", "examples", "kanban", "schema", "global", "cards.yaml")

	table, err := pull.LoadTableFromYAMLFile(schemaPath)
	assert.NoError(t, err)

	tableInfo := map[string]*snapsql.TableInfo{
		strings.ToLower(table.Name): table,
	}

	sql := "UPDATE cards SET title = $1 WHERE id = $2 RETURNING id, list_id, title, description, position, created_at, updated_at"

	stmt, funcDef, err := parser.ParseSQLFile(strings.NewReader(sql), nil, ".", ".", parser.DefaultOptions)
	assert.NoError(t, err)

	cfg := &snapsql.Config{Dialect: "postgres"}

	pipeline := CreateDefaultPipeline(stmt, funcDef, cfg, tableInfo)

	format, err := pipeline.Execute()
	assert.NoError(t, err)

	if len(format.Responses) == 0 {
		t.Fatalf("expected returning responses to be generated, got none")
	}

	// Ensure primary fields are present
	foundID := false
	foundTitle := false

	for _, resp := range format.Responses {
		switch resp.Name {
		case "id":
			foundID = true
		case "title":
			foundTitle = true
		}
	}

	if !foundID || !foundTitle {
		t.Fatalf("expected id and title in responses, got %#v", format.Responses)
	}
}
