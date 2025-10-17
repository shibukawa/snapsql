package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ptrInt は int のポインタを返すヘルパー関数
func ptrInt(i int) *int {
	return &i
}

func TestGenerateInsertInstructions(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectError          bool
		errorContains        string
		expectedInstructions []Instruction
		expectedCELCount     int
		expectedEnvCount     int
	}{
		{
			name:        "basic insert with single row",
			sql:         "INSERT INTO users (id, name, email) VALUES (1, 'John', 'john@example.com')",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, email) VALUES (1, 'John', 'john@example.com')", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert with multiple columns",
			sql:         "INSERT INTO products (id, name, price, category) VALUES (1, 'Widget', 9.99, 'Tools')",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO products (id, name, price, category) VALUES (1, 'Widget', 9.99, 'Tools')", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert with multiple rows",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie')",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie')", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert with returning clause",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'John') RETURNING id, name, created_at",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (1, 'John') RETURNING id, name, created_at", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert with multiple rows",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie')",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie')", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert into select",
			sql:         "INSERT INTO user_archive (id, name) SELECT id, name FROM users WHERE active = true",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO user_archive (id, name) SELECT id, name FROM users WHERE active = true", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name: "insert with directive",
			sql: `/*# parameters: { user_id: int, user_name: string } */
INSERT INTO users (id, name)
VALUES (/*= user_id */ 1, /*= user_name */ 'John')`,
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (", Pos: "2:1"},
				{Op: OpEmitEval, ExprIndex: ptrInt(0), Pos: "3:9"},
				{Op: OpEmitStatic, Value: " 1, ", Pos: "3:23"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "3:27"},
				{Op: OpEmitStatic, Value: " 'John')", Pos: "3:43"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 2,
			expectedEnvCount: 1,
		},
		{
			name: "insert with on conflict",
			sql: `INSERT INTO users (id, name, email)
VALUES (1, 'John', 'john@example.com')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`,
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, email) VALUES (1, 'John', 'john@example.com') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "insert with returning clause on mariadb",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'John') RETURNING id, name",
			dialect:     snapsql.DialectMariaDB,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (1, 'John') RETURNING id, name", Pos: "1:1"},
				{Op: OpEmitSystemFor, Pos: ""},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			insertStmt, ok := stmt.(*parser.InsertIntoStatement)
			require.True(t, ok, "Expected InsertIntoStatement")

			ctx := NewGenerationContext(tt.dialect)
			instructions, expressions, environments, err := GenerateInsertInstructions(insertStmt, ctx)

			if tt.expectError {
				require.Error(t, err, "Expected error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err, "GenerateInsertInstructions should succeed")
			require.NotNil(t, instructions)
			require.NotNil(t, expressions)
			require.NotNil(t, environments)

			assert.Equal(t, tt.expectedInstructions, instructions,
				"Instructions should match exactly")
			assert.Equal(t, tt.expectedCELCount, len(expressions),
				"CEL expression count should match")
			assert.Equal(t, tt.expectedEnvCount, len(environments),
				"CEL environment count should match")

			t.Logf("✓ Generated %d instructions", len(instructions))
			t.Logf("✓ Generated %d CEL expressions", len(expressions))
			t.Logf("✓ Generated %d environments", len(environments))
		})
	}
}
