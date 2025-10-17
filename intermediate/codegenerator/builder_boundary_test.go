package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoopBoundaryValidation tests that loops must be preceded by comma, AND, or OR
// ループ直前の要素がカンマ、AND、ORでない場合、エラーが出力されることをテスト
func TestLoopBoundaryValidation(t *testing.T) {
	t.Skip("Loop boundary validation not yet implemented - this test is for future work")

	tests := []struct {
		name          string
		sql           string
		dialect       snapsql.Dialect
		shouldError   bool
		errorContains string
	}{
		{
			name: "loop preceded by comma (valid)",
			sql: `INSERT INTO user_tags (user_id, tag) 
VALUES /*# for user : users */(/*= user.id */, /*= user.tags */) /*# end */`,
			dialect:     snapsql.DialectPostgres,
			shouldError: false,
		},
		{
			name:        "loop preceded by AND (valid)",
			sql:         `SELECT id FROM users WHERE status = 'active' AND /*# for filter : filters *//*= filter *//*# end */`,
			dialect:     snapsql.DialectPostgres,
			shouldError: false,
		},
		{
			name:        "loop preceded by OR (valid)",
			sql:         `SELECT id FROM users WHERE status = 'active' OR /*# for filter : filters *//*= filter *//*# end */`,
			dialect:     snapsql.DialectPostgres,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := map[string]any{
				"users": []any{
					map[string]any{"id": int64(1), "tags": []any{"tag1"}},
				},
				"filters": []any{"filter1", "filter2"},
				"tags":    []any{"tag1", "tag2"},
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})

			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			ctx := NewGenerationContext(tt.dialect)
			builder := NewInstructionBuilder(ctx)

			for _, clause := range stmt.Clauses() {
				tokens := clause.RawTokens()
				err := builder.ProcessTokens(tokens)
				require.NoError(t, err, "ProcessTokens should succeed")
			}

			instructions := builder.mergeStaticInstructions()
			require.NotEmpty(t, instructions, "instructions should not be empty")
		})
	}
}

// TestLoopEndCommaHandling tests EMIT_UNLESS_BOUNDARY for trailing comma before loop END
// ループEND直前のカンマをEMIT_UNLESS_BOUNDARYで出力することをテスト
func TestLoopEndCommaHandling(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
	}{
		{
			name: "loop with trailing comma",
			sql: `INSERT INTO tags (tag)
VALUES /*# for tag : tags */(/*= tag */), /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO tags (tag) VALUES "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "("},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: ")"},
				{Op: OpEmitUnlessBoundary, Value: ", "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
			},
		},
		{
			name: "nested loops with trailing comma",
			sql: `INSERT INTO user_details (user_id, tag) 
VALUES 
  /*# for user : users */
    /*# for tag : user.tags */(/*= user.id */, /*= tag */), /*# end */
  /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO user_details (user_id, tag) VALUES "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: " "},
				{Op: OpLoopStart, EnvIndex: ptr(2), CollectionExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: "("},
				{Op: OpEmitEval, ExprIndex: ptr(2)},
				{Op: OpEmitStatic, Value: ", "},
				{Op: OpEmitEval, ExprIndex: ptr(3)},
				{Op: OpEmitStatic, Value: ")"},
				{Op: OpEmitUnlessBoundary, Value: ", "},
				{Op: OpLoopEnd, EnvIndex: ptr(1)},
				{Op: OpEmitStatic, Value: " "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
			},
		},
		{
			name: "loop with AND delimiter",
			sql: `SELECT id FROM users 
WHERE status = 'active' AND /*# for cond : conditions *//*= cond */ AND /*# end */status = 'verified'`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE status = 'active' AND "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitUnlessBoundary, Value: " AND "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "status = 'verified'"},
			},
		},
		{
			name: "loop with OR delimiter",
			sql: `SELECT id FROM users 
WHERE status = 'pending' OR /*# for status : statuses */status = '/*= status */' OR /*# end */false`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE status = 'pending' OR "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "status = '/*= status */'"},
				{Op: OpEmitUnlessBoundary, Value: " OR "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "false"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := map[string]any{
				"tags":       []any{"tag1", "tag2"},
				"users":      []any{map[string]any{"id": int64(1), "tags": []any{"tag1", "tag2"}}},
				"conditions": []any{"cond1", "cond2"},
				"statuses":   []any{"active", "inactive"},
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")

			ctx := NewGenerationContext(tt.dialect)
			builder := NewInstructionBuilder(ctx)

			for _, clause := range stmt.Clauses() {
				tokens := clause.RawTokens()
				err := builder.ProcessTokens(tokens)
				require.NoError(t, err, "ProcessTokens should succeed")
			}

			instructions := builder.mergeStaticInstructions()

			// カットしない（全指令を検証）
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			require.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps),
				"Instruction count mismatch")

			for i, expected := range tt.expectedInstructions {
				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Op mismatch at index %d", i)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Value mismatch at index %d", i)
				}

				if expected.ExprIndex != nil {
					assert.Equal(t, expected.ExprIndex, actual.ExprIndex, "ExprIndex mismatch at index %d", i)
				}

				if expected.EnvIndex != nil {
					assert.Equal(t, expected.EnvIndex, actual.EnvIndex, "EnvIndex mismatch at index %d", i)
				}

				if expected.CollectionExprIndex != nil {
					assert.Equal(t, expected.CollectionExprIndex, actual.CollectionExprIndex, "CollectionExprIndex mismatch at index %d", i)
				}
			}
		})
	}
}

