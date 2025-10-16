package parserstep6

import (
	"fmt"
	"strconv"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// isDummyLiteral checks if a token is a dummy literal that should be replaced
func isDummyLiteral(token tokenizer.Token) bool {
	// DUMMY_LITERALトークン（parserstep1で挿入されたもの）
	if token.Type == tokenizer.DUMMY_LITERAL {
		return true
	}

	// 実際のダミーリテラル（開発者が書いたもの）
	// ディレクティブ直後にある通常のリテラルはダミーとして扱う
	if token.Type == tokenizer.NUMBER ||
		token.Type == tokenizer.STRING ||
		token.Type == tokenizer.IDENTIFIER ||
		token.Type == tokenizer.BOOLEAN ||
		token.Type == tokenizer.NULL {
		return true
	}

	return false
}

// validateVariables validates template variables and directives in a parsed statement
func validateVariables(statement cmn.StatementNode, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	// Process all clauses in the statement
	for _, clause := range statement.Clauses() {
		tokens := clause.RawTokens()
		// true is for, false is if
		var nest []bool

		// replace dummy literal tokens with actual values
		var (
			insertions   []tokenInsertion
			replacements []tokenReplacement
		)

		for i, token := range tokens {
			if token.Directive != nil {
				switch token.Directive.Type {
				case "variable", "const":
					var (
						value     any
						valueType string
						ok        bool
					)

					// 値の取得（ソースが異なるだけ）

					if token.Directive.Type == "variable" {
						value, valueType, ok = validateVariableDirective(token, paramNs, perr)
					} else {
						value, valueType, ok = validateConstDirective(token, constNs, perr)
					}

					if ok {
						// 直後のトークンがダミーリテラル、リテラル、または識別子の場合
						if i+1 < len(tokens) && isDummyLiteral(tokens[i+1]) {
							// DUMMY_START/DUMMY_ENDでラップして、ダミーリテラルを実際の値に置換
							literalTokens := createLiteralTokens(value, valueType, token.Position)

							// 元のダミーリテラルを置換
							replacements = append(replacements, tokenReplacement{
								startIndex: i + 1,
								endIndex:   i + 2,
								tokens:     literalTokens, // DUMMY_START + 実際の値 + DUMMY_END
							})
						} else {
							// ダミーリテラルがない場合は、DUMMY_START/DUMMY_ENDでラップして挿入
							literalTokens := createLiteralTokens(value, valueType, token.Position)
							insertions = append(insertions, tokenInsertion{
								index:  i,
								tokens: literalTokens,
							})
						}
					}
				case "if":
					validateIfDirective(token, paramNs, constNs, perr)

					nest = append(nest, false)
				case "for":
					// Handle for loop - process the loop body with extended namespace
					processForLoop(token, paramNs, constNs, perr)

					nest = append(nest, true)
				case "elseif":
					validateElseIfDirective(token, paramNs, constNs, perr)
				case "else":
					// No specific validation needed for else
				case "end":
					if nest[len(nest)-1] {
						paramNs.ExitLoop()
					}

					nest = nest[:len(nest)-1]
				}
			}
		}

		// Perform replacements (execute from the back to prevent index shifting)
		for i := len(replacements) - 1; i >= 0; i-- {
			replacement := replacements[i]
			replaceTokens(clause, replacement.startIndex, replacement.endIndex, replacement.tokens)
		}

		// Perform insertions (execute from the back to prevent index shifting)
		for i := len(insertions) - 1; i >= 0; i-- {
			insertion := insertions[i]
			// Use ClauseNode's InsertTokensAfterIndex method to insert tokens
			clause.InsertTokensAfterIndex(insertion.index, insertion.tokens)
		}

		// Clean up: remove literal NULL tokens that follow dummy wrappers
		toks := clause.RawTokens()
		for i := range len(toks) - 1 {
			if toks[i].Type == tokenizer.DUMMY_END && toks[i+1].Type == tokenizer.NULL {
				clause.ReplaceTokens(i+1, i+2, tokenizer.Token{Type: tokenizer.WHITESPACE, Value: ""})
			}
		}
	}
}

// isSystemColumn checks if a variable name is a known system column
func isSystemColumn(varName string) bool {
	systemColumns := []string{
		"created_at", "updated_at", "created_by", "updated_by", "version",
	}
	for _, col := range systemColumns {
		if varName == col {
			return true
		}
	}

	return false
}

// validateVariableDirective validates a variable directive
func validateVariableDirective(token tokenizer.Token, paramNs *cmn.Namespace, perr *cmn.ParseError) (any, string, bool) {
	expression := extractExpressionFromDirective(token.Value, "/*=", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid variable directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return nil, "", false
	}

	// Check if this is a system column - if so, skip CEL validation
	if isSystemColumn(strings.TrimSpace(expression)) {
		// Return a placeholder value for system columns
		// The actual value will be injected at runtime from context
		return "SYSTEM_VALUE_" + strings.TrimSpace(expression), "string", true
	}

	// Validate the expression using parameter CEL
	if value, valueType, err := paramNs.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", expression, err, token.Position.String()))
		return nil, "", false
	} else {
		return value, valueType, true
	}
}

// validateConstDirective validates a const directive
func validateConstDirective(token tokenizer.Token, constNs *cmn.Namespace, perr *cmn.ParseError) (any, string, bool) {
	expression := extractExpressionFromDirective(token.Value, "/*$", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid const directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return nil, "", false
	}
	// Validate as environment expression
	if value, valueType, err := constNs.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in environment expression '%s': %w at %s", expression, err, token.Position.String()))
		return nil, "", false
	} else {
		return value, valueType, true
	}
}

