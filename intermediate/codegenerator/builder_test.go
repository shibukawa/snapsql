package codegenerator

import (
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
			ctx := &GenerationContext{Dialect: string(snapsql.DialectPostgres)}
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
			ctx := &GenerationContext{Dialect: string(snapsql.DialectPostgres)}
			builder := NewInstructionBuilder(ctx)

			err := builder.ProcessTokens(tt.tokens)
			require.NoError(t, err)

			// Finalize を呼ばずに生の命令列を取得（マージ前）
			assert.Equal(t, tt.expected, builder.instructions, "processed instructions should match expected")
		})
	}
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
			ctx := &GenerationContext{Dialect: string(snapsql.DialectPostgres)}
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
		dialect              string
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				// パーサーが条件を評価してWHERE句を含めるため、
				// 実際の出力は WHERE句が常に含まれる
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users WHERE active = true ", Pos: "2:1"},
				// WHERE 句はない（パーサーがすでに評価済み）ので BOUNDARY は不要
			},
			expectedExpressions: []CELExpression{},
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			constants := map[string]interface{}{
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
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext
			ctx := &GenerationContext{
				Dialect:      tt.dialect,
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

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
			}
		})
	}
}

// TestVariableDirective は変数ディレクティブ (/*= expression */) のテスト
func TestVariableDirective(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              string
		expectedInstructions []Instruction
		expectedExpressions  []CELExpression
	}{
		{
			name: "simple variable in WHERE clause",
			sql: `/*# parameters: { user_id: int} */
			SELECT id FROM users WHERE user_id = /*= user_id */1`,
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			dialect: string(snapsql.DialectPostgres),
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
			constants := map[string]interface{}{
				"user_id":      1,
				"status":       "active",
				"min_priority": 1,
				"field_name":   "id",
				"start_date":   "2024-01-01",
			}

			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
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
			ctx := &GenerationContext{
				Dialect:      tt.dialect,
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

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
		dialect              string
		expectedInstructions []Instruction
	}{
		// === Time Function Conversion ===
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for PostgreSQL",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for MySQL",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP to NOW() for MariaDB",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  string(snapsql.DialectMariaDB),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "CURRENT_TIMESTAMP stays for SQLite",
			sql:      "SELECT id, CURRENT_TIMESTAMP FROM users",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURRENT_TIMESTAMP FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() to CURRENT_TIMESTAMP for SQLite",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURRENT_TIMESTAMP FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() stays for PostgreSQL",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "NOW() stays for MySQL",
			sql:      "SELECT id, NOW() FROM users",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, NOW() FROM users", Pos: "1:1"},
			},
		},
		{
			category: "timefunc",
			name:     "time function in WHERE clause",
			sql:      "SELECT id FROM orders WHERE created_at > NOW()",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM orders WHERE created_at > CURRENT_TIMESTAMP", Pos: "1:1"},
			},
		},
		// === CAST Conversion ===
		{
			category: "cast",
			name:     "CAST to PostgreSQL :: syntax",
			sql:      "SELECT id, CAST(created_at AS TEXT) FROM users",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, (created_at)::TEXT FROM users", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST stays for MySQL",
			sql:      "SELECT id, CAST(created_at AS CHAR) FROM users",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CAST(created_at AS CHAR) FROM users", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST stays for SQLite",
			sql:      "SELECT id, CAST(amount AS INTEGER) FROM orders",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CAST(amount AS INTEGER) FROM orders", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "CAST in WHERE clause to PostgreSQL",
			sql:      "SELECT id FROM users WHERE CAST(age AS TEXT) = '25'",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE (age)::TEXT = '25'", Pos: "1:1"},
			},
		},
		{
			category: "cast",
			name:     "multiple CASTs to PostgreSQL",
			sql:      "SELECT CAST(id AS TEXT), CAST(price AS DECIMAL) FROM products",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT (id)::TEXT, (price)::DECIMAL FROM products", Pos: "1:1"},
			},
		},
		// === DateTime Conversion ===
		{
			category: "datetime",
			name:     "CURDATE to CURRENT_DATE for PostgreSQL",
			sql:      "SELECT CURDATE() FROM users",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_DATE FROM users", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURTIME to CURRENT_TIME for MySQL",
			sql:      "SELECT id, CURTIME() FROM orders",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, CURTIME() FROM orders", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURRENT_DATE stays for MySQL",
			sql:      "SELECT id FROM logs WHERE date = CURRENT_DATE",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM logs WHERE date = CURRENT_DATE", Pos: "1:1"},
			},
		},
		{
			category: "datetime",
			name:     "CURDATE to CURRENT_DATE for SQLite",
			sql:      "SELECT CURDATE() FROM users WHERE active = 1",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_DATE FROM users WHERE active = 1", Pos: "1:1"},
			},
		},
		// === Boolean Conversion ===
		{
			category: "boolean",
			name:     "PostgreSQL TRUE to 1 for MySQL",
			sql:      "SELECT id FROM users WHERE active = TRUE",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE active = 1", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "PostgreSQL FALSE to 0 for SQLite",
			sql:      "SELECT id FROM items WHERE deleted = FALSE",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM items WHERE deleted = 0", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "TRUE stays for PostgreSQL",
			sql:      "SELECT id FROM records WHERE flag = TRUE",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM records WHERE flag = TRUE", Pos: "1:1"},
			},
		},
		{
			category: "boolean",
			name:     "Multiple TRUE/FALSE conversions",
			sql:      "SELECT id FROM logs WHERE success = TRUE AND archived = FALSE",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM logs WHERE success = 1 AND archived = 0", Pos: "1:1"},
			},
		},
		// === String Concatenation Conversion ===
		{
			category: "concat",
			name:     "CONCAT to || for PostgreSQL",
			sql:      "SELECT CONCAT(first_name, ' ', last_name) FROM users",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT first_name || ' ' || last_name FROM users", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "CONCAT stays for MySQL",
			sql:      "SELECT CONCAT(city, ', ', state) FROM locations",
			dialect:  string(snapsql.DialectMySQL),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CONCAT(city, ', ', state) FROM locations", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "CONCAT with multiple arguments",
			sql:      "SELECT CONCAT(a, b, c, d) FROM table1",
			dialect:  string(snapsql.DialectSQLite),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT a || b || c || d FROM table1", Pos: "1:1"},
			},
		},
		{
			category: "concat",
			name:     "Multiple CONCAT functions",
			sql:      "SELECT CONCAT(a, b), CONCAT(c, d) FROM table1",
			dialect:  string(snapsql.DialectPostgres),
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT a || b, c || d FROM table1", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
			// Parse SQL
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext with specific dialect
			ctx := &GenerationContext{
				Dialect:      tt.dialect,
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

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
func TestForDirectiveTableDriven(t *testing.T) {
	tests := []struct {
		name                    string
		sql                     string
		dialect                 string
		expectedHasLoopStart    bool
		expectedHasLoopEnd      bool
		expectedExpressions     []CELExpression
		expectedCELEnvironments []CELEnvironment
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
			dialect:              string(snapsql.DialectPostgres),
			expectedHasLoopStart: true,
			expectedHasLoopEnd:   true,
			expectedExpressions: []CELExpression{
				{Expression: "user.id"},
				{Expression: "user.tags"},
			},
			expectedCELEnvironments: []CELEnvironment{
				{
					AdditionalVariables: []CELVariableInfo{
						{Name: "user", Type: "any"},
					},
					Container: "for user : users",
				},
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
			dialect:              string(snapsql.DialectPostgres),
			expectedHasLoopStart: true,
			expectedHasLoopEnd:   true,
			expectedExpressions: []CELExpression{
				{Expression: "user.id"},
				{Expression: "tag"},
			},
			expectedCELEnvironments: []CELEnvironment{
				{
					AdditionalVariables: []CELVariableInfo{
						{Name: "user", Type: "any"},
					},
					Container: "for user : users",
				},
				{
					AdditionalVariables: []CELVariableInfo{
						{Name: "tag", Type: "any"},
					},
					Container: "for tag : user.tags",
				},
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
			dialect:              string(snapsql.DialectPostgres),
			expectedHasLoopStart: true,
			expectedHasLoopEnd:   true,
			expectedExpressions: []CELExpression{
				{Expression: "user.id"},
				{Expression: "user.id > 0"},
			},
			expectedCELEnvironments: []CELEnvironment{
				{
					AdditionalVariables: []CELVariableInfo{
						{Name: "user", Type: "any"},
					},
					Container: "for user : users",
				},
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
			stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// Create GenerationContext
			ctx := &GenerationContext{
				Dialect:         tt.dialect,
				Expressions:     make([]CELExpression, 0),
				Environments:    make([]string, 0),
				CELEnvironments: make([]CELEnvironment, 0),
			}

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

			// Verify loop start and end instructions exist
			hasLoopStart := false
			hasLoopEnd := false

			for _, instr := range instructions {
				if instr.Op == OpLoopStart {
					hasLoopStart = true
				}

				if instr.Op == OpLoopEnd {
					hasLoopEnd = true
				}
			}

			if tt.expectedHasLoopStart && !hasLoopStart {
				t.Errorf("Expected LOOP_START instruction but not found. Instructions: %v", instructions)
			}

			if tt.expectedHasLoopEnd && !hasLoopEnd {
				t.Errorf("Expected LOOP_END instruction but not found. Instructions: %v", instructions)
			}

			// Verify expected expressions exist
			expressions := ctx.Expressions

			for _, expectedExpr := range tt.expectedExpressions {
				found := false

				for _, expr := range expressions {
					if expr.Expression == expectedExpr.Expression {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected expression '%s' not found. Got expressions: %v", expectedExpr.Expression, expressions)
				}
			}

			// Verify expected CEL environments
			for i, expectedEnv := range tt.expectedCELEnvironments {
				if i >= len(ctx.CELEnvironments) {
					t.Errorf("Expected CEL environment at index %d but not found", i)
					continue
				}

				actualEnv := ctx.CELEnvironments[i]

				// Check container
				if expectedEnv.Container != actualEnv.Container {
					t.Errorf("CEL environment[%d] container mismatch. Expected: %q, Got: %q", i, expectedEnv.Container, actualEnv.Container)
				}

				// Check additional variables
				if len(expectedEnv.AdditionalVariables) != len(actualEnv.AdditionalVariables) {
					t.Errorf("CEL environment[%d] variable count mismatch. Expected: %d, Got: %d", i, len(expectedEnv.AdditionalVariables), len(actualEnv.AdditionalVariables))
					continue
				}

				for j, expectedVar := range expectedEnv.AdditionalVariables {
					actualVar := actualEnv.AdditionalVariables[j]

					if expectedVar.Name != actualVar.Name {
						t.Errorf("CEL environment[%d] variable[%d] name mismatch. Expected: %q, Got: %q", i, j, expectedVar.Name, actualVar.Name)
					}

					if expectedVar.Type != actualVar.Type {
						t.Errorf("CEL environment[%d] variable[%d] type mismatch. Expected: %q, Got: %q", i, j, expectedVar.Type, actualVar.Type)
					}
				}
			}
		})
	}
}

// TestSubqueryInFromClause は FROM 句内のサブクエリーをテストする
func TestSubqueryInFromClause(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		validateInstructions func(t *testing.T, instructions []Instruction) // 命令列の検証
	}{
		{
			name: "simple subquery in FROM",
			sql:  "SELECT u.id FROM (SELECT id FROM users) AS u",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// Instructions に "(SELECT id FROM users)" と ") AS u" が含まれることを確認
				found_open_paren := false
				found_close_paren_as_u := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "(SELECT") {
							found_open_paren = true
						}

						if strings.Contains(instr.Value, ") AS u") {
							found_close_paren_as_u = true
						}
					}
				}

				assert.True(t, found_open_paren, "Should contain '(SELECT' in instructions")
				assert.True(t, found_close_paren_as_u, "Should contain ') AS u' in instructions")
			},
		},
		{
			name: "nested subqueries in FROM",
			sql:  "SELECT id FROM (SELECT id FROM (SELECT id FROM users) AS t1) AS t2",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// ") AS t2" と ") AS t1" が両方含まれることを確認
				found_t2 := false
				found_t1 := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, ") AS t2") {
							found_t2 = true
						}

						if strings.Contains(instr.Value, ") AS t1") {
							found_t1 = true
						}
					}
				}

				assert.True(t, found_t2, "Should contain ') AS t2' in instructions")
				assert.True(t, found_t1, "Should contain ') AS t1' in instructions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parser.ParseSQLFile でパース
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err)
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			// GenerateSelectInstructions で命令と環境を生成
			ctx := &GenerationContext{
				Dialect:      string(snapsql.DialectPostgres),
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証
			if tt.validateInstructions != nil {
				tt.validateInstructions(t, instructions)
			}

			// Expressions が存在することを確認
			assert.NotNil(t, expressions)
		})
	}
}

