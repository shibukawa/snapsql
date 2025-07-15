package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// ExpectedField represents expected test results
type ExpectedField struct {
	Name     string
	Type     string
	Nullable bool
}

// parseSQL is a helper function to parse SQL text for testing
func parseSQL(t *testing.T, sqlText string) parser.StatementNode {
	t.Helper()

	// Tokenize SQL
	tokens, err := tokenizer.Tokenize(sqlText)
	assert.NoError(t, err)

	// Parse with minimal function definition
	funcDef := &parser.FunctionDefinition{
		Name:           "TestFunction",
		Parameters:     map[string]any{},
		ParameterOrder: []string{},
	}

	options := &parser.ParseOptions{}
	stmt, err := parser.Parse(tokens, funcDef, options)
	assert.NoError(t, err)

	return stmt
}

// TestBasicTypeInference tests basic type inference functionality
func TestBasicTypeInference(t *testing.T) {
	// Create test database schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "testdb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "string",
							Nullable: false,
						},
						"age": {
							Name:     "age",
							DataType: "int",
							Nullable: true,
						},
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
		},
	}

	t.Run("Simple SELECT with column references", func(t *testing.T) {
		// Parse SQL
		sqlText := "SELECT id, name FROM users"
		stmt := parseSQL(t, sqlText)

		// Create inference engine
		engine := NewTypeInferenceEngine2(schemas, stmt, nil)

		// Perform type inference
		fields, err := engine.InferSelectTypes()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(fields))

		// Check field types
		assert.Equal(t, "id", fields[0].Name)
		assert.Equal(t, "int", fields[0].Type.BaseType)
		assert.Equal(t, false, fields[0].Type.IsNullable)

		assert.Equal(t, "name", fields[1].Name)
		assert.Equal(t, "string", fields[1].Type.BaseType)
		assert.Equal(t, false, fields[1].Type.IsNullable)
	})

	t.Run("Function call", func(t *testing.T) {
		// Parse SQL
		sqlText := "SELECT COUNT(id) AS user_count FROM users"
		stmt := parseSQL(t, sqlText)

		// Create inference engine
		engine := NewTypeInferenceEngine2(schemas, stmt, nil)

		// Perform type inference
		fields, err := engine.InferSelectTypes()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(fields))

		// Check function result type
		assert.Equal(t, "user_count", fields[0].Name)
		assert.Equal(t, "int", fields[0].Type.BaseType)
		assert.Equal(t, false, fields[0].Type.IsNullable)
	})
}