// TestLoopEndBoundaryInsertion tests BOUNDARY insertion after loop END
// ループ終了後に BOUNDARY が挿入されることをテスト
// NOTE: 実装完了後、期待値を更新する予定
func TestLoopEndBoundaryInsertion(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
	}{
		{
			name: "loop at end of clause - expects BOUNDARY after LOOP_END",
			sql: `INSERT INTO tags (tag)
VALUES /*# for tag : tags */(/*= tag */), /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO tags (tag) VALUES "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "("},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: ")"},
				{Op: OpEmitUnlessBoundary, Value: ", "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
				{Op: OpBoundary},
			},
		},
		{
			name: "loop followed by static text - no BOUNDARY",
			sql: `INSERT INTO tags (tag)
VALUES /*# for tag : tags */(/*= tag */), /*# end */ ON CONFLICT DO NOTHING`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO tags (tag) VALUES "},
				{Op: OpLoopStart, EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "("},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: ")"},
				{Op: OpEmitUnlessBoundary, Value: ", "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
				{Op: OpEmitStatic, Value: " ON CONFLICT DO NOTHING"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := map[string]any{
				"tags": []any{"tag1", "tag2"},
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")

			ctx := NewGenerationContext(tt.dialect)
			builder := NewInstructionBuilder(ctx)

			for _, clause := range stmt.Clauses() {
				tokens := clause.RawTokens()
				err := builder.ProcessTokens(tokens)
				require.NoError(t, err, "ProcessTokens should succeed")
			}

			instructions := builder.Finalize()

			// カットしない（全指令を検証）
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			require.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps),
				"Instruction count mismatch")

			for i, expected := range tt.expectedInstructions {
				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Op mismatch at index %d", i)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Value mismatch at index %d", i)
				}

				if expected.ExprIndex != nil {
					assert.Equal(t, expected.ExprIndex, actual.ExprIndex, "ExprIndex mismatch at index %d", i)
				}

				if expected.EnvIndex != nil {
					assert.Equal(t, expected.EnvIndex, actual.EnvIndex, "EnvIndex mismatch at index %d", i)
				}

				if expected.CollectionExprIndex != nil {
					assert.Equal(t, expected.CollectionExprIndex, actual.CollectionExprIndex, "CollectionExprIndex mismatch at index %d", i)
				}
			}
		})
	}
}
