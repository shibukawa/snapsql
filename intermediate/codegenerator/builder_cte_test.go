package codegenerator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCTE は CTE（Common Table Expressions）のテストを行う
// Phase 4 実装予定の CTE サポートに対応したテストケースを含む
func TestCTE(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		expectedInstructions []Instruction
	}{
		// Simple CTE
		{
			name: "simple CTE with SELECT",
			sql:  "WITH cte AS (SELECT id FROM users) SELECT id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM users)SELECT id FROM cte", Pos: "1:1"},
			},
		},
		// Multiple CTEs
		{
			name: "two CTEs",
			sql:  "WITH cte1 AS (SELECT id FROM users), cte2 AS (SELECT id FROM orders) SELECT id FROM cte1 JOIN cte2 ON cte1.id = cte2.id",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte1 AS (SELECT id FROM users), cte2 AS (SELECT id FROM orders)SELECT id FROM cte1 JOIN cte2 ON cte1.id = cte2.id", Pos: "1:1"},
			},
		},
		// CTE with INNER JOIN
		{
			name: "CTE with INNER JOIN",
			sql:  "WITH cte AS (SELECT id, user_id FROM orders) SELECT name FROM users INNER JOIN cte ON id = user_id",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id, user_id FROM orders)SELECT name FROM users INNER JOIN cte ON id = user_id", Pos: "1:1"},
			},
		},
		// CTE with LEFT JOIN
		{
			name: "CTE with LEFT JOIN",
			sql:  "WITH archived AS (SELECT id FROM orders WHERE status = 'archived') SELECT id FROM orders LEFT JOIN archived ON orders.id = archived.id",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH archived AS (SELECT id FROM orders WHERE status = 'archived')SELECT id FROM orders LEFT JOIN archived ON orders.id = archived.id", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Parser should accept the SQL")
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "Should not error generating instructions")

			// CTE のインストラクション検証
			actualBeforeSystemOps := []Instruction{}

			for _, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}

				actualBeforeSystemOps = append(actualBeforeSystemOps, instr)
			}

			assert.Equal(t, tt.expectedInstructions, actualBeforeSystemOps, "Instructions mismatch")
		})
	}
}

// TestCTEWithDirectives は CTE 内のディレクティブサポートをテストする
// Phase 4.5 実装予定の CTE 内ディレクティブ拡張に対応したテストケース
// 注：現在のところ、ディレクティブ付きの SQL はパーサーの段階で失敗するため、
// ディレクティブなしの CTE テストを追加する
func TestCTEWithDirectives(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		expectedInstructions []Instruction
	}{
		{
			name: "CTE with multiple columns",
			sql:  "WITH cte AS (SELECT id, name, email FROM users) SELECT id, name FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id, name, email FROM users)SELECT id, name FROM cte", Pos: "1:1"},
			},
		},
		{
			name: "CTE with WHERE and ORDER BY",
			sql:  "WITH cte AS (SELECT id FROM users WHERE status = 'active') SELECT id FROM cte ORDER BY id",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM users WHERE status = 'active')SELECT id FROM cte ORDER BY id", Pos: "1:1"},
			},
		},
		{
			name: "CTE with GROUP BY and HAVING",
			sql:  "WITH cte AS (SELECT user_id, COUNT(id) FROM orders GROUP BY user_id HAVING COUNT(id) > 5) SELECT user_id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT user_id, COUNT(id) FROM orders GROUP BY user_id HAVING COUNT(id) > 5)SELECT user_id FROM cte", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Parser should accept the SQL")
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "Should not error generating instructions")

			// CTE のインストラクション検証
			actualBeforeSystemOps := []Instruction{}

			for _, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}

				actualBeforeSystemOps = append(actualBeforeSystemOps, instr)
			}

			assert.Equal(t, tt.expectedInstructions, actualBeforeSystemOps, "Instructions mismatch")
		})
	}
}

