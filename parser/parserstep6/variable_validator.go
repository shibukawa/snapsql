package parserstep6

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

type directiveScope struct {
	isFor       bool
	startIndex  int
	loopEntered bool
}

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
func validateVariables(statement cmn.StatementNode, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError, typeInfo map[string]any) {
	// Process all clauses in the statement
	for _, clause := range statement.Clauses() {
		tokens := clause.RawTokens()
		if len(tokens) == 0 {
			continue
		}

		indexByTokenIndex := make(map[int]int, len(tokens))
		for idx, token := range tokens {
			indexByTokenIndex[token.Index] = idx
		}

		var (
			scopes       []directiveScope
			insertions   []tokenInsertion
			replacements []tokenReplacement
		)

		for i, token := range tokens {
			if token.Directive == nil {
				continue
			}

			switch token.Directive.Type {
			case "variable", "const":
				var (
					value     any
					valueType string
					ok        bool
				)

				if token.Directive.Type == "variable" {
					value, valueType, ok = validateVariableDirective(token, paramNs, perr)
				} else {
					value, valueType, ok = validateConstDirective(token, constNs, perr)
				}

				if ok {
					descriptor := buildTypeDescriptor(value, valueType)
					setTypeInfo(typeInfo, token.Position, descriptor)

					if i+1 < len(tokens) && isDummyLiteral(tokens[i+1]) {
						literalTokens := createLiteralTokens(value, valueType, token.Position)
						replacements = append(replacements, tokenReplacement{
							startIndex: i + 1,
							endIndex:   i + 2,
							tokens:     literalTokens,
						})
					} else {
						literalTokens := createLiteralTokens(value, valueType, token.Position)
						insertions = append(insertions, tokenInsertion{
							index:  i,
							tokens: literalTokens,
						})
					}
				}
			case "if":
				if validateIfDirective(token, paramNs, constNs, perr) {
					setTypeInfo(typeInfo, token.Position, "bool")
				}

				scopes = append(scopes, directiveScope{isFor: false})
			case "for":
				entered, descriptor := processForLoop(token, paramNs, constNs, perr)
				if descriptor != nil {
					setTypeInfo(typeInfo, token.Position, descriptor)
				}

				scopes = append(scopes, directiveScope{isFor: true, startIndex: i, loopEntered: entered})
			case "elseif":
				if validateElseIfDirective(token, paramNs, constNs, perr) {
					setTypeInfo(typeInfo, token.Position, "bool")
				}
			case "else":
				// No additional validation required
			case "end":
				if len(scopes) == 0 {
					perr.Add(fmt.Errorf("%w: unexpected 'end' directive at %s", cmn.ErrInvalidForSnapSQL, token.Position.String()))
					continue
				}

				scope := scopes[len(scopes)-1]
				scopes = scopes[:len(scopes)-1]

				if scope.isFor {
					if scope.loopEntered {
						_ = paramNs.ExitLoop()
					}

					validateForLoopTermination(tokens, scope.startIndex, indexByTokenIndex, perr)
				}
			}
		}

		for i := len(replacements) - 1; i >= 0; i-- {
			replacement := replacements[i]
			replaceTokens(clause, replacement.startIndex, replacement.endIndex, replacement.tokens)
		}

		for i := len(insertions) - 1; i >= 0; i-- {
			insertion := insertions[i]
			clause.InsertTokensAfterIndex(insertion.index, insertion.tokens)
		}

		toks := clause.RawTokens()
		for i := range len(toks) - 1 {
			if toks[i].Type == tokenizer.DUMMY_END && toks[i+1].Type == tokenizer.NULL {
				clause.ReplaceTokens(i+1, i+2, tokenizer.Token{Type: tokenizer.WHITESPACE, Value: ""})
			}
		}
	}
}

