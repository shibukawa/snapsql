package parserstep6

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
)

// parseSQL は完全なパースパイプラインを実行してトークン列を取得する
func parseSQL(sql string, constants map[string]any) (cmn.StatementNode, *cmn.FunctionDefinition, error) {
	// Tokenize the SQL content
	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, nil, err
	}

	// Extract function definition from SQL comments
	functionDef, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, "", "")
	if err != nil {
		return nil, nil, err
	}

	// Step 1: Run parserstep1 - Basic syntax validation and dummy literal insertion
	processedTokens, err := parserstep1.Execute(tokens)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Run parserstep2 - SQL structure parsing
	stmt, err := parserstep2.Execute(processedTokens)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: Run parserstep3 - Clause-level validation and assignment
	if err := parserstep3.Execute(stmt); err != nil {
		return nil, nil, err
	}

	// Step 4: Run parserstep4 - Clause content validation
	if err := parserstep4.Execute(stmt); err != nil {
		return nil, nil, err
	}

	// Step 5: Run parserstep5 - Directive structure validation
	if err := parserstep5.Execute(stmt); err != nil {
		return nil, nil, err
	}

	// Step 6: Run parserstep6 - Variable and directive validation
	// Create namespace from function definition for parameters
	paramNamespace, err := cmn.NewNamespaceFromDefinition(functionDef)
	if err != nil {
		return nil, nil, err
	}

	// Create a separate namespace for constants if provided
	var constNamespace *cmn.Namespace
	if len(constants) > 0 {
		constNamespace, err = cmn.NewNamespaceFromConstants(constants)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Create an empty constants namespace if none provided
		constNamespace, err = cmn.NewNamespaceFromConstants(map[string]any{})
		if err != nil {
			return nil, nil, err
		}
	}

	if err := Execute(stmt, paramNamespace, constNamespace); err != nil {
		return nil, nil, err
	}

	return stmt, functionDef, nil
}

// TokenExpectation はトークンの期待値を表す
type TokenExpectation struct {
	Type  tokenizer.TokenType
	Value string
}

// ClauseExpectation は句ごとの期待値を表す
type ClauseExpectation struct {
	ClauseType string // "SELECT", "FROM", "WHERE", etc.
	Tokens     []TokenExpectation
}

func TestTokenOutputByClause(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		constants   map[string]any
		description string
		// 期待値は実際の出力を確認してから設定するため、検証ロジックで対応
	}{
		{
			name: "Variable directive with dummy literal",
			sql: `/*#
function_name: test_func
parameters:
  user_id: int
*/
SELECT /*= user_id */123 FROM users WHERE id = 1`,
			constants: map[string]any{
				"user_id": 456,
			},
			description: "Variable directive creates DUMMY_START/DUMMY_END in SELECT clause and keeps original dummy literal",
		},
		{
			name: "Const directive with dummy literal",
			sql: `/*#
function_name: test_func
parameters:
  user_id: int
*/
SELECT id FROM users_/*$ table_suffix */test WHERE active = true`,
			constants: map[string]any{
				"table_suffix": "prod",
			},
			description: "Const directive inserts value in FROM clause but doesn't replace dummy literal",
		},
		{
			name: "Mixed directives across clauses",
			sql: `/*#
function_name: test_func
parameters:
  user_id: int
  status: string
*/
SELECT /*= user_id */123, name FROM users WHERE status = /*= status */'active'`,
			constants: map[string]any{
				"user_id": 456,
				"status":  "inactive",
			},
			description: "Mixed variable directives in SELECT and WHERE clauses show consistent duplication pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, funcDef, err := parseSQL(tt.sql, tt.constants)
			assert.NoError(t, err)
			assert.True(t, funcDef != nil, "Function definition should not be nil")

			clauses := stmt.Clauses()
			t.Logf("Description: %s", tt.description)
			t.Logf("Found %d clauses", len(clauses))

			// 各句の詳細分析
			for clauseIdx, clause := range clauses {
				tokens := clause.RawTokens()
				
				// 句の種類を推定
				clauseType := "UNKNOWN"
				if len(tokens) > 0 {
					switch tokens[0].Type {
					case tokenizer.SELECT:
						clauseType = "SELECT"
					case tokenizer.FROM:
						clauseType = "FROM"
					case tokenizer.WHERE:
						clauseType = "WHERE"
					case tokenizer.ORDER:
						clauseType = "ORDER BY"
					case tokenizer.GROUP:
						clauseType = "GROUP BY"
					case tokenizer.HAVING:
						clauseType = "HAVING"
					case tokenizer.LIMIT:
						clauseType = "LIMIT"
					}
				}

				t.Logf("\nClause %d (%s):", clauseIdx, clauseType)
				t.Log("Token sequence:")
				for i, token := range tokens {
					directive := ""
					if token.Directive != nil {
						directive = " (directive: " + token.Directive.Type + ")"
					}
					t.Logf("  %d: Type=%-15s Value='%s'%s", i, token.Type, token.Value, directive)
				}

				// 重複問題の検証
				duplicationsFound := analyzeDuplicationInClause(t, clauseType, tokens)
				
				// ディレクティブの動作検証
				analyzeDirectiveBehavior(t, clauseType, tokens)
				
				if duplicationsFound {
					t.Logf("✓ Duplication bug confirmed in %s clause", clauseType)
				}
			}
		})
	}
}