// TestWhereSubqueryInClause は WHERE 句内の IN サブクエリーをテストする
func TestWhereSubqueryInClause(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		validateInstructions func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "IN with subquery",
			sql:  "SELECT id FROM users WHERE id IN (SELECT user_id FROM orders)",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// IN (SELECT ...) が含まれることを確認
				found_in := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "IN") {
							found_in = true
						}

						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}
					}
				}

				assert.True(t, found_in, "Should contain 'IN' in instructions")
				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
			},
		},
		{
			name: "NOT IN with subquery",
			sql:  "SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM orders)",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// NOT IN (SELECT ...) が含まれることを確認
				found_not_in := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "NOT IN") {
							found_not_in = true
						}

						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}
					}
				}

				assert.True(t, found_not_in, "Should contain 'NOT IN' in instructions")
				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
			},
		},
		{
			name: "EXISTS with subquery",
			sql:  "SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// EXISTS (SELECT ...) が含まれることを確認
				found_exists := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "EXISTS") {
							found_exists = true
						}

						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}
					}
				}

				assert.True(t, found_exists, "Should contain 'EXISTS' in instructions")
				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
			},
		},
		{
			name: "scalar subquery in WHERE",
			sql:  "SELECT id FROM users WHERE status = (SELECT default_status FROM configs LIMIT 1)",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// status = (SELECT ...) が含まれることを確認
				found_equals := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "=") {
							found_equals = true
						}

						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}
					}
				}

				assert.True(t, found_equals, "Should contain '=' in instructions")
				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parser.ParseSQLFile でパース
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err)
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			// GenerateSelectInstructions で命令を生成
			ctx := &GenerationContext{
				Dialect:      string(snapsql.DialectPostgres),
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証
			if tt.validateInstructions != nil {
				tt.validateInstructions(t, instructions)
			}
		})
	}
}