func validateForLoopTermination(tokens []tokenizer.Token, startIndex int, indexByTokenIndex map[int]int, perr *cmn.ParseError) {
	if startIndex < 0 || startIndex >= len(tokens) {
		return
	}

	directive := tokens[startIndex].Directive
	if directive == nil {
		return
	}

	endPos, ok := indexByTokenIndex[directive.NextIndex]
	if !ok || endPos <= startIndex {
		return
	}

	topLevel := findFirstTopLevelToken(tokens, startIndex, endPos)
	if topLevel >= 0 {
		if nextDirective := tokens[topLevel].Directive; nextDirective != nil && nextDirective.Type == "if" {
			validateForLoopConditionalBranches(tokens, topLevel, endPos, indexByTokenIndex, perr)
			return
		}
	}

	if !containsAllowedTerminator(tokens, startIndex+1, endPos) {
		perr.Add(fmt.Errorf("%w at %s: for directive body must include comma, AND, or OR", cmn.ErrInvalidForSnapSQL, tokens[startIndex].Position.String()))
	}
}

func validateForLoopConditionalBranches(tokens []tokenizer.Token, branchStart int, forEnd int, indexByTokenIndex map[int]int, perr *cmn.ParseError) {
	current := branchStart
	for current >= 0 && current < len(tokens) {
		token := tokens[current]
		if token.Directive == nil {
			break
		}

		nextPos, ok := indexByTokenIndex[token.Directive.NextIndex]
		if !ok || nextPos <= current {
			break
		}

		if !containsAllowedTerminator(tokens, current+1, nextPos) {
			perr.Add(fmt.Errorf("%w at %s: branch in for-if must include comma, AND, or OR", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		}

		if nextPos >= len(tokens) {
			break
		}

		nextToken := tokens[nextPos]
		if nextToken.Directive == nil {
			break
		}

		if nextToken.Directive.Type == "end" || nextPos >= forEnd {
			break
		}

		if nextToken.Directive.Type == "elseif" || nextToken.Directive.Type == "else" {
			current = nextPos
			continue
		}

		break
	}
}

func findFirstTopLevelToken(tokens []tokenizer.Token, startPos, endPos int) int {
	depth := 0

	for i := startPos + 1; i < endPos; i++ {
		tok := tokens[i]
		if tok.Directive != nil {
			switch tok.Directive.Type {
			case "for", "if":
				if depth == 0 {
					return i
				}

				depth++

				continue
			case "end":
				if depth > 0 {
					depth--
				}

				continue
			case "elseif", "else":
				if depth == 0 {
					return i
				}

				continue
			}
		}

		if depth > 0 {
			continue
		}

		switch tok.Type {
		case tokenizer.WHITESPACE, tokenizer.LINE_COMMENT, tokenizer.BLOCK_COMMENT, tokenizer.DUMMY_START, tokenizer.DUMMY_END, tokenizer.DUMMY_LITERAL, tokenizer.DUMMY_PLACEHOLDER:
			continue
		}

		return i
	}

	return -1
}

func containsAllowedTerminator(tokens []tokenizer.Token, startPos, endPos int) bool {
	for i := startPos; i < endPos; i++ {
		tok := tokens[i]
		if tok.Directive != nil {
			continue
		}

		switch tok.Type {
		case tokenizer.COMMA, tokenizer.AND, tokenizer.OR:
			return true
		}
	}

	return false
}

func setTypeInfo(typeInfo map[string]any, pos tokenizer.Position, descriptor any) {
	if descriptor == nil {
		return
	}

	key := pos.String()
	if existing, ok := typeInfo[key]; ok {
		existingMap, existingIsMap := existing.(map[string]any)

		newMap, newIsMap := descriptor.(map[string]any)
		if existingIsMap && newIsMap {
			maps.Copy(existingMap, newMap)

			typeInfo[key] = existingMap

			return
		}
	}

	typeInfo[key] = descriptor
}

func buildTypeDescriptor(value any, typeHint string) any {
	normalized := normalizeTypeHint(typeHint)
	if normalized == "" {
		normalized = normalizeTypeHint(inferTypeFromValue(value))
	}

	if normalized == "" {
		normalized = "any"
	}

	if before, ok := strings.CutSuffix(normalized, "[]"); ok {
		baseHint := before

		switch v := value.(type) {
		case []any:
			if len(v) > 0 {
				return []any{buildTypeDescriptor(v[0], baseHint)}
			}
		}

		return []any{buildTypeDescriptor(nil, baseHint)}
	}

	switch normalized {
	case "object", "map":
		if descriptorMap := normalizeObjectDescriptor(value); len(descriptorMap) > 0 {
			return descriptorMap
		}

		return "object"
	case "json":
		return normalized
	case "any":
		if descriptorMap := normalizeObjectDescriptor(value); len(descriptorMap) > 0 {
			return descriptorMap
		}

		if slice, ok := value.([]any); ok {
			if len(slice) > 0 {
				return []any{buildTypeDescriptor(slice[0], "")}
			}

			return []any{buildTypeDescriptor(nil, "")}
		}

		return "any"
	}

	return normalized
}

func normalizeObjectDescriptor(value any) map[string]any {
	switch m := value.(type) {
	case map[string]any:
		return buildDescriptorMap(m)
	default:
		return nil
	}
}

func buildDescriptorMap(source map[string]any) map[string]any {
	result := make(map[string]any)

	for k, v := range source {
		if k == "#" {
			continue
		}

		result[k] = buildTypeDescriptor(v, "")
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func normalizeTypeHint(typeStr string) string {
	trimmed := strings.TrimSpace(typeStr)
	if trimmed == "" {
		return ""
	}

	if strings.HasSuffix(trimmed, "[]") {
		base := normalizeTypeHint(trimmed[:len(trimmed)-2])
		if base == "" {
			base = "any"
		}

		return base + "[]"
	}

	if strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case "integer", "long", "int64":
		return "int"
	case "smallint":
		return "int16"
	case "tinyint":
		return "int8"
	case "text", "varchar", "str":
		return "string"
	case "double", "number":
		return "float"
	case "decimal", "numeric":
		return "decimal"
	case "boolean":
		return "bool"
	case "array":
		return "any[]"
	}

	return lower
}

func inferTypeFromValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return "int"
	case float64, float32:
		return "float"
	case bool:
		return "bool"
	case string:
		return "string"
	case []any:
		if len(v) == 0 {
			return "any[]"
		}

		base := inferTypeFromValue(v[0])
		if base == "" {
			base = "any"
		}

		return base + "[]"
	case map[string]any:
		if tag, ok := v["#"]; ok {
			if tagStr, ok2 := tag.(string); ok2 {
				return tagStr
			}
		}

		return "object"
	default:
		return "any"
	}
}

// isSystemColumn checks if a variable name is a known system column
func isSystemColumn(varName string) bool {
	systemColumns := []string{
		"created_at", "updated_at", "created_by", "updated_by", "version",
	}

	return slices.Contains(systemColumns, varName)
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
func validateIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) bool {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: if directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return false
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in if directive '%s': %w at %s", condition, err, token.Position.String()))
		return false
	}

	return true
}