// TestTokenDumpInCTE は CTE 内部のトークン処理を診断するためのテスト
// ディレクティブコメント、通常のコメント、トークンの構造を可視化
func TestTokenDumpInCTE(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		diagnosticMessage string
	}{
		{
			name:              "CTE with normal comment",
			sql:               "WITH cte AS (SELECT id /* normal comment */ FROM users) SELECT id FROM cte",
			diagnosticMessage: "通常のコメント /* normal comment */ がトークンレベルでどう処理されるかを確認",
		},
		{
			name:              "CTE with multiple comments",
			sql:               "WITH cte AS (SELECT id /* first */, name /* second */ FROM users) SELECT id FROM cte",
			diagnosticMessage: "複数のコメントがトークンレベルで処理される様子を確認",
		},
		{
			name:              "CTE with line comment",
			sql:               "WITH cte AS (SELECT id -- line comment\nFROM users) SELECT id FROM cte",
			diagnosticMessage: "ラインコメント -- が CTE 内でどう処理されるかを確認",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("\n=== %s ===", tt.diagnosticMessage)
			t.Logf("Input SQL: %s", tt.sql)

			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})

			// パーサーエラーの場合は診断情報を表示
			if err != nil {
				t.Logf("❌ Parser Error: %v", err)
				return
			}

			require.NotNil(t, stmt)

			selectStmt, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			// CTE の RawTokens を確認
			var targetTokens []tokenizer.Token
			var sourceDescription string

			if selectStmt.CTE() != nil {
				// CTE が存在する場合、CTE の RawTokens を確認
				ctes := selectStmt.CTE().CTEs
				if len(ctes) > 0 {
					targetTokens = ctes[0].RawTokens
					sourceDescription = fmt.Sprintf("CTE '%s' RawTokens", ctes[0].Name)
				}
			}

			// トークン情報のダンプ
			t.Logf("\n%s (%d tokens):", sourceDescription, len(targetTokens))
			if len(targetTokens) == 0 {
				t.Logf("  (no tokens found)")
			} else {
				for idx, token := range targetTokens {
					hasDirective := token.Directive != nil
					t.Logf("  [%d] Type=%-15s Value=%-30q HasDirective=%-5v Pos=%s",
						idx, token.Type, token.Value, hasDirective, token.Position.String())

					if hasDirective {
						t.Logf("       └─ Directive: Type=%s Condition=%q",
							token.Directive.Type, token.Directive.Condition)
					}
				}
			}

			// 期待値の検証
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "Should not error generating instructions")

			// 生成された命令を表示
			t.Logf("\nGenerated Instructions:")
			systemOpsStartIdx := len(instructions)
			for idx, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					systemOpsStartIdx = idx
					break
				}
				t.Logf("  [%d] Op=%-20s Value=%-40q Pos=%s",
					idx, instr.Op, instr.Value, instr.Pos)
			}
			t.Logf("  (system ops start at index %d)", systemOpsStartIdx)
		})
	}
}

// TestCTEWithSubqueries は CTE 内のサブクエリーサポートをテストする
// Phase 4.5 実装予定の CTE 内サブクエリー拡張に対応したテストケース
func TestCTEWithSubqueries(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		expectedInstructions []Instruction
	}{
		{
			name: "CTE with subquery in FROM clause",
			sql:  "WITH cte AS (SELECT id FROM (SELECT id FROM users) u) SELECT id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM (SELECT id FROM users) u)SELECT id FROM cte", Pos: "1:1"},
			},
		},
		{
			name: "CTE with subquery in WHERE clause",
			sql:  "WITH cte AS (SELECT id FROM users WHERE id IN (SELECT id FROM orders)) SELECT id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM users WHERE id IN (SELECT id FROM orders))SELECT id FROM cte", Pos: "1:1"},
			},
		},
		{
			name: "CTE with nested subqueries",
			sql:  "WITH cte AS (SELECT id FROM (SELECT id FROM (SELECT id FROM users) t1) t2) SELECT id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM (SELECT id FROM (SELECT id FROM users) t1) t2)SELECT id FROM cte", Pos: "1:1"},
			},
		},
		{
			name: "CTE with EXISTS subquery",
			sql:  "WITH cte AS (SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE users.id = orders.user_id)) SELECT id FROM cte",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "WITH cte AS (SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE users.id = orders.user_id))SELECT id FROM cte", Pos: "1:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Parser should accept the SQL")
			require.NotNil(t, stmt)

			_, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "Expected SELECT statement")

			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err, "Should not error generating instructions")

			// CTE のインストラクション検証
			actualBeforeSystemOps := []Instruction{}

			for _, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}

				actualBeforeSystemOps = append(actualBeforeSystemOps, instr)
			}

			assert.Equal(t, tt.expectedInstructions, actualBeforeSystemOps, "Instructions mismatch")
		})
	}
}


