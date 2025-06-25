package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestBulkInsertStatement(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		expectError    bool
		expectedValues int // 期待されるVALUES clauseの数
	}{
		{
			name:           "single VALUES clause",
			sql:            "INSERT INTO users (name, email) VALUES ('John', 'john@example.com');",
			expectError:    false,
			expectedValues: 1,
		},
		{
			name:           "bulk insert (2 rows)",
			sql:            "INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com');",
			expectError:    false,
			expectedValues: 2,
		},
		{
			name:           "bulk insert (3 rows)",
			sql:            "INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com'), ('Bob', 'bob@example.com');",
			expectError:    false,
			expectedValues: 3,
		},
		{
			name:           "bulk insert without column specification",
			sql:            "INSERT INTO users VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com');",
			expectError:    false,
			expectedValues: 2,
		},
		{
			name:           "bulk insert with numeric values",
			sql:            "INSERT INTO products (name, price, stock) VALUES ('Product A', 100.50, 10), ('Product B', 200.75, 5);",
			expectError:    false,
			expectedValues: 2,
		},
		{
			name:        "incomplete bulk insert (comma without values)",
			sql:         "INSERT INTO users VALUES ('John', 'john@example.com'),;",
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
			parser := NewSqlParser(tokens, nil, nil)
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

				// VALUES clauseの数をverification
				assert.Equal(t, test.expectedValues, len(insertStmt.ValuesList), "Unexpected number of VALUES clauses")

				// 各VALUES clauseが適切な数のvalue を持つことをverification
				for i, values := range insertStmt.ValuesList {
					assert.True(t, len(values) > 0, "VALUES clause %d should have at least one value", i)
				}
			}
		})
	}
}

func TestBulkInsertWithSnapSQL(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "bulk insert with SnapSQL variables",
			sql:         "INSERT INTO users (name, email) VALUES (/*= user1.name */'John', /*= user1.email */'john@example.com'), (/*= user2.name */'Jane', /*= user2.email */'jane@example.com');",
			expectError: false,
		},
		{
			name:        "conditional bulk insert",
			sql:         "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')/*# if include_jane */, ('Jane', 'jane@example.com')/*# end */;",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 簡単なschema を作成
			schema := &InterfaceSchema{
				Parameters: map[string]any{
					"user1": map[string]any{
						"name":  "str",
						"email": "str",
					},
					"user2": map[string]any{
						"name":  "str",
						"email": "str",
					},
					"users": []any{
						map[string]any{
							"name":  "str",
							"email": "str",
						},
					},
					"include_jane": "bool",
				},
			}

			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// 名前空間を作成
			ns := NewNamespace(schema)

			// パース
			parser := NewSqlParser(tokens, ns, nil)
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

func TestBulkInsertStringGeneration(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "bulk insert String() output",
			sql:      "INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com');",
			expected: "INSERT INTO   (NAME, EMAIL) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com')",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// パース
			parser := NewSqlParser(tokens, nil, nil)
			stmt, err := parser.Parse()
			assert.NoError(t, err)

			// INSERT statementであることをverification
			insertStmt, ok := stmt.(*InsertStatement)
			assert.True(t, ok, "Expected InsertStatement")

			// String()output をverification
			result := insertStmt.String()
			t.Logf("Actual output: %s", result)
			t.Logf("Expected output: %s", test.expected)
			assert.Equal(t, test.expected, result)
		})
	}
}
