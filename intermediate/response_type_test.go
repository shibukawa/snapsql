package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
)

func TestDetermineResponseType(t *testing.T) {
	// Test with empty table info
	t.Run("EmptyTableInfo", func(t *testing.T) {
		// This test would require a proper SQL statement node
		// For now, we'll test the conversion function
		result := convertTableInfoToSchemas(nil)
		assert.Equal(t, []DatabaseSchema(nil), result)
	})

	// Test table info conversion
	t.Run("ConvertTableInfoToSchemas", func(t *testing.T) {
		tableInfo := map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":    {Name: "id", DataType: "int", Nullable: false, IsPrimaryKey: true},
					"name":  {Name: "name", DataType: "string", Nullable: true},
					"email": {Name: "email", DataType: "string", Nullable: true},
				},
			},
			"orders": {
				Name: "orders",
				Columns: map[string]*ColumnInfo{
					"id":      {Name: "id", DataType: "int", Nullable: false, IsPrimaryKey: true},
					"user_id": {Name: "user_id", DataType: "int", Nullable: false},
					"amount":  {Name: "amount", DataType: "decimal", Nullable: false},
				},
			},
		}

		schemas := convertTableInfoToSchemas(tableInfo)

		// Verify schema structure
		assert.Equal(t, 1, len(schemas))
		assert.Equal(t, "default", schemas[0].Name)
		assert.Equal(t, 2, len(schemas[0].Tables))

		// Find users table
		var (
			usersTable  *TableInfo
			ordersTable *TableInfo
		)

		for _, table := range schemas[0].Tables {
			switch table.Name {
			case "users":
				usersTable = table
			case "orders":
				ordersTable = table
			}
		}

		// Verify users table
		assert.True(t, usersTable != nil)
		assert.Equal(t, 3, len(usersTable.Columns))

		// Verify orders table
		assert.True(t, ordersTable != nil)
		assert.Equal(t, 3, len(ordersTable.Columns))
	})

	// When no schema is provided, determineResponseType synthesizes fallback responses with warnings
	t.Run("FallbackExtraction", func(t *testing.T) {
		// Prepare a very small in-memory SQL to parse
		md := "" +
			"# Title\n\n" +
			"## Description\n\nFallback test\n\n" +
			"## SQL\n\n" +
			"```sql\nSELECT a.id AS parent__id, a.name AS parent__name, b.id AS parent__children__id FROM a LEFT JOIN b ON b.a_id = a.id;\n```\n"

		// Use markdownparser + parser.ParseMarkdownFile similar to generation pipeline
		doc, err := markdownparser.Parse(strings.NewReader(md))
		assert.NoError(t, err)
		stmt, _, _, err := parser.ParseMarkdownFile(doc, "memory.md", ".", nil, parser.DefaultOptions)
		assert.NoError(t, err)

		// Call determineResponseType with empty schema to trigger fallback any responses
		responses, warnings := determineResponseType(stmt, nil)
		assert.Equal(t, 3, len(responses))
		assert.Equal(t, "parent__id", responses[0].Name)
		assert.Equal(t, "any", responses[0].Type)
		assert.NotZero(t, len(warnings))
		assert.Contains(t, strings.Join(warnings, ";"), "fallback")
	})
}

func TestNormalizeColumnNameSubquery(t *testing.T) {
	assert.Equal(t, "id", cleanIdentifier("sq.id"))
	assert.Equal(t, "name", cleanIdentifier("\"sq\".\"name\""))
}

func TestFallbackFieldNamingFromSQL(t *testing.T) {
	sql := "SELECT sq.id, sq.name FROM (SELECT id, name FROM users) AS sq"
	stmt, _, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, "inline.sql", "", parser.DefaultOptions)
	assert.NoError(t, err)

	selectStmt, ok := stmt.(*parser.SelectStatement)
	assert.True(t, ok)

	if ok {
		field := selectStmt.Select.Fields[0]
		t.Logf("FieldName=%q OriginalField=%q", field.FieldName, field.OriginalField)
	}

	responses := buildFallbackResponses(stmt)
	assert.Equal(t, 2, len(responses))
	assert.Equal(t, "id", responses[0].Name)
	assert.Equal(t, "name", responses[1].Name)
}