// validateElseIfDirective validates an elseif directive
func validateElseIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) bool {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: elseif directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return false
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in elseif directive '%s': %w at %s", condition, err, token.Position.String()))
		return false
	}

	return true
}

// processForLoop processes a for loop directive and returns the end index
func processForLoop(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) (bool, any) {
	forDirective := token.Directive

	// Parse the for directive: "for item : items"
	parts := strings.Split(forDirective.Condition, ":")
	if len(parts) != 2 {
		perr.Add(fmt.Errorf("%w at %s: invalid for directive format, expected 'for item : items'", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return false, nil
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
		return false, nil
	}

	// Enter the loop (even with empty array - EnterLoop will create dummy values for type inference)
	if err := paramNs.EnterLoop(itemName, items); err != nil {
		perr.Add(fmt.Errorf("error entering loop: %w at %s", err, token.Position.String()))
		return false, nil
	}

	var itemDescriptor any

	itemType, _ := paramNs.GetLoopVariableType(itemName)

	if value, valueType, evalErr := paramNs.Eval(itemName); evalErr == nil {
		itemDescriptor = buildTypeDescriptor(value, valueType)
	} else if len(items) > 0 {
		itemDescriptor = buildTypeDescriptor(items[0], itemType)
	} else {
		itemDescriptor = buildTypeDescriptor(nil, itemType)
	}

	return true, itemDescriptor
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
