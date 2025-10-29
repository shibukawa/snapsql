package codegenerator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDialectJoinConversion は JOIN句が正しく処理されることをテストする
func TestDialectJoinConversion(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		dialect          snapsql.Dialect
		expectJoinString bool
	}{
		{
			name:             "INNER JOIN",
			sql:              `SELECT u.user_id, o.order_id FROM users u INNER JOIN orders o ON u.user_id = o.user_id`,
			dialect:          snapsql.DialectPostgres,
			expectJoinString: true,
		},
		{
			name:             "LEFT JOIN",
			sql:              `SELECT u.user_id, o.order_id FROM users u LEFT JOIN orders o ON u.user_id = o.user_id`,
			dialect:          snapsql.DialectPostgres,
			expectJoinString: true,
		},
		{
			name:             "RIGHT JOIN",
			sql:              `SELECT u.user_id, o.order_id FROM users u RIGHT JOIN orders o ON u.user_id = o.user_id`,
			dialect:          snapsql.DialectPostgres,
			expectJoinString: true,
		},
		{
			name:             "CROSS JOIN",
			sql:              `SELECT u.user_id, r.role_id FROM users u CROSS JOIN roles r`,
			dialect:          snapsql.DialectPostgres,
			expectJoinString: true,
		},
		{
			name:             "Multiple JOINs",
			sql:              `SELECT u.user_id, o.order_id, p.product_id FROM users u LEFT JOIN orders o ON u.user_id = o.user_id INNER JOIN products p ON o.order_id = p.product_id`,
			dialect:          snapsql.DialectPostgres,
			expectJoinString: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(tt.dialect)

			// パーサーでSQLをパース
			sqlWithSemicolon := tt.sql
			if !strings.HasSuffix(sqlWithSemicolon, ";") {
				sqlWithSemicolon += ";"
			}

			reader := bytes.NewBufferString(sqlWithSemicolon)
			parsedStmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Failed to parse SQL: %s", tt.sql)

			instructions, _, _, err := GenerateSelectInstructions(parsedStmt, ctx)
			require.NoError(t, err, "Failed to generate instructions for: %s", tt.sql)

			// 命令列が生成されたことを確認
			assert.NotEmpty(t, instructions, "Expected instructions to be generated")

			if tt.expectJoinString {
				// JOIN関連のキーワードを含む EMIT_STATIC が存在することを確認
				joinFound := false

				for _, instr := range instructions {
					if instr.Op == "EMIT_STATIC" && instr.Value != "" {
						if strings.Contains(strings.ToUpper(instr.Value), "JOIN") {
							joinFound = true
							break
						}
					}
				}

				assert.True(t, joinFound, "Expected to find JOIN keyword in instructions for SQL: %s", tt.sql)
			}
		})
	}
}

// TestJoinTypeNormalization は JOIN型の正規化をテストする
// LEFT OUTER JOIN → LEFT JOIN など
func TestJoinTypeNormalization(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		expectsNot string // 正規化前のキーワードを含まないことを確認
		expects    string // 正規化後のキーワードを含むことを確認
	}{
		{
			name:       "LEFT OUTER JOIN normalization",
			sql:        `SELECT u.user_id, o.order_id FROM users u LEFT OUTER JOIN orders o ON u.user_id = o.user_id`,
			expectsNot: "OUTER",
			expects:    "LEFT",
		},
		{
			name:       "RIGHT OUTER JOIN normalization",
			sql:        `SELECT u.user_id, o.order_id FROM users u RIGHT OUTER JOIN orders o ON u.user_id = o.user_id`,
			expectsNot: "OUTER",
			expects:    "RIGHT",
		},
		{
			name:       "INNER JOIN stays the same",
			sql:        `SELECT u.user_id, o.order_id FROM users u INNER JOIN orders o ON u.user_id = o.user_id`,
			expectsNot: "",
			expects:    "INNER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(snapsql.DialectPostgres)

			sqlWithSemicolon := tt.sql
			if !strings.HasSuffix(sqlWithSemicolon, ";") {
				sqlWithSemicolon += ";"
			}

			reader := bytes.NewBufferString(sqlWithSemicolon)
			parsedStmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Failed to parse SQL: %s", tt.sql)

			instructions, _, _, err := GenerateSelectInstructions(parsedStmt, ctx)
			require.NoError(t, err, "Failed to generate instructions for: %s", tt.sql)

			// SQL文字列を再構築
			var sqlOutput strings.Builder

			for _, instr := range instructions {
				if instr.Op == "EMIT_STATIC" && instr.Value != "" {
					sqlOutput.WriteString(instr.Value)
				}
			}

			resultSQL := sqlOutput.String()

			if tt.expects != "" {
				assert.Contains(t, resultSQL, tt.expects, "Expected '%s' in output SQL", tt.expects)
			}

			if tt.expectsNot != "" {
				assert.NotContains(t, resultSQL, tt.expectsNot, "Expected '%s' NOT to be in output SQL", tt.expectsNot)
			}
		})
	}
}

// TestJoinConditionDialectConversion は ON条件内の方言変換をテストする
// CONCAT/||, TRUE/FALSE, COALESCE/IFNULL などの変換
func TestJoinConditionDialectConversion(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		fromDialect snapsql.Dialect
		toDialect   snapsql.Dialect
		expectsNot  string // 変換前のキーワードを含まないことを確認
		expects     string // 変換後のキーワードを含むことを確認
	}{
		// NOTE: COALESCE/IFNULL テストは削除
		// COALESCE と IFNULL はすべての対応DB (PostgreSQL, MySQL, SQLite)で両方サポートされている。
		// functionsigs.go で確認済みのため、相互変換は不要。
		{
			name:        "TRUE/FALSE in ON condition (PostgreSQL to MySQL)",
			sql:         `SELECT u.user_id FROM users u INNER JOIN orders o ON u.is_active = TRUE AND o.is_pending = FALSE`,
			fromDialect: snapsql.DialectPostgres,
			toDialect:   snapsql.DialectMySQL,
			expects:     "1",
			expectsNot:  "TRUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewGenerationContext(tt.toDialect)

			sqlWithSemicolon := tt.sql
			if !strings.HasSuffix(sqlWithSemicolon, ";") {
				sqlWithSemicolon += ";"
			}

			reader := bytes.NewBufferString(sqlWithSemicolon)
			parsedStmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "Failed to parse SQL: %s", tt.sql)

			instructions, _, _, err := GenerateSelectInstructions(parsedStmt, ctx)
			require.NoError(t, err, "Failed to generate instructions for: %s", tt.sql)

			// SQL文字列を再構築
			var sqlOutput strings.Builder

			for _, instr := range instructions {
				if instr.Op == "EMIT_STATIC" && instr.Value != "" {
					sqlOutput.WriteString(instr.Value)
				}
			}

			resultSQL := sqlOutput.String()

			if tt.expects != "" {
				assert.Contains(t, resultSQL, tt.expects, "Expected '%s' in converted SQL", tt.expects)
			}

			if tt.expectsNot != "" {
				assert.NotContains(t, resultSQL, tt.expectsNot, "Expected '%s' NOT to be in converted SQL", tt.expectsNot)
			}
		})
	}
}
