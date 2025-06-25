package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestBulkVariableIntegration(t *testing.T) {
	tests := []struct {
		name string
		sql string
		schema *InterfaceSchema
		expected string
	}{
		{
			name: "maparray bulk insert String() output",
			sql: "INSERT INTO products (name, price) VALUES /*= products */('Product A', 100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"products": []map[string]any{
						{"name": "Product A", "price": 100.50},
						{"name": "Product B", "price": 200.75},
					},
				},
			},
			expected: "INSERT INTO   (NAME, PRICE) VALUES /*= products */('Product A', 100.50)",
		},
		{
			name: "anyarray bulk insert String() output",
			sql: "INSERT INTO users (name, email) VALUES /*= users */('John', 'john@example.com');",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"users": []any{
						map[string]any{"name": "John", "email": "john@example.com"},
						map[string]any{"name": "Jane", "email": "jane@example.com"},
					},
				},
			},
			expected: "INSERT INTO   (NAME, EMAIL) VALUES /*= users */('John', 'john@example.com')",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// 名前空間を作成
			ns := NewNamespace(test.schema)

			// パース
			parser := NewSqlParser(tokens, ns)
			stmt, err := parser.Parse()
			assert.NoError(t, err)

			// INSERT statementであることをverification 
			insertStmt, ok := stmt.(*InsertStatement)
			assert.True(t, ok,"Expected InsertStatement")

			// bulk variable が設定されていることをverification 
			assert.NotZero(t, insertStmt.BulkVariable,"Expected BulkVariable to be set")

			// Stringoutput をverification 
			result := insertStmt.String()
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestBulkVariableVsRegularInsert(t *testing.T) {
	tests := []struct {
		name string
		sql string
		schema *InterfaceSchema
		expectBulk bool
	}{
		{
			name: "maparray is detected as bulk variable",
			sql: "INSERT INTO products (name, price) VALUES /*= products */('Product A', 100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"products": []map[string]any{
						{"name": "Product A", "price": 100.50},
					},
				},
			},
			expectBulk: true,
		},
		{
			name: "single map is processed as normal INSERT",
			sql: "INSERT INTO products (name, price) VALUES (/*= product.name */'Product A', /*= product.price */100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"product": map[string]any{
						"name": "Product A",
						"price": 100.50,
					},
				},
			},
			expectBulk: false,
		},
		{
			name: "normal variable is processed as normal INSERT",
			sql: "INSERT INTO products (name, price) VALUES (/*= name */'Product A', /*= price */100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"name": "Product A",
					"price": 100.50,
				},
			},
			expectBulk: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// トークナイズ
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// 名前空間を作成
			ns := NewNamespace(test.schema)

			// パース
			parser := NewSqlParser(tokens, ns)
			stmt, err := parser.Parse()
			assert.NoError(t, err)

			// INSERT statementであることをverification 
			insertStmt, ok := stmt.(*InsertStatement)
			assert.True(t, ok,"Expected InsertStatement")

			if test.expectBulk {
				assert.NotZero(t, insertStmt.BulkVariable,"Expected BulkVariable to be set")
				assert.Zero(t, len(insertStmt.ValuesList),"Expected ValuesList to be empty for bulk variable")
			} else {
				assert.Zero(t, insertStmt.BulkVariable,"Expected BulkVariable to be nil")
				// normal INSERTの場合,ValuesList または SelectStmt が設定されているはず
				hasValues := len(insertStmt.ValuesList) > 0 || insertStmt.SelectStmt != nil
				assert.True(t, hasValues,"Expected either ValuesList or SelectStmt to be set")
			}
		})
	}
}
