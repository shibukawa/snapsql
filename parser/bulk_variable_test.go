package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestBulkVariableSubstitution(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		schema      *InterfaceSchema
		expectBulk  bool
		expectError bool
	}{
		{
			name: "maparray bulk variable",
			sql:  "INSERT INTO products (name, price) VALUES /*= products */('Product A', 100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"products": []map[string]any{
						{"name": "Product A", "price": 100.50},
						{"name": "Product B", "price": 200.75},
					},
				},
			},
			expectBulk:  true,
			expectError: false,
		},
		{
			name: "single map bulk variable (normal INSERT processing)",
			sql:  "INSERT INTO products (name, price) VALUES (/*= product.name */'Product A', /*= product.price */100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"product": map[string]any{
						"name":  "Product A",
						"price": 100.50,
					},
				},
			},
			expectBulk:  false,
			expectError: false,
		},
		{
			name: "normal variable (not bulk)",
			sql:  "INSERT INTO products (name, price) VALUES (/*= name */'Product A', /*= price */100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"name":  "Product A",
					"price": 100.50,
				},
			},
			expectBulk:  false,
			expectError: false,
		},
		{
			name: "anyarray bulk variable",
			sql:  "INSERT INTO products (name, price) VALUES /*= products */('Product A', 100.50);",
			schema: &InterfaceSchema{
				Parameters: map[string]any{
					"products": []any{
						map[string]any{"name": "Product A", "price": 100.50},
						map[string]any{"name": "Product B", "price": 200.75},
					},
				},
			},
			expectBulk:  true,
			expectError: false,
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

				if test.expectBulk {
					// bulk variable が設定されていることをverification
					assert.NotZero(t, insertStmt.BulkVariable, "Expected BulkVariable to be set")

					bulkVar, ok := insertStmt.BulkVariable.(*BulkVariableSubstitution)
					assert.True(t, ok, "Expected BulkVariableSubstitution")
					assert.True(t, len(bulkVar.Expression) > 0, "Expected non-empty expression")
				} else {
					// bulk variable が設定されていないことをverification
					assert.Zero(t, insertStmt.BulkVariable, "Expected BulkVariable to be nil")
				}
			}
		})
	}
}

func TestBulkVariableTypeDetection(t *testing.T) {
	tests := []struct {
		name       string
		paramType  any
		expectBulk bool
	}{
		{
			name: "maparray",
			paramType: []map[string]any{
				{"name": "Product A", "price": 100.50},
			},
			expectBulk: true,
		},
		{
			name: "anyarray (map element)",
			paramType: []any{
				map[string]any{"name": "Product A", "price": 100.50},
			},
			expectBulk: true,
		},
		{
			name:       "empty array",
			paramType:  []any{},
			expectBulk: true,
		},
		{
			name: "string array ",
			paramType: []string{
				"Product A", "Product B",
			},
			expectBulk: false,
		},
		{
			name: "single map",
			paramType: map[string]any{
				"name":  "Product A",
				"price": 100.50,
			},
			expectBulk: false,
		},
		{
			name:       "string ",
			paramType:  "Product A",
			expectBulk: false,
		},
		{
			name:       "nil",
			paramType:  nil,
			expectBulk: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isBulkVariableType(test.paramType)
			assert.Equal(t, test.expectBulk, result)
		})
	}
}

func TestBulkVariableWithDummyValues(t *testing.T) {
	sql := "INSERT INTO products (name, price) VALUES /*= products */('Product A', 100.50);"
	schema := &InterfaceSchema{
		Parameters: map[string]any{
			"products": []map[string]any{
				{"name": "Product A", "price": 100.50},
				{"name": "Product B", "price": 200.75},
			},
		},
	}

	// トークナイズ
	tok := tokenizer.NewSqlTokenizer(sql, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	// 名前空間を作成
	ns := NewNamespace(schema)

	// パース
	parser := NewSqlParser(tokens, ns, nil)
	stmt, err := parser.Parse()
	assert.NoError(t, err)

	// INSERT statementであることをverification
	insertStmt, ok := stmt.(*InsertStatement)
	assert.True(t, ok, "Expected InsertStatement")

	// bulk variable が設定されていることをverification
	assert.NotZero(t, insertStmt.BulkVariable, "Expected BulkVariable to be set")

	bulkVar, ok := insertStmt.BulkVariable.(*BulkVariableSubstitution)
	assert.True(t, ok, "Expected BulkVariableSubstitution")
	assert.Equal(t, "products", bulkVar.Expression)
	assert.Equal(t, "('Product A', 100.50)", bulkVar.DummyValue)
	assert.Equal(t, []string{"NAME", "PRICE"}, bulkVar.Columns)
}
