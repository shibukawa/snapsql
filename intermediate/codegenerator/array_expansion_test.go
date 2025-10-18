package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArrayExpansionInstructionsDeepEqual tests array expansion with deep equal comparison of instructions.
//
// This test validates the complete instruction sequence for array expansion patterns,
// matching the pattern used in system_fields_test.go for comprehensive validation.
func TestArrayExpansionInstructionsDeepEqual(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		config               *snapsql.Config
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		description          string
	}{
		// ============ INSERT VALUES - Loop Directive Present (array and object expansion) ============
		{
			name:    "INSERT with FOR loop, array and object expansion, no system fields",
			sql:     `/*# parameters: { rows: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES /*= rows */(1, 'name')`,
			config:  nil,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES ", Pos: "1:55"},
				{Op: OpLoopStart, Variable: "r", CollectionExprIndex: ptrInt(0), EnvIndex: ptrInt(0)},
				{Op: OpEmitStatic, Value: "( ", Pos: "1:91"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:96"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:105"},
				{Op: OpEmitEval, ExprIndex: ptrInt(2), Pos: "1:107"},
				{Op: OpEmitStatic, Value: " )", Pos: "1:118"},
				{Op: OpLoopEnd, EnvIndex: ptrInt(0)},
			},
			description: "Array and Object expansion with FOR loop outside parentheses, no system fields",
		},
		{
			name: "INSERT with FOR loop, array and object expansion, with created_at system field",
			sql:  `/*# parameters: { rows: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES /*=rows */(1, 'name')`,
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
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at) VALUES ", Pos: "1:55"},
				{Op: OpLoopStart, Variable: "r", CollectionExprIndex: ptrInt(0), EnvIndex: ptrInt(0)},
				{Op: OpEmitStatic, Value: "( ", Pos: "1:91"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:96"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:105"},
				{Op: OpEmitEval, ExprIndex: ptrInt(2), Pos: "1:107"},
				{Op: OpEmitStatic, Value: ", ", Pos: ""},
				{Op: OpEmitSystemValue, SystemField: "created_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: " )", Pos: "1:118"},
				{Op: OpLoopEnd, EnvIndex: ptrInt(0)},
			},
			description: "Array and Object expansion with FOR loop and created_at system field",
		},

		// ============ INSERT VALUES - Loop Directive Present (object expansion) ============
		{
			name:    "INSERT with FOR loop, object expansion, no system fields",
			sql:     `/*# parameters: { rows: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES /*# for r : rows *//*= r */( 1, 'name' )/*# end */`,
			config:  nil,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES ", Pos: "1:55"},
				{Op: OpLoopStart, Variable: "r", CollectionExprIndex: ptrInt(0), EnvIndex: ptrInt(0)},
				{Op: OpEmitStatic, Value: "( ", Pos: "1:91"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:96"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:105"},
				{Op: OpEmitEval, ExprIndex: ptrInt(2), Pos: "1:107"},
				{Op: OpEmitStatic, Value: " )", Pos: "1:118"},
				{Op: OpLoopEnd, EnvIndex: ptrInt(0)},
			},
			description: "Object expansion with FOR loop outside parentheses, no system fields",
		},
		{
			name: "INSERT with FOR loop, object expansion, with created_at system field",
			sql:  `/*# parameters: { rows: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES /*# for r : rows *//*= r */( 1, 'name' )/*# end */`,
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
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at) VALUES ", Pos: "1:55"},
				{Op: OpLoopStart, Variable: "r", CollectionExprIndex: ptrInt(0), EnvIndex: ptrInt(0)},
				{Op: OpEmitStatic, Value: "( ", Pos: "1:91"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:96"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:105"},
				{Op: OpEmitEval, ExprIndex: ptrInt(2), Pos: "1:107"},
				{Op: OpEmitStatic, Value: ", ", Pos: ""},
				{Op: OpEmitSystemValue, SystemField: "created_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: " )", Pos: "1:118"},
				{Op: OpLoopEnd, EnvIndex: ptrInt(0)},
			},
			description: "Object expansion with FOR loop and created_at system field",
		},

		// ============ INSERT VALUES - No Loop (Single Object Expansion) ============
		{
			name:    "INSERT without FOR loop, single object expansion, no system fields",
			sql:     `/*# parameters: { u: { id: int, name: string } } */ INSERT INTO users (id, name) VALUES /*= u */( 1, 'name' )`,
			config:  nil,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name) VALUES ( ", Pos: "1:55"},
				{Op: OpEmitEval, ExprIndex: ptrInt(0), Pos: "1:94"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:103"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:105"},
				{Op: OpEmitStatic, Value: " )", Pos: "1:116"},
			},
			description: "Single object expansion without FOR loop, no system fields",
		},
		{
			name: "INSERT without FOR loop, single object expansion, with created_at system field",
			sql:  `/*# parameters: { u: { id: int, name: string } } */ INSERT INTO users (id, name) VALUES /*= u */( 1, 'name' )`,
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
					},
				},
			},
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO users (id, name, created_at) VALUES ( ", Pos: "1:55"},
				{Op: OpEmitEval, ExprIndex: ptrInt(0), Pos: "1:94"},
				{Op: OpEmitStatic, Value: ", ", Pos: "1:103"},
				{Op: OpEmitEval, ExprIndex: ptrInt(1), Pos: "1:105"},
				{Op: OpEmitStatic, Value: ", ", Pos: ""},
				{Op: OpEmitSystemValue, SystemField: "created_at", DefaultValue: "CURRENT_TIMESTAMP", Pos: ""},
				{Op: OpEmitStatic, Value: " )", Pos: "1:116"},
			},
			description: "Single object expansion without FOR loop, with created_at system field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			if tt.config != nil {
				ctx.SetConfig(tt.config)
			}

			instructions, _, _, err := GenerateInsertInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateInsertInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			// Compare instruction sequences using deep equal
			assert.Equal(t, tt.expectedInstructions, instructions,
				"Array expansion instructions should match expected sequence")

			t.Logf("✓ %s: Generated %d instructions", tt.name, len(instructions))
		})
	}
}