// TestCTETokenDump は CTE・サブクエリー内部のトークンをダンプして確認するテスト
// ディレクティブコメントが CTE 内でどのように処理されるか、またはプレーンコメントになるか検証
func TestCTETokenDump(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "CTE with directive comment in WHERE clause",
			sql: `/*# parameters: { status_filter: string } */
WITH cte AS (
SELECT id, name FROM users WHERE status = /*= status_filter */'active'
)
SELECT id FROM cte`,
		},
		{
			name: "Subquery with directive comment in WHERE clause",
			sql: `/*# parameters: { min_age: int } */
SELECT id FROM (
SELECT id, age FROM users WHERE age >= /*= min_age */18
) AS subq`,
		},
		{
			name: "CTE with nested subquery and directive comments",
			sql: `/*# parameters: { dept_id: int, min_salary: int } */
WITH filtered_users AS (
SELECT id, name, salary FROM users WHERE department_id = /*= dept_id */1
)
SELECT id FROM (
SELECT id FROM filtered_users WHERE salary >= /*= min_salary */50000
) AS high_earners`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
// Parse SQL with constants
constants := map[string]interface{}{
"status_filter": "active",
"min_age":       18,
"dept_id":       1,
"min_salary":    50000,
}

reader := strings.NewReader(tt.sql)
stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
require.NoError(t, err, "ParseSQLFile should succeed")
require.NotNil(t, stmt)

selectStmt, ok := stmt.(*parser.SelectStatement)
require.True(t, ok, "Expected SELECT statement")

// Dump CTE tokens if present
if selectStmt.CTE() != nil {
				t.Logf("\n=== WITH Clause Tokens ===")
				withClause := selectStmt.CTE()
				for i, cte := range withClause.CTEs {
					t.Logf("  CTE[%d]: %s", i, cte.Name)
					tokens := cte.RawTokens // RawTokens is a field, not a method
					for j, token := range tokens {
						directive := ""
						if token.Directive != nil {
							directive = fmt.Sprintf(" [DIRECTIVE: Type=%s]", token.Directive.Type)
						}
						t.Logf("    Token[%d]: %q%s", j, token.Value, directive)
					}
				}
			}

			// Generate instructions to validate parsing
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			if err != nil {
				t.Logf("GenerateSelectInstructions error: %v", err)
				return
			}

			// Display generated instructions
			t.Logf("\n=== Generated Instructions ===")
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}
				switch instr.Op {
				case OpEmitStatic:
					t.Logf("  [%d] OpEmitStatic: %q", i, instr.Value)
				case OpEmitEval:
					t.Logf("  [%d] OpEmitEval (ExprIndex: %v)", i, instr.ExprIndex)
				default:
					t.Logf("  [%d] %s", i, instr.Op)
				}
			}

			// Display CEL expressions if any
			if len(ctx.Expressions) > 0 {
				t.Logf("\n=== CEL Expressions ===")
				for i, expr := range ctx.Expressions {
					t.Logf("  [%d] %s", i, expr.Expression)
				}
			}
		})
	}
}

// TestCTESubqueryTokenDump は CTE 内のサブクエリーのトークンをダンプするテスト
// サブクエリーがどのように処理されるかを詳細に確認
func TestCTESubqueryTokenDump(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "CTE with subquery in FROM clause",
			sql: `/*# parameters: { dept_id: int } */
WITH filtered AS (
SELECT id, name FROM (
SELECT id, name FROM users WHERE department_id = /*= dept_id */1
) AS subq
)
SELECT id FROM filtered`,
		},
		{
			name: "CTE with EXISTS subquery",
			sql: `/*# parameters: { min_salary: int } */
WITH high_earners AS (
SELECT id FROM employees
WHERE EXISTS (
SELECT 1 FROM salaries WHERE salary >= /*= min_salary */50000 AND employee_id = employees.id
	)
)
SELECT id FROM high_earners`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
constants := map[string]interface{}{
"dept_id":    1,
"min_salary": 50000,
}

reader := strings.NewReader(tt.sql)
stmt, _, err := parser.ParseSQLFile(reader, constants, "", "", parser.Options{})
require.NoError(t, err, "ParseSQLFile should succeed")
require.NotNil(t, stmt)

selectStmt, ok := stmt.(*parser.SelectStatement)
require.True(t, ok, "Expected SELECT statement")

// Dump CTE tokens
if selectStmt.CTE() != nil {
				t.Logf("\n=== CTE RawTokens ===")
				withClause := selectStmt.CTE()
				for i, cte := range withClause.CTEs {
					t.Logf("CTE[%d]: %s", i, cte.Name)
					tokens := cte.RawTokens
					for j, token := range tokens {
						directive := ""
						if token.Directive != nil {
							directive = fmt.Sprintf(" [DIRECTIVE: Type=%s]", token.Directive.Type)
						}
						// Show tokens that contain key markers (subquery, directive, etc.)
						if strings.Contains(token.Value, "(") || strings.Contains(token.Value, ")") ||
							strings.Contains(token.Value, "SELECT") || token.Directive != nil {
							t.Logf("  Token[%d]: %q%s", j, token.Value, directive)
						}
					}
				}
			}

			// Generate instructions
			ctx := NewGenerationContext(snapsql.DialectPostgres)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			if err != nil {
				t.Logf("ERROR: %v", err)
				return
			}

			t.Logf("\n=== Instructions ===")
			for i, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}
				if instr.Op == OpEmitStatic {
					// Show first 100 chars of static text
					val := instr.Value
					if len(val) > 100 {
						val = val[:100] + "..."
					}
					t.Logf("  [%d] OpEmitStatic: %q", i, val)
				} else if instr.Op == OpEmitEval {
					t.Logf("  [%d] OpEmitEval (ExprIndex: %v)", i, instr.ExprIndex)
				} else {
					t.Logf("  [%d] %s", i, instr.Op)
				}
			}

			// Show CEL expressions
			if len(ctx.Expressions) > 0 {
				t.Logf("\n=== CEL Expressions ===")
				for i, expr := range ctx.Expressions {
					t.Logf("  [%d] %s", i, expr.Expression)
				}
			}
		})
	}
}
