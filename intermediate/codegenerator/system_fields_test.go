package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSystemFieldsInInsert tests that system fields (created_at, updated_at) are properly
// inserted when Config with SystemFields is provided to GenerationContext.
//
// Validates:
// 1. System fields are added to column list when generating INSERT instructions
// 2. System fields are added to VALUES when generating INSERT instructions
// 3. Instructions respect Config settings (enabled, nil, or empty)
func TestSystemFieldsInInsert(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		config               *snapsql.Config
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		description          string
	}{
		{
			name: "INSERT with system fields enabled",
			sql:  `INSERT INTO users (id, name) VALUES (1, 'Alice')`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "created_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at, updated_at) VALUES (1, 'Alice', ", Pos: "1:1"},
				{Op: OpEmitSystemValue, SystemField: "created_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: ", ", Pos: ""},
				{Op: OpEmitSystemValue, SystemField: "updated_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: ")", Pos: "1:48"},
			},
			description: "INSERT with created_at and updated_at system fields",
		},
		{
			name:    "INSERT without system fields (nil config)",
			sql:     `INSERT INTO users (id, name) VALUES (1, 'Alice')`,
			config:  nil,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (1, 'Alice')", Pos: "1:1"},
			},
			description: "INSERT without system fields configuration",
		},
		{
			name: "INSERT with empty system fields",
			sql:  `INSERT INTO users (id, name) VALUES (2, 'Bob')`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES (2, 'Bob')", Pos: "1:1"},
			},
			description: "INSERT with empty system fields configuration",
		},
		{
			name: "INSERT with system field already in column list",
			sql:  `INSERT INTO users (id, name, created_at) VALUES (3, 'Charlie', '2025-01-01')`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "created_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				// created_at is already in the column list, so should be skipped
				// only updated_at should be added as system field
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at, updated_at) VALUES (3, 'Charlie', '2025-01-01', ", Pos: "1:1"},
				{Op: OpEmitSystemValue, SystemField: "updated_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: ")", Pos: "1:76"},
			},
			description: "INSERT with created_at already explicitly in column list - should not duplicate",
		},
		{
			name: "INSERT with all system fields already in column list",
			sql:  `INSERT INTO users (id, name, created_at, updated_at) VALUES (4, 'David', '2025-01-01', '2025-01-01')`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "created_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnInsert: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				// Both created_at and updated_at are already in the column list, so no system fields added
				// But VALUES still have actual values, so no EMIT_SYSTEM_VALUE instructions
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at, updated_at) VALUES (4, 'David', '2025-01-01', '2025-01-01')", Pos: "1:1"},
			},
			description: "INSERT with all system fields already explicitly in column list - should not duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			if tt.config != nil {
				ctx.SetConfig(tt.config)
			}

			instructions, _, _, err := GenerateInsertInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateInsertInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			// Compare instruction sequences
			assert.Equal(t, tt.expectedInstructions, instructions,
				"System field instructions should match expected sequence")

			// Log for debugging
			t.Logf("✓ %s: Generated %d instructions", tt.name, len(instructions))
		})
	}
}