// TestSelectScalarSubquery は SELECT 句内のスカラーサブクエリーをテストする
func TestSelectScalarSubquery(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		validateInstructions func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "scalar subquery in SELECT clause",
			sql:  "SELECT id, (SELECT count(*) FROM orders WHERE user_id = users.id) as order_count FROM users",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// (SELECT count(*) ...) と order_count が含まれることを確認
				found_select := false
				found_count := false
				found_alias := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}

						if strings.Contains(instr.Value, "count") {
							found_count = true
						}

						if strings.Contains(instr.Value, "order_count") {
							found_alias = true
						}
					}
				}

				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
				assert.True(t, found_count, "Should contain 'count' in instructions")
				assert.True(t, found_alias, "Should contain 'order_count' in instructions")
			},
		},
		{
			name: "correlated subquery in SELECT",
			sql:  "SELECT u.id, (SELECT max(order_date) FROM orders WHERE user_id = u.id) as latest FROM users u",
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// (SELECT max ...), latest, user_id, u.id が含まれることを確認
				found_select := false
				found_max := false
				found_alias := false

				for _, instr := range instructions {
					if instr.Op == OpEmitStatic {
						if strings.Contains(instr.Value, "SELECT") {
							found_select = true
						}

						if strings.Contains(instr.Value, "max") {
							found_max = true
						}

						if strings.Contains(instr.Value, "latest") {
							found_alias = true
						}
					}
				}

				assert.True(t, found_select, "Should contain 'SELECT' in instructions")
				assert.True(t, found_max, "Should contain 'max' in instructions")
				assert.True(t, found_alias, "Should contain 'latest' in instructions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parser.ParseSQLFile でパース
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err)
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			// GenerateSelectInstructions で命令を生成
			ctx := &GenerationContext{
				Dialect:      string(snapsql.DialectPostgres),
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証
			if tt.validateInstructions != nil {
				tt.validateInstructions(t, instructions)
			}
		})
	}
}

