package codegenerator

import (
	"strconv"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeStaticInstructions は連続する EMIT_STATIC 命令のマージをテストする
func TestMergeStaticInstructions(t *testing.T) {
	tests := []struct {
		name     string
		input    []Instruction
		expected []Instruction
	}{
		{
			name:     "empty instructions",
			input:    []Instruction{},
			expected: []Instruction{},
		},
		{
			name: "single instruction",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
			},
		},
		{
			name: "consecutive EMIT_STATIC instructions",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:8"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:10"},
				{Op: OpEmitStatic, Value: "FROM", Pos: "1:11"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM", Pos: "1:1"},
			},
		},
		{
			name: "EMIT_STATIC with system instructions",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:8"},
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id", Pos: "1:1"},
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
			},
		},
		{
			name: "multiple groups of EMIT_STATIC",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpIf, Pos: "1:8"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:9"},
				{Op: OpEmitStatic, Value: ",", Pos: "1:11"},
				{Op: OpEnd, Pos: "1:12"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:13"},
				{Op: OpEmitStatic, Value: "FROM", Pos: "1:14"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT ", Pos: "1:1"},
				{Op: OpIf, Pos: "1:8"},
				{Op: OpEmitStatic, Value: "id,", Pos: "1:9"},
				{Op: OpEnd, Pos: "1:12"},
				{Op: OpEmitStatic, Value: " FROM", Pos: "1:13"},
			},
		},
		{
			name: "no consecutive EMIT_STATIC",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpIf, Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:8"},
				{Op: OpEnd, Pos: "1:10"},
				{Op: OpEmitStatic, Value: "FROM", Pos: "1:11"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpIf, Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:8"},
				{Op: OpEnd, Pos: "1:10"},
				{Op: OpEmitStatic, Value: "FROM", Pos: "1:11"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			builder := NewInstructionBuilder(ctx)
			builder.instructions = tt.input

			result := builder.mergeStaticInstructions()

			assert.Equal(t, tt.expected, result, "merged instructions should match expected")
		})
	}
}

// TestProcessTokensWithWhitespaceAndComments はホワイトスペースとコメントのマージをテストする
func TestProcessTokensWithWhitespaceAndComments(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		expected []Instruction
	}{
		{
			name: "consecutive whitespaces merged to single space",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.WHITESPACE, Value: "  ", Position: tokenizer.Position{Line: 1, Column: 8}},
				{Type: tokenizer.WHITESPACE, Value: "\t", Position: tokenizer.Position{Line: 1, Column: 10}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 1, Column: 11}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:11"},
			},
		},
		{
			name: "block comment treated as single space",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.BLOCK_COMMENT, Value: "/* comment */", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 1, Column: 20}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:20"},
			},
		},
		{
			name: "line comment treated as single space",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.LINE_COMMENT, Value: "-- comment", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 2, Column: 1}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "2:1"},
			},
		},
		{
			name: "mixed whitespace and comments merged",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.BLOCK_COMMENT, Value: "/* comment */", Position: tokenizer.Position{Line: 1, Column: 8}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 21}},
				{Type: tokenizer.LINE_COMMENT, Value: "-- comment", Position: tokenizer.Position{Line: 1, Column: 22}},
				{Type: tokenizer.WHITESPACE, Value: "\n", Position: tokenizer.Position{Line: 1, Column: 32}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 2, Column: 1}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: " ", Pos: "1:7"},
				{Op: OpEmitStatic, Value: "id", Pos: "2:1"},
			},
		},
		{
			name: "no whitespace or comments",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 1, Column: 7}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT", Pos: "1:1"},
				{Op: OpEmitStatic, Value: "id", Pos: "1:7"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			builder := NewInstructionBuilder(ctx)

			err := builder.ProcessTokens(tt.tokens)
			require.NoError(t, err)

			// Finalize を呼ばずに生の命令列を取得（マージ前）
			assert.Equal(t, tt.expected, builder.instructions, "processed instructions should match expected")
		})
	}
}

func TestProcessTokensSkipLeadingTrivia(t *testing.T) {
	tokens := []tokenizer.Token{
		{Type: tokenizer.WHITESPACE, Value: "\n\n", Position: tokenizer.Position{Line: 1, Column: 1}},
		{Type: tokenizer.LINE_COMMENT, Value: "-- comment", Position: tokenizer.Position{Line: 3, Column: 1}},
		{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 4, Column: 1}},
		{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 4, Column: 7}},
		{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 4, Column: 8}},
	}

	ctx := NewGenerationContext(snapsql.DialectPostgres)
	builder := NewInstructionBuilder(ctx)

	err := builder.ProcessTokens(tokens, WithSkipLeadingTrivia())
	require.NoError(t, err)

	require.NotEmpty(t, builder.instructions)
	assert.Equal(t, "SELECT", builder.instructions[0].Value)
}