// TestSystemFieldsInUpdate tests that system fields (updated_at) are properly
// updated when Config with SystemFields is provided to GenerationContext.
//
// Validates:
// 1. System fields are added to SET clause when generating UPDATE instructions
// 2. System field default values are correctly placed in SET clause
// 3. Instructions respect Config settings (enabled or nil)
func TestSystemFieldsInUpdate(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		config               *snapsql.Config
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		description          string
	}{
		{
			name: "UPDATE with system fields enabled",
			sql:  `UPDATE users SET name = 'Alice' WHERE id = 1`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnUpdate: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET name = 'Alice' , updated_at = ", Pos: "1:1"},
				{Op: OpEmitSystemValue, SystemField: "updated_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: "WHERE id = 1", Pos: "1:33"},
			},
			description: "UPDATE with updated_at system field",
		},
		{
			name:    "UPDATE without system fields",
			sql:     `UPDATE users SET name = 'Bob' WHERE id = 2`,
			config:  nil,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET name = 'Bob' WHERE id = 2", Pos: "1:1"},
			},
			description: "UPDATE without system fields configuration",
		},
		{
			name: "UPDATE with system field already in SET clause",
			sql:  `UPDATE users SET name = 'Eve', updated_at = '2025-01-02' WHERE id = 5`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnUpdate: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				// updated_at is already in SET clause, so should not duplicate
				{Op: OpEmitStatic, Value: "UPDATE users SET name = 'Eve', updated_at = '2025-01-02' WHERE id = 5", Pos: "1:1"},
			},
			description: "UPDATE with updated_at already explicitly in SET clause - should not duplicate",
		},
		{
			name: "UPDATE with multiple system fields",
			sql:  `UPDATE users SET name = 'Frank' WHERE id = 6`,
			config: &snapsql.Config{
				System: snapsql.SystemConfig{
					Fields: []snapsql.SystemField{
						{
							Name:              "updated_at",
							Type:              "datetime",
							ExcludeFromSelect: true,
							OnUpdate: snapsql.SystemFieldOperation{
								Default: "CURRENT_TIMESTAMP",
							},
						},
						{
							Name:              "updated_by",
							Type:              "varchar",
							ExcludeFromSelect: true,
							OnUpdate: snapsql.SystemFieldOperation{
								Default: "CURRENT_USER",
							},
						},
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET name = 'Frank' , updated_at = ", Pos: "1:1"},
				{Op: OpEmitSystemValue, SystemField: "updated_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: ", updated_by = ", Pos: ""},
				{Op: OpEmitSystemValue, SystemField: "updated_by", DefaultValue: "CURRENT_USER", Pos: ""},
				{Op: OpEmitStatic, Value: "WHERE id = 6", Pos: "1:33"},
			},
			description: "UPDATE with multiple system fields (updated_at and updated_by)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			if tt.config != nil {
				ctx.SetConfig(tt.config)
			}

			instructions, _, _, err := GenerateUpdateInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateUpdateInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			// Compare instruction sequences
			assert.Equal(t, tt.expectedInstructions, instructions,
				"System field instructions should match expected sequence")

			// Log for debugging
			t.Logf("✓ %s: Generated %d instructions", tt.name, len(instructions))
		})
	}
}

// TestSystemFieldsInInsertSelect tests INSERT...SELECT with system fields
func TestSystemFieldsInInsertSelect(t *testing.T) {
	// System fields configuration
	config := &snapsql.Config{
		System: snapsql.SystemConfig{
			Fields: []snapsql.SystemField{
				{
					Name: "created_at",
					Type: "timestamp",
					OnInsert: snapsql.SystemFieldOperation{
						Default: "NOW()",
					},
				},
				{
					Name: "updated_at",
					Type: "timestamp",
					OnInsert: snapsql.SystemFieldOperation{
						Default: "NOW()",
					},
				},
			},
		},
	}

	tests := []struct {
		name                   string
		sql                    string
		dialect                snapsql.Dialect
		config                 *snapsql.Config
		expectedInstructions   []Instruction
		shouldHaveSystemFields bool
	}{
		{
			name:                   "INSERT...SELECT with system fields",
			sql:                    "INSERT INTO lists (board_id, name) SELECT id, name FROM templates",
			dialect:                snapsql.DialectSQLite,
			config:                 config,
			shouldHaveSystemFields: true,
			expectedInstructions: []Instruction{
				// Should include system field names in INSERT INTO
				// Then SELECT with system field expressions
				// Note: For SQLite dialect, NOW() is normalized to CURRENT_TIMESTAMP
				{Op: OpEmitStatic, Value: "INSERT INTO lists ( board_id, name , created_at, updated_at) SELECT id, name , CURRENT_TIMESTAMP AS created_at, CURRENT_TIMESTAMP AS updated_at FROM templates", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			if tt.config != nil {
				ctx.SetConfig(tt.config)
			}

			instructions, _, _, err := GenerateInsertInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateInsertInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			// Find the main EMIT_STATIC instruction (should contain full SQL)
			mainInstr := instructions[0]
			assert.Equal(t, OpEmitStatic, mainInstr.Op)

			if tt.shouldHaveSystemFields {
				// Check that system fields appear in SELECT
				assert.Contains(t, mainInstr.Value, "created_at")
				assert.Contains(t, mainInstr.Value, "updated_at")
				// For SQLite, NOW() should be normalized to CURRENT_TIMESTAMP
				assert.Contains(t, mainInstr.Value, "CURRENT_TIMESTAMP")
			}

			t.Logf("✓ %s: Generated SQL: %s", tt.name, mainInstr.Value)
		})
	}
}
