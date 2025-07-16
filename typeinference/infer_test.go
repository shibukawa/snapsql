package typeinference

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Helper function to load schema from YAML
func loadSchemaFromYAML(yamlContent string) ([]snapsql.DatabaseSchema, error) {
	var schema snapsql.DatabaseSchema
	err := yaml.Unmarshal([]byte(yamlContent), &schema)
	if err != nil {
		return nil, err
	}
	return []snapsql.DatabaseSchema{schema}, nil
}

// Parse SQL using actual parser
func parseSQL(sql string) (parser.StatementNode, error) {
	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, err
	}

	stmt, err := parser.Parse(tokens, nil, nil)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

// Test data structure for table-driven tests
type inferenceTestCase struct {
	name      string
	sql       string
	schema    string // YAML schema definition
	expected  []InferredFieldInfo
	expectErr bool
	errMsg    string
}

// Test InferFieldTypes with table-driven tests
func TestInferFieldTypes_TableDriven(t *testing.T) {
	testCases := []inferenceTestCase{
		{
			name: "simple SELECT with basic columns",
			sql:  "SELECT id, name FROM users",
			schema: `
name: test_db
tables:
- name: users
  columns:
    id:
      name: id
      dataType: INTEGER
      nullable: false
      isPrimaryKey: true
    name:
      name: name
      dataType: TEXT
      nullable: false
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`,
			expected: []InferredFieldInfo{
				{
					Name:         "id",
					OriginalName: "id",
					Type: &TypeInfo{
						BaseType:   "INTEGER",
						IsNullable: false,
					},
					Source: FieldSource{
						Type:   "column",
						Table:  "users",
						Column: "id",
					},
				},
				{
					Name:         "name",
					OriginalName: "name",
					Type: &TypeInfo{
						BaseType:   "TEXT",
						IsNullable: false,
					},
					Source: FieldSource{
						Type:   "column",
						Table:  "users",
						Column: "name",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "SELECT with nullable column",
			sql:  "SELECT email FROM users",
			schema: `
name: test_db
tables:
  - name: users
    columns:
      email:
        name: email
        dataType: TEXT
        nullable: true
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`,
			expected: []InferredFieldInfo{
				{
					Name:         "email",
					OriginalName: "email",
					Type: &TypeInfo{
						BaseType:   "TEXT",
						IsNullable: true,
					},
					Source: FieldSource{
						Type:   "column",
						Table:  "users",
						Column: "email",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "missing table error",
			sql:  "SELECT id FROM nonexistent",
			schema: `
name: test_db
tables: []
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`,
			expectErr: true,
			errMsg:    "does not exist in any available table",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load schema from YAML
			schemas, err := loadSchemaFromYAML(tc.schema)
			assert.NoError(t, err, "Failed to load schema from YAML")

			// Parse SQL
			stmt, err := parseSQL(tc.sql)
			assert.NoError(t, err, "Failed to parse SQL")

			// Perform type inference
			results, err := InferFieldTypes(schemas, stmt, nil)

			if tc.expectErr {
				assert.True(t, err != nil, "Expected error or validation errors but got none")
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg, "Error message doesn't match")
				}
				return
			}

			assert.NoError(t, err, "Unexpected error during type inference")
			assert.Equal(t, len(tc.expected), len(results), "Number of inferred fields doesn't match")

			for i, expected := range tc.expected {
				if i < len(results) {
					actual := results[i]
					assert.Equal(t, expected.Name, actual.Name, "Name mismatch at index %d", i)
					assert.Equal(t, expected.OriginalName, actual.OriginalName, "OriginalName mismatch at index %d", i)
					if expected.Type != nil && actual.Type != nil {
						assert.Equal(t, expected.Type.BaseType, actual.Type.BaseType, "BaseType mismatch at index %d", i)
						assert.Equal(t, expected.Type.IsNullable, actual.Type.IsNullable, "IsNullable mismatch at index %d", i)
					}
					assert.Equal(t, expected.Source.Type, actual.Source.Type, "Source.Type mismatch at index %d", i)
					assert.Equal(t, expected.Source.Table, actual.Source.Table, "Source.Table mismatch at index %d", i)
					assert.Equal(t, expected.Source.Column, actual.Source.Column, "Source.Column mismatch at index %d", i)
				}
			}
		})
	}
}

// Test ValidateStatementSchema function
func TestValidateStatementSchema_TableDriven(t *testing.T) {
	testCases := []struct {
		name      string
		sql       string
		schema    string
		expectErr bool
	}{
		{
			name: "valid schema",
			sql:  "SELECT id, name FROM users",
			schema: `
name: test_db
tables:
  - name: users
    columns:
      id:
        name: id
        dataType: INTEGER
        nullable: false
      name:
        name: name
        dataType: TEXT
        nullable: false
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`,
			expectErr: false,
		},
		{
			name: "missing table",
			sql:  "SELECT id FROM nonexistent",
			schema: `
name: test_db
tables: []
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load schema from YAML
			schemas, err := loadSchemaFromYAML(tc.schema)
			assert.NoError(t, err, "Failed to load schema from YAML")

			// Parse SQL
			stmt, err := parseSQL(tc.sql)
			assert.NoError(t, err, "Failed to parse SQL")

			// Perform schema validation
			validationErrors, err := ValidateStatementSchema(schemas, stmt)

			if tc.expectErr {
				assert.True(t, err != nil || len(validationErrors) > 0, "Expected error or validation errors but got none")
			} else {
				assert.NoError(t, err, "Unexpected error during schema validation")
				assert.Equal(t, 0, len(validationErrors), "Expected no validation errors")
			}
		})
	}
}

// Test edge cases
func TestInferFieldTypes_EdgeCases(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		stmt, err := parseSQL("SELECT id FROM users")
		assert.NoError(t, err)

		results, err := InferFieldTypes(nil, stmt, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("nil statement", func(t *testing.T) {
		schemas, err := loadSchemaFromYAML(`
name: test_db
tables: []
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`)
		assert.NoError(t, err)

		results, err := InferFieldTypes(schemas, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("empty schema list", func(t *testing.T) {
		stmt, err := parseSQL("SELECT id FROM users")
		assert.NoError(t, err)

		results, err := InferFieldTypes([]snapsql.DatabaseSchema{}, stmt, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, len(results))
	})
}

// Test basic functionality (simple smoke test)
func TestInferFieldTypes_Basic(t *testing.T) {
	schema := `
name: test_db
tables:
  - name: users
    columns:
      id:
        name: id
        dataType: INTEGER
        nullable: false
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`

	schemas, err := loadSchemaFromYAML(schema)
	assert.NoError(t, err)

	stmt, err := parseSQL("SELECT id FROM users")
	assert.NoError(t, err)

	results, err := InferFieldTypes(schemas, stmt, nil)
	assert.NoError(t, err)
	assert.True(t, len(results) > 0, "Should return at least one field")
}

// Test ValidateStatementSchema basic functionality
func TestValidateStatementSchema_Basic(t *testing.T) {
	schema := `
name: test_db
tables:
  - name: users
    columns:
      id:
        name: id
        dataType: INTEGER
        nullable: false
databaseInfo:
  type: sqlite
  version: "3.0"
  name: test_db
`

	schemas, err := loadSchemaFromYAML(schema)
	assert.NoError(t, err)

	stmt, err := parseSQL("SELECT id FROM users")
	assert.NoError(t, err)

	validationErrors, err := ValidateStatementSchema(schemas, stmt)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(validationErrors), "Expected no validation errors")
}
