package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestSimpleTypeInference tests basic type inference functionality
func TestSimpleTypeInference(t *testing.T) {
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
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
		},
	}

	// Parse simple SQL
	sqlText := "SELECT id, name FROM users"
	tokens, err := tokenizer.Tokenize(sqlText)
	assert.NoError(t, err)

	funcDef := &parser.FunctionDefinition{
		Name:           "TestFunction",
		Parameters:     map[string]any{},
		ParameterOrder: []string{},
	}

	stmt, err := parser.Parse(tokens, funcDef, &parser.ParseOptions{})
	assert.NoError(t, err)

	// Create inference engine
	engine := NewTypeInferenceEngine2(schemas, stmt, nil)
	assert.True(t, engine != nil)

	// Test basic functionality
	t.Run("Engine Creation", func(t *testing.T) {
		assert.True(t, engine.schemaResolver != nil)
		assert.True(t, engine.fieldNameGen != nil)
		assert.Equal(t, snapsql.DialectPostgres, engine.context.Dialect)
	})

	t.Run("Basic Type Inference", func(t *testing.T) {
		// Perform type inference
		fields, err := engine.InferSelectTypes()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(fields))

		// Check first field (id)
		assert.Equal(t, "id", fields[0].Name)
		assert.Equal(t, "int", fields[0].Type.BaseType)
		assert.Equal(t, false, fields[0].Type.IsNullable)

		// Check second field (name)
		assert.Equal(t, "name", fields[1].Name)
		assert.Equal(t, "string", fields[1].Type.BaseType)
		assert.Equal(t, false, fields[1].Type.IsNullable)
	})
}