// validateIfDirective validates an if directive
func validateIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: if directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in if directive '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// validateElseIfDirective validates an elseif directive
func validateElseIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: elseif directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in elseif directive '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// processForLoop processes a for loop directive and returns the end index
func processForLoop(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) bool {
	forDirective := token.Directive

	// Parse the for directive: "for item : items"
	parts := strings.Split(forDirective.Condition, ":")
	if len(parts) != 2 {
		perr.Add(fmt.Errorf("%w at %s: invalid for directive format, expected 'for item : items'", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return false
	}

	itemName := strings.TrimSpace(parts[0])
	itemsExpr := strings.TrimSpace(parts[1])

	// Try to evaluate the items expression with parameter namespace first
	itemsValue, _, err := paramNs.Eval(itemsExpr)
	if err != nil {
		// If evaluation fails (e.g., expression not found), use empty array
		// EnterLoop will create a dummy value for type inference
		itemsValue = []any{}
	}

	// Enter the loop with the first item (if available)
	items, ok := itemsValue.([]any)
	if !ok {
		perr.Add(fmt.Errorf("%w at %s: items expression '%s' must evaluate to a list", cmn.ErrInvalidForSnapSQL, token.Position.String(), itemsExpr))
		return false
	}

	// Enter the loop (even with empty array - EnterLoop will create dummy values for type inference)
	if err := paramNs.EnterLoop(itemName, items); err != nil {
		perr.Add(fmt.Errorf("error entering loop: %w at %s", err, token.Position.String()))
		return false
	}

	return true
}

// extractExpressionFromDirective extracts the expression from a directive comment
func extractExpressionFromDirective(value string, prefix string, suffix string) string {
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) {
		return ""
	}

	return strings.TrimSpace(value[len(prefix) : len(value)-len(suffix)])
}

// tokenInsertion はトークンの挿入情報を表します
type tokenInsertion struct {
	index  int               // 挿入位置
	tokens []tokenizer.Token // 挿入するトークン
}

// tokenReplacement はトークンの置換情報を表します
type tokenReplacement struct {
	startIndex int               // 置換開始位置
	endIndex   int               // 置換終了位置
	tokens     []tokenizer.Token // 置換するトークン群（複数可）
}

// replaceTokens は指定された範囲のトークンを新しいトークン群に置き換えます
func replaceTokens(clause cmn.ClauseNode, startIndex, endIndex int, newTokens []tokenizer.Token) {
	// ClauseNodeのReplaceTokensメソッドを使用してトークンを置換
	// 複数トークンの場合は、最初のトークンで置換し、残りを挿入
	if len(newTokens) > 0 {
		clause.ReplaceTokens(startIndex, endIndex, newTokens[0])
		// 残りのトークンを挿入
		if len(newTokens) > 1 {
			clause.InsertTokensAfterIndex(startIndex, newTokens[1:])
		}
	}
}

// createLiteralTokens は値と型に基づいてDUMMY_START/DUMMY_ENDでラップされたトークンを作成します
func createLiteralTokens(value any, valueType string, pos tokenizer.Position) []tokenizer.Token {
	// DUMMY_STARTトークン
	startToken := tokenizer.Token{
		Type:     tokenizer.DUMMY_START,
		Value:    "DUMMY_START",
		Position: pos,
	}

	// DUMMY_ENDトークン
	endToken := tokenizer.Token{
		Type:     tokenizer.DUMMY_END,
		Value:    "DUMMY_END",
		Position: pos,
	}

	// 値のトークン
	valueToken := createValueToken(value, valueType, pos)

	return []tokenizer.Token{startToken, valueToken, endToken}
}

// createValueToken は値と型に基づいて値のトークンを作成します
func createValueToken(value any, valueType string, pos tokenizer.Position) tokenizer.Token {
	var valueToken tokenizer.Token

	switch valueType {
	case "int":
		// 整数リテラル
		valueToken = tokenizer.Token{
			Type:     tokenizer.NUMBER,
			Value:    fmt.Sprintf("%d", value),
			Position: pos,
		}
	case "float":
		// 浮動小数点リテラル
		floatVal, ok := value.(float64)
		if !ok {
			// 型アサーションが失敗した場合はデフォルト値を使用
			floatVal = 0.0
		}

		valueToken = tokenizer.Token{
			Type:     tokenizer.NUMBER,
			Value:    strconv.FormatFloat(floatVal, 'f', -1, 64),
			Position: pos,
		}
	case "string":
		// 文字列リテラル（シングルクォートで囲む）
		strVal, ok := value.(string)
		if !ok {
			// 型アサーションが失敗した場合はデフォルト値を使用
			strVal = ""
		}

		valueToken = tokenizer.Token{
			Type:     tokenizer.STRING,
			Value:    fmt.Sprintf("'%s'", escapeString(strVal)),
			Position: pos,
		}
	case "bool":
		// 真偽値リテラル
		var boolStr string

		boolVal, ok := value.(bool)
		if !ok {
			// 型アサーションが失敗した場合はデフォルト値を使用
			boolVal = false
		}

		if boolVal {
			boolStr = "TRUE"
		} else {
			boolStr = "FALSE"
		}

		valueToken = tokenizer.Token{
			Type:     tokenizer.BOOLEAN,
			Value:    boolStr,
			Position: pos,
		}
	default:
		// その他の型は文字列として扱う
		valueToken = tokenizer.Token{
			Type:     tokenizer.STRING,
			Value:    fmt.Sprintf("'%s'", escapeString(fmt.Sprintf("%v", value))),
			Position: pos,
		}
	}

	return valueToken
}

// escapeString は文字列内のシングルクォートをエスケープします
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
