package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubqueryInFromClause は FROM 句内のサブクエリーをテストする
func TestSubqueryInFromClause(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		expectedInstructions []Instruction
	}{
		{
			name: "simple subquery in FROM",
			sql:  "SELECT u.id FROM (SELECT id FROM users) AS u",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT u.id FROM (SELECT id FROM users) AS u", Pos: "1:1"},
			},
		},
		{
			name: "nested subqueries in FROM",
			sql:  "SELECT id FROM (SELECT id FROM (SELECT id FROM users) AS t1) AS t2",
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM (SELECT id FROM (SELECT id FROM users) AS t1) AS t2", Pos: "1:1"},
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
			ctx := NewGenerationContext(snapsql.DialectPostgres)

			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証（システム命令の前までのみ）
			actualBeforeSystemOps := []Instruction{}

			for _, instr := range instructions {
				if instr.Op == OpIfSystemLimit || instr.Op == OpIfSystemOffset || instr.Op == OpEmitSystemFor {
					break
				}

				actualBeforeSystemOps = append(actualBeforeSystemOps, instr)
			}

			assert.Equal(t, tt.expectedInstructions, actualBeforeSystemOps, "Instructions mismatch") // Expressions が存在することを確認
			assert.NotNil(t, expressions)
		})
	}
}

// TestWhereSubqueryInClause は WHERE 句内の IN サブクエリーをテストする
func TestWhereSubqueryInClause(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
	}{
		{
			name:    "IN with subquery",
			sql:     "SELECT id FROM users WHERE id IN (SELECT user_id FROM orders)",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE id IN (SELECT user_id FROM orders)", Pos: "1:1"},
			},
		},
		{
			name:    "NOT IN with subquery",
			sql:     "SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM orders)",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM orders)", Pos: "1:1"},
			},
		},
		{
			name:    "EXISTS with subquery",
			sql:     "SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)", Pos: "1:1"},
			},
		},
		{
			name:    "scalar subquery in WHERE",
			sql:     "SELECT id FROM users WHERE status = (SELECT default_status FROM configs LIMIT 1)",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id FROM users WHERE status = (SELECT default_status FROM configs LIMIT 1)", Pos: "1:1"},
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
			ctx := NewGenerationContext(tt.dialect)

			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証（システム命令の前までのみ）
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

// TestSelectScalarSubquery は SELECT 句内のスカラーサブクエリーをテストする
func TestSelectScalarSubquery(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectedInstructions []Instruction
	}{
		{
			name:    "scalar subquery in SELECT clause",
			sql:     "SELECT id, (SELECT count(*) FROM orders WHERE user_id = users.id) as order_count FROM users",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, (SELECT count(*) FROM orders WHERE user_id = users.id) as order_count FROM users", Pos: "1:1"},
			},
		},
		{
			name:    "correlated subquery in SELECT",
			sql:     "SELECT u.id, (SELECT max(order_date) FROM orders WHERE user_id = u.id) as latest FROM users u",
			dialect: snapsql.DialectPostgres,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT u.id, (SELECT max(order_date) FROM orders WHERE user_id = u.id) as latest FROM users u", Pos: "1:1"},
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
			ctx := NewGenerationContext(tt.dialect)

			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)

			// 命令列の検証（システム命令の前までのみ）
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

// TestDirectiveInSubquery はサブクエリー内でディレクティブ（パラメータ参照）が使えることをテストする
func TestDirectiveInSubquery(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedExpressions int
		shouldHaveEmitEval  bool
	}{
		{
			name:                "parameter in subquery WHERE",
			sql:                 `/*# parameters: { status_val: int } */SELECT id FROM users WHERE id IN (SELECT user_id FROM orders WHERE status = /*= status_val */1)`,
			expectedExpressions: 1,
			shouldHaveEmitEval:  true,
		},
		{
			name:                "parameter in scalar subquery",
			sql:                 `/*# parameters: { status_val: int } */SELECT id, (SELECT count(*) FROM orders WHERE user_id = users.id AND status = /*= status_val */1) as completed_count FROM users`,
			expectedExpressions: 1,
			shouldHaveEmitEval:  true,
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
			ctx := NewGenerationContext(snapsql.DialectPostgres)

			instructions, expressions, _, err := GenerateSelectInstructions(stmt, ctx)
			require.NoError(t, err)

			// 基本的なチェック
			assert.NotNil(t, instructions)
			assert.GreaterOrEqual(t, len(expressions), tt.expectedExpressions, "Should have at least %d expressions", tt.expectedExpressions)

			// OpEmitEval が含まれることを確認
			if tt.shouldHaveEmitEval {
				found := false

				for _, instr := range instructions {
					if instr.Op == OpEmitEval {
						found = true
						break
					}
				}

				assert.True(t, found, "Should contain OpEmitEval for directive")
			}
		})
	}
}