// analyzeDuplicationInClause は句内の重複問題を分析する
func analyzeDuplicationInClause(t *testing.T, clauseType string, tokens []tokenizer.Token) bool {
	valueCount := make(map[string]int)
	duplicationsFound := false
	
	for _, token := range tokens {
		if token.Type == tokenizer.NUMBER || token.Type == tokenizer.STRING || token.Type == tokenizer.IDENTIFIER {
			valueCount[token.Value]++
		}
	}
	
	for value, count := range valueCount {
		if count > 1 {
			t.Logf("  DUPLICATE in %s: '%s' appears %d times", clauseType, value, count)
			duplicationsFound = true
		}
	}
	
	return duplicationsFound
}

// analyzeDirectiveBehavior はディレクティブの動作を分析する
func analyzeDirectiveBehavior(t *testing.T, clauseType string, tokens []tokenizer.Token) {
	directiveCount := 0
	dummyStartCount := 0
	dummyEndCount := 0
	dummyLiteralCount := 0
	
	for _, token := range tokens {
		if token.Directive != nil {
			directiveCount++
			t.Logf("  DIRECTIVE in %s: %s", clauseType, token.Directive.Type)
		}
		if token.Type == tokenizer.DUMMY_START {
			dummyStartCount++
		}
		if token.Type == tokenizer.DUMMY_END {
			dummyEndCount++
		}
		if token.Type == tokenizer.DUMMY_LITERAL {
			dummyLiteralCount++
		}
	}
	
	if directiveCount > 0 {
		t.Logf("  ANALYSIS in %s: %d directive(s), %d DUMMY_START, %d DUMMY_END, %d DUMMY_LITERAL", 
			clauseType, directiveCount, dummyStartCount, dummyEndCount, dummyLiteralCount)
			
		// Variable directive の問題パターン
		if dummyStartCount > 0 && dummyEndCount > 0 {
			t.Logf("  ISSUE in %s: Variable directive creates DUMMY_START/DUMMY_END instead of replacing", clauseType)
		}
		
		// Const directive の問題パターン
		if directiveCount > 0 && dummyStartCount == 0 && dummyEndCount == 0 {
			// Const directive の場合、挿入されるが置換されない可能性
			valueCount := make(map[string]int)
			for _, token := range tokens {
				if token.Type == tokenizer.STRING || token.Type == tokenizer.IDENTIFIER {
					valueCount[token.Value]++
				}
			}
			
			duplicateValues := 0
			for _, count := range valueCount {
				if count > 1 {
					duplicateValues++
				}
			}
			
			if duplicateValues > 0 {
				t.Logf("  ISSUE in %s: Const directive inserts but doesn't replace dummy literals", clauseType)
			}
		}
	}
}

// TestClauseDuplicationAnalysis は句ごとの重複問題の詳細分析を行う
func TestClauseDuplicationAnalysis(t *testing.T) {
	sql := `/*#
function_name: test_func
parameters:
  user_id: int
  status: string
*/
SELECT /*= user_id */123, name FROM users_/*$ table_suffix */test WHERE status = /*= status */'active'`
	
	constants := map[string]any{
		"user_id":      456,
		"status":       "inactive", 
		"table_suffix": "prod",
	}

	stmt, _, err := parseSQL(sql, constants)
	assert.NoError(t, err)

	clauses := stmt.Clauses()
	
	t.Log("Clause-by-clause duplication analysis:")
	
	for i, clause := range clauses {
		tokens := clause.RawTokens()
		
		// 句の種類を推定
		clauseType := "UNKNOWN"
		if len(tokens) > 0 {
			switch tokens[0].Type {
			case tokenizer.SELECT:
				clauseType = "SELECT"
			case tokenizer.FROM:
				clauseType = "FROM"
			case tokenizer.WHERE:
				clauseType = "WHERE"
			}
		}
		
		t.Logf("\nClause %d (%s):", i, clauseType)
		
		// 重複分析
		valueCount := make(map[string]int)
		directiveCount := 0
		dummyTokenCount := 0
		
		for _, token := range tokens {
			if token.Type == tokenizer.NUMBER || token.Type == tokenizer.STRING || token.Type == tokenizer.IDENTIFIER {
				valueCount[token.Value]++
			}
			if token.Directive != nil {
				directiveCount++
			}
			if token.Type == tokenizer.DUMMY_START || token.Type == tokenizer.DUMMY_END || token.Type == tokenizer.DUMMY_LITERAL {
				dummyTokenCount++
			}
		}
		
		t.Logf("  Directives: %d", directiveCount)
		t.Logf("  Dummy tokens: %d", dummyTokenCount)
		
		duplicatesFound := false
		for value, count := range valueCount {
			if count > 1 {
				t.Logf("  DUPLICATE: '%s' appears %d times", value, count)
				duplicatesFound = true
			}
		}
		
		if !duplicatesFound && directiveCount > 0 && dummyTokenCount > 0 {
			t.Logf("  POTENTIAL ISSUE: %d directive(s) with %d dummy token(s) - check for insertion vs replacement", directiveCount, dummyTokenCount)
		}
		
		if !duplicatesFound && directiveCount == 0 {
			t.Logf("  OK: No directives, no duplication issues")
		}
	}
	
	t.Log("\nSummary of identified issues:")
	t.Log("1. Variable directives create DUMMY_START/DUMMY_END tokens instead of replacing dummy literals")
	t.Log("2. Const directives insert values but don't replace existing dummy literals")
	t.Log("3. Both patterns result in duplication of values in the token stream")
}