func TestSkipLeadingTriviaKeepsDirectiveComments(t *testing.T) {
	directive := &tokenizer.Directive{Type: "if", Condition: "cond"}
	tokens := []tokenizer.Token{
		{Type: tokenizer.LINE_COMMENT, Value: "/*# if cond */", Directive: directive},
		{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
	}

	trimmed := skipLeadingTrivia(tokens)
	assert.Equal(t, tokens, trimmed)
}

// TestFinalizeWithOptimization は Finalize による最適化の統合テスト
func TestFinalizeWithOptimization(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		expected []Instruction
	}{
		{
			name: "whitespace merge + static instruction merge",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.WHITESPACE, Value: "  ", Position: tokenizer.Position{Line: 1, Column: 8}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 1, Column: 10}},
				{Type: tokenizer.BLOCK_COMMENT, Value: "/* comment */", Position: tokenizer.Position{Line: 1, Column: 12}},
				{Type: tokenizer.FROM, Value: "FROM", Position: tokenizer.Position{Line: 1, Column: 25}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 29}},
				{Type: tokenizer.IDENTIFIER, Value: "users", Position: tokenizer.Position{Line: 1, Column: 30}},
			},
			expected: []Instruction{
				// ProcessTokens: SELECT, " ", id, " ", FROM, " ", users
				// After merge: "SELECT id FROM users"
				{Op: OpEmitStatic, Value: "SELECT id FROM users", Pos: "1:1"},
			},
		},
		{
			name: "optimized query with system instructions",
			tokens: []tokenizer.Token{
				{Type: tokenizer.SELECT, Value: "SELECT", Position: tokenizer.Position{Line: 1, Column: 1}},
				{Type: tokenizer.WHITESPACE, Value: " ", Position: tokenizer.Position{Line: 1, Column: 7}},
				{Type: tokenizer.WHITESPACE, Value: "\t", Position: tokenizer.Position{Line: 1, Column: 8}},
				{Type: tokenizer.IDENTIFIER, Value: "id", Position: tokenizer.Position{Line: 1, Column: 9}},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			builder := NewInstructionBuilder(ctx)

			err := builder.ProcessTokens(tt.tokens)
			require.NoError(t, err)

			// Finalize を呼んで最適化を実行
			result := builder.Finalize()

			assert.Equal(t, tt.expected, result, "finalized instructions should match expected")
		})
	}
}

