package parserstep6

import (
	"fmt"
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
)

// parseUpToStep5 は parser.RawParse の一部を再実装し、parserstep5までを実行します
func parseUpToStep5(sql string) (cmn.StatementNode, error) {
	// トークナイズ
	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Step 1: Run parserstep1 - Basic syntax validation and dummy literal insertion
	processedTokens, err := parserstep1.Execute(tokens)
	if err != nil {
		return nil, fmt.Errorf("parserstep1 failed: %w", err)
	}

	// Step 2: Run parserstep2 - SQL structure parsing
	stmt, err := parserstep2.Execute(processedTokens)
	if err != nil {
		return nil, fmt.Errorf("parserstep2 failed: %w", err)
	}

	// Step 3: Run parserstep3 - Clause-level validation and assignment
	if err := parserstep3.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep3 failed: %w", err)
	}

	// Step 4: Run parserstep4 - Clause content validation
	if err := parserstep4.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep4 failed: %w", err)
	}

	// Step 5: Run parserstep5 - Directive structure validation
	if err := parserstep5.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep5 failed: %w", err)
	}

	return stmt, nil
}

// findVariableDirectiveTokens は指定された変数名のディレクティブトークンとその周辺のトークンを検索します
func findVariableDirectiveTokens(stmt cmn.StatementNode, varName string, isConst bool) ([]tokenizer.Token, bool) {
	prefix := "/*= "
	if isConst {
		prefix = "/*$ "
	}
	
	for _, clause := range stmt.Clauses() {
		tokens := clause.RawTokens()
		for i, token := range tokens {
			if token.Type == tokenizer.BLOCK_COMMENT && token.Value == prefix+varName+" */" {
				if isConst {
					// /*$ */形式の場合、コメントの次のトークンが値
					if i+1 < len(tokens) {
						return tokens[i:i+2], true
					}
				} else {
					// /*= */形式の場合、コメントの次にDUMMY_STARTトークンがあるか確認
					if i+3 < len(tokens) && 
					   tokens[i+1].Type == tokenizer.DUMMY_START && 
					   tokens[i+3].Type == tokenizer.DUMMY_END {
						// コメント、DUMMY_START、値、DUMMY_ENDの4つのトークンを返す
						return tokens[i:i+4], true
					}
				}
				return nil, false
			}
		}
	}
	return nil, false
}

