package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestInsertStatement(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "basic INSERT statement",
			sql:         "INSERT INTO users (name, email) VALUES ('John', 'john@example.com');",
			expectError: false,
		},
		{
			name:        "INSERT statement without column specification",
			sql:         "INSERT INTO users VALUES ('John', 'john@example.com');",
			expectError: false,
		},
		{
			name:        "INSERT SELECT statement",
			sql:         "INSERT INTO users (name) SELECT name FROM temp_users;",
			expectError: false,
		},
		{
			name:        "incomplete INSERT statement",
			sql:         "INSERT INTO;",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// パース
			parser := NewSqlParser(tokens, nil)
			stmt, err := parser.Parse()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, stmt)

				// INSERT statementであることをverification
				insertStmt, ok := stmt.(*InsertStatement)
				assert.True(t, ok, "Expected InsertStatement")
				assert.NotZero(t, insertStmt.Table)
			}
		})
	}
}

func TestUpdateStatement(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "basic UPDATE statement",
			sql:         "UPDATE users SET name = 'Jane' WHERE id = 1;",
			expectError: false,
		},
		{
			name:        "multi-column UPDATE statement",
			sql:         "UPDATE users SET name = 'Jane', email = 'jane@example.com' WHERE id = 1;",
			expectError: false,
		},
		{
			name:        "UPDATE statement without WHERE clause",
			sql:         "UPDATE users SET name = 'Jane';",
			expectError: false,
		},
		{
			name:        "incomplete UPDATE statement",
			sql:         "UPDATE users SET;",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// パース
			parser := NewSqlParser(tokens, nil)
			stmt, err := parser.Parse()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, stmt)

				// UPDATE statementであることをverification
				updateStmt, ok := stmt.(*UpdateStatement)
				assert.True(t, ok, "Expected UpdateStatement")
				assert.NotZero(t, updateStmt.Table)
				assert.True(t, len(updateStmt.SetClauses) > 0, "Expected at least one SET clause")
			}
		})
	}
}

func TestDeleteStatement(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "basic DELETE statement",
			sql:         "DELETE FROM users WHERE id = 1;",
			expectError: false,
		},
		{
			name:        "DELETE statement without WHERE clause",
			sql:         "DELETE FROM users;",
			expectError: false,
		},
		{
			name:        "incomplete DELETE statement",
			sql:         "DELETE FROM;",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// パース
			parser := NewSqlParser(tokens, nil)
			stmt, err := parser.Parse()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, stmt)

				// DELETE statementであることをverification
				deleteStmt, ok := stmt.(*DeleteStatement)
				assert.True(t, ok, "Expected DeleteStatement")
				assert.NotZero(t, deleteStmt.Table)
			}
		})
	}
}

func TestDMLWithSnapSQL(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "INSERT statementでのSnapSQLvariable ",
			sql:         "INSERT INTO users (name) VALUES (/*= user_name */'test');",
			expectError: false,
		},
		{
			name:        "UPDATE statementでのSnapSQLvariable ",
			sql:         "UPDATE users SET name = /*= new_name */'test' WHERE id = 1;",
			expectError: false,
		},
		{
			name:        "DELETE statementでのSnapSQLvariable ",
			sql:         "DELETE FROM users WHERE active = /*= filters.active */true;",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 簡単なschema を作成
			schema := &InterfaceSchema{
				Parameters: map[string]any{
					"env":       "str",
					"user_name": "str",
					"new_name":  "str",
					"active":    "bool",
					"filters": map[string]any{
						"active": "bool",
					},
				},
			}

			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// 名前空間を作成
			ns := NewNamespace(schema)

			// パース
			parser := NewSqlParser(tokens, ns)
			stmt, err := parser.Parse()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, stmt)
			}
		})
	}
}