// TestConditionalDirective は条件分岐ディレクティブ (if/elseif/else/end) のテスト
func TestConditionalDirective(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		expectedExpressions  []CELExpression
	}{
		{
			name: "simple if/end within WHERE clause",
			sql: `/*# parameters: { include_age_filter: bool } */
SELECT id, name FROM users
WHERE active = true
/*# if include_age_filter */
    AND age >= 18
/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE active = true ", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "4:1"},
				{Op: OpEmitStatic, Value: " AND age >= 18 ", Pos: "5:0"},
				{Op: OpEnd, Pos: "6:1"},
				{Op: OpBoundary, Pos: "0:0"}, // WHERE 句が END で終わるので BOUNDARY を追加
			},
			expectedExpressions: []CELExpression{{Expression: "include_age_filter"}},
		},
		{
			name: "if/else/end within WHERE clause",
			sql: `/*# parameters: { use_premium: bool } */
SELECT id, name FROM users
WHERE /*# if use_premium */premium = true/*# else */active = true/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE ", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "3:7"},
				{Op: OpEmitStatic, Value: "premium = true", Pos: "3:28"},
				{Op: OpElse, Pos: "3:42"},
				{Op: OpEmitStatic, Value: "active = true", Pos: "3:53"},
				{Op: OpEnd, Pos: "3:66"},
				{Op: OpBoundary, Pos: "0:0"}, // WHERE 句が END で終わるので BOUNDARY を追加
			},
			expectedExpressions: []CELExpression{{Expression: "use_premium"}},
		},
		{
			name: "if/elseif/else/end within WHERE clause",
			sql: `/*# parameters: { priority: int } */
SELECT id, name FROM tasks
WHERE status = 'open'
/*# if priority == 1 */
    AND urgency = 'critical'
/*# elseif priority == 2 */
    AND urgency = 'high'
/*# else */
    AND urgency = 'normal'
/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name FROM tasks WHERE status = 'open' ", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "4:1"},
				{Op: OpEmitStatic, Value: " AND urgency = 'critical' ", Pos: "5:0"},
				{Op: OpElseIf, ExprIndex: ptr(1), Pos: "6:1"},
				{Op: OpEmitStatic, Value: " AND urgency = 'high' ", Pos: "7:0"},
				{Op: OpElse, Pos: "8:1"},
				{Op: OpEmitStatic, Value: " AND urgency = 'normal' ", Pos: "9:0"},
				{Op: OpEnd, Pos: "10:1"},
				{Op: OpBoundary, Pos: "0:0"}, // WHERE 句が END で終わるので BOUNDARY を追加
			},
			expectedExpressions: []CELExpression{
				{Expression: "priority == 1"},
				{Expression: "priority == 2"},
			},
		},
		{
			name: "entire WHERE clause wrapped in conditional - skipped (parser evaluates condition)",
			sql: `/*# parameters: { apply_filter: bool } */
SELECT id, name FROM users
/*# if apply_filter */
WHERE active = true
/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				// パーサーが条件を評価してWHERE句を含めるため、
				// 実際の出力は WHERE句が常に含まれる
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE active = true ", Pos: "2:1"},
				// WHERE 句はない（パーサーがすでに評価済み）ので BOUNDARY は不要
			},
			expectedExpressions: []CELExpression{{Expression: "apply_filter"}},
		},
		{
			name: "nested conditionals within WHERE clause",
			sql: `/*# parameters: { filter_active: bool, filter_verified: bool } */
SELECT id, name FROM users
WHERE 1=1
/*# if filter_active */
    AND active = true
    /*# if filter_verified */
        AND verified = true
    /*# end */
/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE 1=1 ", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "4:1"},
				{Op: OpEmitStatic, Value: " AND active = true ", Pos: "5:0"},
				{Op: OpIf, ExprIndex: ptr(1), Pos: "6:5"},
				{Op: OpEmitStatic, Value: " AND verified = true ", Pos: "7:0"},
				{Op: OpEnd, Pos: "8:5"},                    // 内側の end
				{Op: OpEmitStatic, Value: " ", Pos: "9:0"}, // 余分な空白（最適化で削除される可能性あり）
				{Op: OpEnd, Pos: "9:1"},                    // 外側の end
				{Op: OpBoundary, Pos: "0:0"},               // WHERE 句が END で終わるので BOUNDARY を追加
			},
			expectedExpressions: []CELExpression{
				{Expression: "filter_active"},
				{Expression: "filter_verified"},
			},
		},
		{
			name: "conditional with variable directive",
			sql: `/*# parameters: { has_filter: bool, min_age: int } */
SELECT id, name FROM users
WHERE 1=1
/*# if has_filter */
    AND age >= /*= min_age */18
/*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE 1=1 ", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "4:1"},
				{Op: OpEmitStatic, Value: " AND age >= ", Pos: "5:0"},
				{Op: OpEmitEval, ExprIndex: ptr(1), Pos: "5:16"},
				{Op: OpEmitStatic, Value: " ", Pos: "6:0"}, // 余分な空白
				{Op: OpEnd, Pos: "6:1"},
				{Op: OpBoundary, Pos: "0:0"}, // WHERE 句が END で終わるので BOUNDARY を追加
			},
			expectedExpressions: []CELExpression{
				{Expression: "has_filter"},
				{Expression: "min_age"},
			},
		},
		{
			name: "conditional in SELECT clause",
			sql: `/*# parameters: { include_email: bool } */
SELECT 
    id,
    name/*# if include_email */,
    email/*# end */
FROM users`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, name", Pos: "2:1"},
				{Op: OpIf, ExprIndex: ptr(0), Pos: "4:9"},
				{Op: OpEmitStatic, Value: ", email", Pos: "4:32"},
				{Op: OpEnd, Pos: "5:10"},
				{Op: OpEmitStatic, Value: " FROM users", Pos: "6:0"},
				// WHERE 句がないので BOUNDARY は不要
			},
			expectedExpressions: []CELExpression{{Expression: "include_email"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL - パーサーがディレクティブを処理するために定数を提供
			constants := map[string]any{
				"include_age_filter": true,
				"use_premium":        true,
				"priority":           1,
				"apply_filter":       true,
				"filter_active":      true,
				"filter_verified":    true,
				"has_filter":         true,
				"min_age":            18,
				"include_email":      true,
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext
			ctx := NewGenerationContext(tt.dialect)

			// Generate instructions
			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "GenerateSelectInstructions should succeed")

			// 命令の検証
			// OpIfSystemLimit 以降をカットして比較
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			if !assert.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps), "Instruction count mismatch") {
				t.Logf("Expected %d instructions, got %d", len(tt.expectedInstructions), len(actualBeforeSystemOps))

				for i, instr := range actualBeforeSystemOps {
					t.Logf("  [%d] %+v", i, instr)
				}
			}

			for i, expected := range tt.expectedInstructions {
				if i >= len(actualBeforeSystemOps) {
					break
				}

				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Instruction[%d] Op mismatch", i)
				assert.Equal(t, expected.Pos, actual.Pos, "Instruction[%d] Pos mismatch", i)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Instruction[%d] Value mismatch", i)
				}

				if expected.ExprIndex != nil {
					require.NotNil(t, actual.ExprIndex, "Instruction[%d] ExprIndex is nil", i)
					assert.Equal(t, *expected.ExprIndex, *actual.ExprIndex, "Instruction[%d] ExprIndex mismatch", i)
				}
			}

			// 式の検証
			assert.Equal(t, len(tt.expectedExpressions), len(expressions), "Expression count mismatch")

			for i, expected := range tt.expectedExpressions {
				if i >= len(expressions) {
					break
				}

				assert.Equal(t, expected.Expression, expressions[i].Expression, "Expression[%d] Expression mismatch", i)
				assert.Equal(t, expected.EnvironmentIndex, expressions[i].EnvironmentIndex, "Expression[%d] EnvironmentIndex mismatch", i)
				assert.NotEmpty(t, expressions[i].ID, "Expression[%d] ID should not be empty", i)
			}
		})
	}
}

// TestVariableDirective は変数ディレクティブ (/*= expression */) のテスト
func TestVariableDirective(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		expectedExpressions  []CELExpression
	}{
		{
			name: "simple variable in WHERE clause",
			sql: `/*# parameters: { user_id: int} */
			SELECT id FROM users WHERE user_id = /*= user_id */1`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE user_id = ", Pos: "2:4"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:41"}, // Variable directive position
			},
			expectedExpressions: []CELExpression{{Expression: "user_id"}},
		},
		{
			name: "multiple variables in same query",
			sql: `/*# parameters: { status: string, min_priority: int } */
SELECT id FROM tasks WHERE status = /*= status */'active' AND priority >= /*= min_priority */1`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM tasks WHERE status = ", Pos: "2:1"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:37"},
				{Op: OpEmitStatic, Value: " AND priority >= ", Pos: "2:58"},
				{Op: OpEmitEval, ExprIndex: ptr(1), Pos: "2:75"},
			},
			expectedExpressions: []CELExpression{{Expression: "status"}, {Expression: "min_priority"}},
		},
		{
			name: "variable in SELECT clause",
			sql: `/*# parameters: { field_name: string } */
SELECT /*= field_name */'id' FROM users`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT ", Pos: "2:1"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:8"},
				{Op: OpEmitStatic, Value: " FROM users", Pos: "2:29"},
			},
			expectedExpressions: []CELExpression{{Expression: "field_name"}},
		},
		{
			name: "variable with object field access",
			sql: `/*# parameters: { user: { department_id: int } } */
SELECT id FROM users WHERE department_id = /*= user.department_id */1`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE department_id = ", Pos: "2:1"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:44"},
			},
			expectedExpressions: []CELExpression{{Expression: "user.department_id"}},
		},
		{
			name: "duplicate expressions reuse index",
			sql: `/*# parameters: { status: string } */
SELECT id FROM users WHERE status = /*= status */'active' OR priority_status = /*= status */'high'`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE status = ", Pos: "2:1"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:37"}, // First occurrence
				{Op: OpEmitStatic, Value: " OR priority_status = ", Pos: "2:58"},
				{Op: OpEmitEval, ExprIndex: ptr(0), Pos: "2:80"}, // Reuses same index
			},
			expectedExpressions: []CELExpression{{Expression: "status"}}, // Only one expression
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL - パーサーがディレクティブを処理するために定数を提供
			constants := map[string]any{
				"user_id":      1,
				"status":       "active",
				"min_priority": 1,
				"field_name":   "id",
				"start_date":   "2024-01-01",
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Debug: print tokens to see actual structure
			t.Logf("Clauses count: %d", len(stmt.Clauses()))

			for clauseIdx, clause := range stmt.Clauses() {
				if clause.Type() == parser.WHERE_CLAUSE {
					tokens := clause.RawTokens() // Use RawTokens instead of ContentTokens
					t.Logf("Clause[%d] has %d RawTokens", clauseIdx, len(tokens))

					for i, tok := range tokens {
						if tok.Directive != nil {
							t.Logf("  Token[%d]: Type=%s Value=%q Directive={Type:%s Condition:%q}",
								i, tok.Type, tok.Value, tok.Directive.Type, tok.Directive.Condition)
						}
					}
				}
			}

			// Create GenerationContext
			ctx := NewGenerationContext(tt.dialect)

			// Generate instructions
			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "GenerateSelectInstructions should succeed")

			// 命令の検証
			// OpIfSystemLimit 以降をカットして比較
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			if !assert.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps), "Instruction count mismatch") {
				t.Logf("Expected %d instructions, got %d", len(tt.expectedInstructions), len(actualBeforeSystemOps))

				for i, instr := range actualBeforeSystemOps {
					t.Logf("  [%d] %+v", i, instr)
				}
			}

			for i, expected := range tt.expectedInstructions {
				if i >= len(actualBeforeSystemOps) {
					break
				}

				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Instruction[%d] Op mismatch", i)
				assert.Equal(t, expected.Pos, actual.Pos, "Instruction[%d] Pos mismatch", i)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Instruction[%d] Value mismatch", i)
				}

				if expected.ExprIndex != nil {
					require.NotNil(t, actual.ExprIndex, "Instruction[%d] ExprIndex is nil", i)
					assert.Equal(t, *expected.ExprIndex, *actual.ExprIndex, "Instruction[%d] ExprIndex mismatch", i)
				}
			}

			// 式の検証
			assert.Equal(t, len(tt.expectedExpressions), len(expressions), "Expression count mismatch")

			for i, expected := range tt.expectedExpressions {
				if i >= len(expressions) {
					break
				}

				assert.Equal(t, expected.Expression, expressions[i].Expression, "Expression[%d] mismatch", i)
				assert.Equal(t, expected.EnvironmentIndex, expressions[i].EnvironmentIndex, "Expression[%d] EnvironmentIndex mismatch", i)
				// ID is auto-generated, so we don't check it in the expected value
				assert.NotEmpty(t, expressions[i].ID, "Expression[%d] ID should not be empty", i)
			}
		})
	}
} // TestDialectTimeFunctionConversion は方言による時間関数の変換をテスト
// TestDialectConversions は各種の方言による変換をテスト（時間関数、CAST、日時関数、真偽値、文字列連結）
func TestDialectConversions(t *testing.T) {
	tests := []struct {
		category             string
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
	}{
		// === Time Function Conversion ===
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for PostgreSQL",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for MySQL",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for MariaDB",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  snapsql.DialectMariaDB,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP stays for SQLite",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURRENT_TIMESTAMP FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() to CURRENT_TIMESTAMP for SQLite",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURRENT_TIMESTAMP FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() stays for PostgreSQL",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() stays for MySQL",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "time function in WHERE clause",
			sql:      "SELECT id FROM orders WHERE created_at > NOW()",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM orders WHERE created_at > NOW()", Pos: "1:1"},
			},
		},
		// === CAST Conversion ===
		{
			category: "cast",
			name:     "CAST to PostgreSQL :: syntax",
			sql:      "SELECT id, CAST(created_at AS TEXT) FROM users",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, (created_at)::TEXT FROM users", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST stays for MySQL",
			sql:      "SELECT id, CAST(created_at AS CHAR) FROM users",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CAST(created_at AS CHAR) FROM users", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST stays for SQLite",
			sql:      "SELECT id, CAST(amount AS INTEGER) FROM orders",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CAST(amount AS INTEGER) FROM orders", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST in WHERE clause to PostgreSQL",
			sql:      "SELECT id FROM users WHERE CAST(age AS TEXT) = '25'",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE (age)::TEXT = '25'", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "multiple CASTs to PostgreSQL",
			sql:      "SELECT CAST(id AS TEXT), CAST(price AS DECIMAL) FROM products",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT (id)::TEXT, (price)::DECIMAL FROM products", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "Double colon cast to CAST for MySQL",
			sql:      "SELECT (created_at)::DATETIME FROM logs",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CAST(created_at AS DATETIME) FROM logs", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "Double colon without parentheses to CAST",
			sql:      "SELECT amount::DECIMAL(10,2) FROM invoices",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CAST(amount AS DECIMAL(10,2)) FROM invoices", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "Double colon multi word type",
			sql:      "SELECT value::DOUBLE PRECISION FROM stats",
			dialect:  snapsql.DialectMariaDB,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CAST(value AS DOUBLE PRECISION) FROM stats", Pos: "1:1"},
			},
		},
		// === DateTime Conversion ===
		{
			category: "datetime",
			name:     "CURDATE to CURRENT_DATE for PostgreSQL",
			sql:      "SELECT CURDATE() FROM users",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_DATE FROM users", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURTIME to CURRENT_TIME for MySQL",
			sql:      "SELECT id, CURTIME() FROM orders",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURTIME() FROM orders", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURRENT_DATE stays for MySQL",
			sql:      "SELECT id FROM logs WHERE date = CURRENT_DATE",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM logs WHERE date = CURRENT_DATE", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURDATE to CURRENT_DATE for SQLite",
			sql:      "SELECT CURDATE() FROM users WHERE active = 1",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_DATE FROM users WHERE active = 1", Pos: "1:1"},
			},
		},
		// === Boolean Conversion ===
		{
			category: "boolean",
			name:     "PostgreSQL TRUE to 1 for MySQL",
			sql:      "SELECT id FROM users WHERE active = TRUE",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE active = 1", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "PostgreSQL FALSE to 0 for SQLite",
			sql:      "SELECT id FROM items WHERE deleted = FALSE",
			dialect:  snapsql.DialectSQLite,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM items WHERE deleted = 0", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "TRUE stays for PostgreSQL",
			sql:      "SELECT id FROM records WHERE flag = TRUE",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM records WHERE flag = TRUE", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "Multiple TRUE/FALSE conversions",
			sql:      "SELECT id FROM logs WHERE success = TRUE AND archived = FALSE",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM logs WHERE success = 1 AND archived = 0", Pos: "1:1"},
			},
		},
		// === String Concatenation Conversion ===
		{
			category: "concat",
			name:     "CONCAT to || for PostgreSQL",
			sql:      "SELECT CONCAT(first_name, ' ', last_name) FROM users",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT first_name || ' ' || last_name FROM users", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "CONCAT stays for MySQL",
			sql:      "SELECT CONCAT(city, ', ', state) FROM locations",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CONCAT(city, ', ', state) FROM locations", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "CONCAT with multiple arguments",
			sql:      "SELECT CONCAT(a, b, c, d) FROM table1",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT a || b || c || d FROM table1", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "Multiple CONCAT functions",
			sql:      "SELECT CONCAT(a, b), CONCAT(c, d) FROM table1",
			dialect:  snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT a || b, c || d FROM table1", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "Double pipe to CONCAT for MySQL",
			sql:      "SELECT first_name || ' ' || last_name FROM users",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CONCAT(first_name, ' ', last_name) FROM users", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "Nested pipe expressions flatten",
			sql:      "SELECT (a || b) || c FROM items",
			dialect:  snapsql.DialectMySQL,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CONCAT(a, b, c) FROM items", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "Pipe inside parentheses on right",
			sql:      "SELECT col1 || (col2 || col3) AS merged FROM t",
			dialect:  snapsql.DialectMariaDB,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CONCAT(col1, col2, col3) AS merged FROM t", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
			// Parse SQL
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext with specific dialect
			ctx := NewGenerationContext(tt.dialect)

			// Generate instructions
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "GenerateSelectInstructions should succeed")

			// 命令の検証
			// OpIfSystemLimit 以降をカットして比較
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			if !assert.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps), "Instruction count mismatch for dialect %s", tt.dialect) {
				t.Logf("Expected %d instructions, got %d", len(tt.expectedInstructions), len(actualBeforeSystemOps))

				for i, instr := range actualBeforeSystemOps {
					t.Logf("  [%d] %+v", i, instr)
				}
			}

			for i, expected := range tt.expectedInstructions {
				if i >= len(actualBeforeSystemOps) {
					break
				}

				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Instruction[%d] Op mismatch for dialect %s", i, tt.dialect)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Instruction[%d] Value mismatch for dialect %s\nExpected: %q\nActual: %q", i, tt.dialect, expected.Value, actual.Value)
				}
			}
		})
	}
}

// ptr is a helper function to create an int pointer
func ptr(i int) *int {
	return &i
}

// TestForDirectiveTableDriven tests for loop directive (/*# for variable : expression */) using table-driven approach
// This test parses complete SQL files and validates the instruction generation, expressions, and environments
// Constants are intentionally left empty to test type inference with dummy values
// TestForDirectiveTableDriven は for ループディレクティブの命令生成をテスト
func TestForDirectiveTableDriven(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
		expectedExpressions  []CELExpression
	}{
		{
			name: "simple for loop with variable reference",
			sql: `/*#
parameters:
  users:
    - id: int
      tags:
        - string
*/
INSERT INTO user_tags (user_id, tag) 
VALUES /*# for user : users */(/*= user.id */, /*= user.tags */) /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO user_tags (user_id, tag) VALUES "},
				{Op: OpLoopStart, ExprIndex: ptr(0), EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: "("},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: ", "},
				{Op: OpEmitEval, ExprIndex: ptr(2)},
				{Op: OpEmitStatic, Value: ") "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
			},
			expectedExpressions: []CELExpression{
				{Expression: "users", EnvironmentIndex: 0},
				{Expression: "user.id", EnvironmentIndex: 1},
				{Expression: "user.tags", EnvironmentIndex: 1},
			},
		},
		{
			name: "nested for loops",
			sql: `/*#
parameters:
  users:
    - id: int
      tags:
        - string
*/
INSERT INTO user_details (user_id, tag) 
VALUES 
  /*# for user : users */
    /*# for tag : user.tags */
      (/*= user.id */, /*= tag */)
    /*# end */
  /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO user_details (user_id, tag) VALUES "},
				{Op: OpLoopStart, ExprIndex: ptr(0), EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: " "},
				{Op: OpLoopStart, ExprIndex: ptr(1), EnvIndex: ptr(2), CollectionExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: " ("},
				{Op: OpEmitEval, ExprIndex: ptr(2)},
				{Op: OpEmitStatic, Value: ", "},
				{Op: OpEmitEval, ExprIndex: ptr(3)},
				{Op: OpEmitStatic, Value: ") "},
				{Op: OpLoopEnd, EnvIndex: ptr(1)},
				{Op: OpEmitStatic, Value: " "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
			},
			expectedExpressions: []CELExpression{
				{Expression: "users", EnvironmentIndex: 0},
				{Expression: "user.tags", EnvironmentIndex: 0},
				{Expression: "user.id", EnvironmentIndex: 2},
				{Expression: "tag", EnvironmentIndex: 2},
			},
		},
		{
			name: "for loop with conditional inside",
			sql: `/*#
parameters:
  users:
    - id: int
      tags:
        - string
*/
INSERT INTO user_summary (user_id, summary) 
VALUES 
  /*# for user : users */
    (
      /*= user.id */,
      /*# if user.id > 0 */
        'active'
      /*# else */
        'inactive'
      /*# end */
    )
  /*# end */`,
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "INSERT INTO user_summary (user_id, summary) VALUES "},
				{Op: OpLoopStart, ExprIndex: ptr(0), EnvIndex: ptr(1), CollectionExprIndex: ptr(0)},
				{Op: OpEmitStatic, Value: " ( "},
				{Op: OpEmitEval, ExprIndex: ptr(1)},
				{Op: OpEmitStatic, Value: ", "},
				{Op: OpIf, ExprIndex: ptr(2)},
				{Op: OpEmitStatic, Value: " 'active' "},
				{Op: OpElse},
				{Op: OpEmitStatic, Value: " 'inactive' "},
				{Op: OpLoopEnd, EnvIndex: ptr(0)},
				{Op: OpEmitStatic, Value: " ) "},
				{Op: OpEnd},
			},
			expectedExpressions: []CELExpression{
				{Expression: "users", EnvironmentIndex: 0},
				{Expression: "user.id", EnvironmentIndex: 1},
				{Expression: "user.id > 0", EnvironmentIndex: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL with parameter definitions from SQL comment
			// Provide dummy data in constants for type inference
			constants := map[string]any{
				"users": []any{
					map[string]any{
						"id":   int64(1),
						"tags": []any{"tag1", "tag2"},
					},
				},
			}
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext (root environment is automatically initialized)
			ctx := NewGenerationContext(tt.dialect)

			// Generate instructions from the parsed statement
			builder := NewInstructionBuilder(ctx)

			// Process all clauses in the statement
			for _, clause := range stmt.Clauses() {
				tokens := clause.RawTokens()
				err := builder.ProcessTokens(tokens)
				require.NoError(t, err, "ProcessTokens should succeed for clause type %s", clause.Type())
			}

			// Merge static instructions and get final result
			instructions := builder.mergeStaticInstructions()

			// 命令の検証
			// OpIfSystemLimit 以降をカットして比較
			cutIndex := len(instructions)
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit {
					cutIndex = i
					break
				}
			}

			actualBeforeSystemOps := instructions[:cutIndex]

			if !assert.Equal(t, len(tt.expectedInstructions), len(actualBeforeSystemOps), "Instruction count mismatch") {
				t.Logf("Expected %d instructions, got %d", len(tt.expectedInstructions), len(actualBeforeSystemOps))

				for i, instr := range actualBeforeSystemOps {
					t.Logf("  [%d] Op=%s Value=%q ExprIndex=%v EnvIndex=%v", i, instr.Op, instr.Value, instr.ExprIndex, instr.EnvIndex)
				}
			} else {
				// Log actual instructions for debugging
				for i, instr := range actualBeforeSystemOps {
					var collExprIdx, exprIdx, envIdx string

					if instr.CollectionExprIndex != nil {
						collExprIdx = strconv.Itoa(*instr.CollectionExprIndex)
					} else {
						collExprIdx = "<nil>"
					}

					if instr.ExprIndex != nil {
						exprIdx = strconv.Itoa(*instr.ExprIndex)
					} else {
						exprIdx = "<nil>"
					}

					if instr.EnvIndex != nil {
						envIdx = strconv.Itoa(*instr.EnvIndex)
					} else {
						envIdx = "<nil>"
					}

					t.Logf("  [%d] Op=%s Value=%q CollectionExprIndex=%s ExprIndex=%s EnvIndex=%s", i, instr.Op, instr.Value, collExprIdx, exprIdx, envIdx)
				}
			}

			for i, expected := range tt.expectedInstructions {
				if i >= len(actualBeforeSystemOps) {
					break
				}

				actual := actualBeforeSystemOps[i]
				assert.Equal(t, expected.Op, actual.Op, "Instruction[%d] Op mismatch", i)

				if expected.Value != "" {
					assert.Equal(t, expected.Value, actual.Value, "Instruction[%d] Value mismatch\nExpected: %q\nActual: %q", i, expected.Value, actual.Value)
				}

				// Check ExprIndex if specified (for OpLoopStart, this maps to CollectionExprIndex)
				if expected.ExprIndex != nil {
					if actual.Op == OpLoopStart {
						require.NotNil(t, actual.CollectionExprIndex, "Instruction[%d] CollectionExprIndex (for LOOP_START) should not be nil", i)
						assert.Equal(t, *expected.ExprIndex, *actual.CollectionExprIndex, "Instruction[%d] CollectionExprIndex mismatch", i)
					} else {
						require.NotNil(t, actual.ExprIndex, "Instruction[%d] ExprIndex should not be nil", i)
						assert.Equal(t, *expected.ExprIndex, *actual.ExprIndex, "Instruction[%d] ExprIndex mismatch", i)
					}
				}

				// Check EnvIndex if specified
				if expected.EnvIndex != nil {
					require.NotNil(t, actual.EnvIndex, "Instruction[%d] EnvIndex should not be nil", i)
					assert.Equal(t, *expected.EnvIndex, *actual.EnvIndex, "Instruction[%d] EnvIndex mismatch", i)
				}
			}

			// 式の検証
			assert.Equal(t, len(tt.expectedExpressions), len(ctx.Expressions), "Expression count mismatch")

			for i, expected := range tt.expectedExpressions {
				if i >= len(ctx.Expressions) {
					break
				}

				actual := ctx.Expressions[i]
				assert.Equal(t, expected.Expression, actual.Expression, "Expression[%d] Expression mismatch", i)
				assert.Equal(t, expected.EnvironmentIndex, actual.EnvironmentIndex, "Expression[%d] EnvironmentIndex mismatch", i)
				assert.NotEmpty(t, actual.ID, "Expression[%d] ID should not be empty", i)
			}
		})
	}
}