func TestDummyLiteralReplacer(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		params         map[string]any
		expectedValues map[string]string // 変数名とその期待される置換後の値
		isConst        map[string]bool   // 変数名とそれが/*$ */形式かどうか
	}{
		{
			name: "Replace integer literal",
			sql:  "SELECT /*= value */ FROM users",
			params: map[string]any{
				"value": 42,
			},
			expectedValues: map[string]string{
				"value": "42",
			},
			isConst: map[string]bool{
				"value": false,
			},
		},
		{
			name: "Replace string literal",
			sql:  "SELECT /*= name */ FROM users",
			params: map[string]any{
				"name": "John",
			},
			expectedValues: map[string]string{
				"name": "'John'",
			},
			isConst: map[string]bool{
				"name": false,
			},
		},
		{
			name: "Replace boolean literal",
			sql:  "SELECT /*= active */ FROM users",
			params: map[string]any{
				"active": true,
			},
			expectedValues: map[string]string{
				"active": "TRUE",
			},
			isConst: map[string]bool{
				"active": false,
			},
		},
		{
			name: "Replace float literal",
			sql:  "SELECT /*= price */ FROM products",
			params: map[string]any{
				"price": 19.99,
			},
			expectedValues: map[string]string{
				"price": "19.99",
			},
			isConst: map[string]bool{
				"price": false,
			},
		},
		{
			name: "Multiple replacements",
			sql:  "SELECT /*= name */, /*= age */ FROM users WHERE active = /*= isActive */",
			params: map[string]any{
				"name":     "Alice",
				"age":      30,
				"isActive": true,
			},
			expectedValues: map[string]string{
				"name":     "'Alice'",
				"age":      "30",
				"isActive": "TRUE",
			},
			isConst: map[string]bool{
				"name":     false,
				"age":      false,
				"isActive": false,
			},
		},
		{
			name: "Replace const integer literal",
			sql:  "SELECT /*$ value */ FROM users",
			params: map[string]any{
				"value": 42,
			},
			expectedValues: map[string]string{
				"value": "42",
			},
			isConst: map[string]bool{
				"value": true,
			},
		},
		{
			name: "Replace const string literal",
			sql:  "SELECT /*$ name */ FROM users",
			params: map[string]any{
				"name": "John",
			},
			expectedValues: map[string]string{
				"name": "'John'",
			},
			isConst: map[string]bool{
				"name": true,
			},
		},
		{
			name: "Replace const boolean literal",
			sql:  "SELECT /*$ active */ FROM users",
			params: map[string]any{
				"active": true,
			},
			expectedValues: map[string]string{
				"active": "TRUE",
			},
			isConst: map[string]bool{
				"active": true,
			},
		},
		{
			name: "Mix of dummy and const literals",
			sql:  "SELECT /*= name */, /*$ age */ FROM users WHERE active = /*$ isActive */",
			params: map[string]any{
				"name":     "Alice",
				"age":      30,
				"isActive": true,
			},
			expectedValues: map[string]string{
				"name":     "'Alice'",
				"age":      "30",
				"isActive": "TRUE",
			},
			isConst: map[string]bool{
				"name":     false,
				"age":      true,
				"isActive": true,
			},
		},
		// SELECT文のWHERE句のテストケース
		{
			name: "SELECT with WHERE clause and literals",
			sql:  "SELECT * FROM users WHERE id = /*= id */ AND name LIKE /*= pattern */",
			params: map[string]any{
				"id":      1,
				"pattern": "J%",
			},
			expectedValues: map[string]string{
				"id":      "1",
				"pattern": "'J%'",
			},
			isConst: map[string]bool{
				"id":      false,
				"pattern": false,
			},
		},
		{
			name: "SELECT with complex WHERE clause and literals",
			sql:  "SELECT * FROM users WHERE id = /*= id */ OR name = /*= name */",
			params: map[string]any{
				"id":   1,
				"name": "John",
			},
			expectedValues: map[string]string{
				"id":   "1",
				"name": "'John'",
			},
			isConst: map[string]bool{
				"id":   false,
				"name": false,
			},
		},
		{
			name: "SELECT with WHERE IN clause and literals",
			sql:  "SELECT * FROM users WHERE id IN (/*= id1 */, /*= id2 */, /*= id3 */)",
			params: map[string]any{
				"id1": 1,
				"id2": 2,
				"id3": 3,
			},
			expectedValues: map[string]string{
				"id1": "1",
				"id2": "2",
				"id3": "3",
			},
			isConst: map[string]bool{
				"id1": false,
				"id2": false,
				"id3": false,
			},
		},
		{
			name: "SELECT with WHERE clause and mix of dummy and const literals",
			sql:  "SELECT * FROM users WHERE id = /*= id */ AND name LIKE /*$ pattern */",
			params: map[string]any{
				"id":      1,
				"pattern": "J%",
			},
			expectedValues: map[string]string{
				"id":      "1",
				"pattern": "'J%'",
			},
			isConst: map[string]bool{
				"id":      false,
				"pattern": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parserstep5までパース
			stmt, err := parseUpToStep5(tt.sql)
			assert.NoError(t, err)
			
			// パラメータ名前空間を作成
			paramNs, err := cmn.NewNamespaceFromConstants(tt.params)
			assert.NoError(t, err)
			
			// ダミーリテラル置換を実行
			perr := &cmn.ParseError{}
			replaceDummyLiterals(stmt, paramNs, perr)
			assert.Empty(t, perr.Errors)
			
			// 各変数の置換結果を確認
			for varName, expectedValue := range tt.expectedValues {
				isConst := tt.isConst[varName]
				tokens, found := findVariableDirectiveTokens(stmt, varName, isConst)
				assert.True(t, found, "Variable %s not found in tokens", varName)
				if found {
					if isConst {
						// /*$ */形式の場合、コメントの次のトークンが値
						assert.Equal(t, expectedValue, tokens[1].Value, "Unexpected value for const variable %s", varName)
					} else {
						// /*= */形式の場合、DUMMY_STARTの次のトークンが値
						assert.Equal(t, expectedValue, tokens[2].Value, "Unexpected value for variable %s", varName)
					}
				}
			}
		})
	}
}