// TestDirectiveInSubquery はサブクエリー内でディレクティブ（パラメータ参照）が使えることをテストする
func TestDirectiveInSubquery(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		expectedExpressions  int
		validateInstructions func(t *testing.T, instructions []Instruction)
	}{
		{
			name:                "parameter in subquery WHERE",
			sql:                 `/*# parameters: { status_val: int } */SELECT id FROM users WHERE id IN (SELECT user_id FROM orders WHERE status = /*= status_val */1)`,
			expectedExpressions: 1,
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// サブクエリー内の /*= status_val */ が変数ディレクティブとして認識されることを確認
				// Instructions に OpEmitEval が含まれることを確認
				found_emit_eval := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitEval {
						found_emit_eval = true
					}

					if instr.Op == OpEmitStatic && strings.Contains(instr.Value, "SELECT") {
						found_select = true
					}
				}

				assert.True(t, found_select, "Should contain SELECT in instructions")
				assert.True(t, found_emit_eval, "Should contain OpEmitEval for /*= status_val */ directive")
			},
		},
		{
			name:                "parameter in scalar subquery",
			sql:                 `/*# parameters: { status_val: int } */SELECT id, (SELECT count(*) FROM orders WHERE user_id = users.id AND status = /*= status_val */1) as completed_count FROM users`,
			expectedExpressions: 1,
			validateInstructions: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// スカラーサブクエリー内の /*= status_val */ が変数ディレクティブとして認識されることを確認
				found_emit_eval := false
				found_select := false

				for _, instr := range instructions {
					if instr.Op == OpEmitEval {
						found_emit_eval = true
					}

					if instr.Op == OpEmitStatic && strings.Contains(instr.Value, "SELECT") {
						found_select = true
					}
				}

				assert.True(t, found_select, "Should contain SELECT in instructions")
				assert.True(t, found_emit_eval, "Should contain OpEmitEval for /*= status_val */ directive")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parser.ParseSQLFile でパース
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err)
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			// GenerateSelectInstructions で命令を生成
			ctx := &GenerationContext{
				Dialect:      string(snapsql.DialectPostgres),
				Expressions:  make([]CELExpression, 0),
				Environments: make([]string, 0),
			}

			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)
			assert.GreaterOrEqual(t, len(expressions), tt.expectedExpressions, "Should have at least %d expressions", tt.expectedExpressions)

			// 命令列の検証
			if tt.validateInstructions != nil {
				tt.validateInstructions(t, instructions)
			}
		})
	}
}
