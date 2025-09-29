package intermediate

import (
	"fmt"
	"strings"
	"unicode"

	tok "github.com/shibukawa/snapsql/tokenizer"
)

// InstructionGenerator generates execution instructions from tokens
type InstructionGenerator struct{}

func (i *InstructionGenerator) Name() string {
	return "InstructionGenerator"
}

func (i *InstructionGenerator) Process(ctx *ProcessingContext) error {
	// Use existing GenerateInstructions function for all advanced features
	// The TokenTransformer should have already added system field tokens
	// Extract expressions from CEL expressions for backward compatibility
	expressions := make([]string, len(ctx.CELExpressions))
	for j, celExpr := range ctx.CELExpressions {
		expressions[j] = celExpr.Expression
	}

	instructions := GenerateInstructions(ctx.Tokens, expressions)

	// Normalize INSERT ... SELECT statements so system fields appear in the SELECT list instead of before the table name
	instructions = normalizeInsertSelectSystemValues(instructions)

	// Detect SQL patterns that need dialect-specific handling
	dialectConversions := detectDialectPatterns(ctx.Tokens)

	// Insert dialect-specific instructions where needed
	instructions = insertDialectInstructions(instructions, dialectConversions, ctx.Dialect)

	// Set env_index in loop instructions based on environments
	if len(ctx.Environments) > 0 {
		envs := convertEnvironmentsToEnvs(ctx.Environments)
		setEnvIndexInInstructions(envs, instructions)
	}

	ctx.Instructions = instructions

	return nil
}

// DialectVariant represents a SQL variant for different database dialects
type DialectVariant struct {
	Dialects    []string `json:"dialects"`     // Target database dialects
	SqlFragment string   `json:"sql_fragment"` // SQL fragment for these dialects
}

// DialectConversion represents a detected SQL pattern that needs dialect-specific conversion
type DialectConversion struct {
	StartTokenIndex int              `json:"start_token_index"`
	EndTokenIndex   int              `json:"end_token_index"`
	OriginalTokens  []tok.Token      `json:"original_tokens"`
	Variants        []DialectVariant `json:"variants"`
}

// detectDialectPatterns scans tokens for SQL patterns that need dialect-specific handling
func detectDialectPatterns(tokens []tok.Token) []DialectConversion {
	var conversions []DialectConversion

	dummyDepth := 0

	isDummyLiteral := func(idx int) bool {
		for j := idx - 1; j >= 0; j-- {
			t := tokens[j]
			switch t.Type {
			case tok.WHITESPACE:
				continue
			case tok.DUMMY_END:
				return true
			case tok.BLOCK_COMMENT:
				if t.Directive != nil && (t.Directive.Type == "variable" || t.Directive.Type == "const") {
					return true
				}

				return false
			default:
				return false
			}
		}

		return false
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token.Type {
		case tok.DUMMY_START:
			dummyDepth++
			continue
		case tok.DUMMY_END:
			if dummyDepth > 0 {
				dummyDepth--
			}

			continue
		}

		if dummyDepth > 0 {
			continue
		}

		// Skip header comments and directives
		if token.Type == tok.BLOCK_COMMENT && isHeaderComment(token.Value) {
			continue
		}

		if token.Directive != nil {
			continue
		}

		// Check for CAST syntax: CAST(expr AS type)
		if token.Type == tok.CAST && i+6 < len(tokens) {
			if conversion := detectCastSyntax(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}

		// Check for PostgreSQL cast syntax: expr::type
		if token.Type == tok.DOUBLE_COLON && i > 0 && i+1 < len(tokens) {
			if conversion := detectPostgreSQLCast(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}

		// Check for NOW() function
		if strings.ToUpper(strings.TrimSpace(token.Value)) == "NOW" && i+2 < len(tokens) {
			if conversion := detectNowFunction(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}

		// Check for CURRENT_TIMESTAMP
		if strings.ToUpper(strings.TrimSpace(token.Value)) == "CURRENT_TIMESTAMP" {
			conversion := &DialectConversion{
				StartTokenIndex: i,
				EndTokenIndex:   i,
				OriginalTokens:  []tok.Token{token},
				Variants: []DialectVariant{
					{
						Dialects:    []string{"postgresql", "sqlite"},
						SqlFragment: "CURRENT_TIMESTAMP",
					},
					{
						Dialects:    []string{"mysql"},
						SqlFragment: "NOW()",
					},
				},
			}
			conversions = append(conversions, *conversion)
		}

		// Check for TRUE/FALSE literals
		if token.Type == tok.BOOLEAN {
			if isDummyLiteral(i) {
				continue
			}

			upperValue := strings.ToUpper(strings.TrimSpace(token.Value))

			boolValue := "1"
			if upperValue == "FALSE" {
				boolValue = "0"
			}

			conversion := &DialectConversion{
				StartTokenIndex: i,
				EndTokenIndex:   i,
				OriginalTokens:  []tok.Token{token},
				Variants: []DialectVariant{
					{
						Dialects:    []string{"postgresql"},
						SqlFragment: upperValue,
					},
					{
						Dialects:    []string{"mysql", "sqlite"},
						SqlFragment: boolValue,
					},
				},
			}
			conversions = append(conversions, *conversion)
		}

		// Check for CONCAT function
		if strings.ToUpper(strings.TrimSpace(token.Value)) == "CONCAT" && i+1 < len(tokens) {
			if conversion := detectConcatFunction(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}

		// Check for RAND() function
		if strings.ToUpper(strings.TrimSpace(token.Value)) == "RAND" && i+2 < len(tokens) {
			if conversion := detectRandFunction(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}

		// Check for RANDOM() function
		if strings.ToUpper(strings.TrimSpace(token.Value)) == "RANDOM" && i+2 < len(tokens) {
			if conversion := detectRandomFunction(tokens, i); conversion != nil {
				conversions = append(conversions, *conversion)
				i = conversion.EndTokenIndex // Skip processed tokens

				continue
			}
		}
	}

	return conversions
}

// detectCastSyntax detects CAST(expr AS type) syntax and splits it into parts
func detectCastSyntax(tokens []tok.Token, startIndex int) *DialectConversion {
	// Expected pattern: CAST ( expr AS type )
	if startIndex+6 >= len(tokens) {
		return nil
	}

	// Check for CAST ( ... AS ... )
	if strings.TrimSpace(tokens[startIndex+1].Value) != "(" {
		return nil
	}

	// Find the matching closing parenthesis and AS keyword
	parenCount := 1
	asIndex := -1
	closeParenIndex := -1

	for i := startIndex + 2; i < len(tokens) && parenCount > 0; i++ {
		token := tokens[i]
		if token.Type == tok.OPENED_PARENS {
			parenCount++
		} else if token.Type == tok.CLOSED_PARENS {
			parenCount--
			if parenCount == 0 {
				closeParenIndex = i
				break
			}
		} else if parenCount == 1 && token.Type == tok.AS {
			asIndex = i
		}
	}

	if asIndex == -1 || closeParenIndex == -1 {
		return nil
	}

	// Extract expression and type
	var (
		exprTokens []tok.Token
		typeTokens []tok.Token
	)

	for i := startIndex + 2; i < asIndex; i++ {
		exprTokens = append(exprTokens, tokens[i])
	}

	for i := asIndex + 1; i < closeParenIndex; i++ {
		typeTokens = append(typeTokens, tokens[i])
	}

	// Build SQL fragments
	exprSQL := buildSQLFromTokens(exprTokens)
	typeSQL := buildSQLFromTokens(typeTokens)

	// Create a special conversion that indicates this should be split
	return &DialectConversion{
		StartTokenIndex: startIndex,
		EndTokenIndex:   closeParenIndex,
		OriginalTokens:  tokens[startIndex : closeParenIndex+1],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"mysql", "sqlite"},
				SqlFragment: fmt.Sprintf("CAST_SPLIT|%s|AS|%s", exprSQL, typeSQL),
			},
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: fmt.Sprintf("CAST_SPLIT|(%s)|::|%s", exprSQL, typeSQL),
			},
		},
	}
}

// detectPostgreSQLCast detects expr::type syntax (including complex expressions)
func detectPostgreSQLCast(tokens []tok.Token, colonIndex int) *DialectConversion {
	// Expected pattern: expr :: type
	if colonIndex == 0 || colonIndex+1 >= len(tokens) {
		return nil
	}

	// Find the start of the expression (handle parentheses and complex expressions)
	exprStartIndex := colonIndex - 1

	// If the previous token is a closing parenthesis, find the matching opening parenthesis
	if strings.TrimSpace(tokens[colonIndex-1].Value) == ")" {
		parenCount := 1
		for i := colonIndex - 2; i >= 0 && parenCount > 0; i-- {
			token := tokens[i]
			if token.Type == tok.CLOSED_PARENS {
				parenCount++
			} else if token.Type == tok.OPENED_PARENS {
				parenCount--
				if parenCount == 0 {
					exprStartIndex = i
					break
				}
			}
		}
	} else {
		// For simple expressions, look for word boundaries
		// Go back to find the start of the identifier/expression
		for i := colonIndex - 1; i >= 0; i-- {
			token := tokens[i]

			// Stop at keywords, operators, or punctuation that would end an expression
			if isExpressionBoundary(token) {
				exprStartIndex = i + 1
				break
			}

			// If we reach the beginning, start from index 0
			if i == 0 {
				exprStartIndex = 0
				break
			}
		}
	}

	// Find the end of the type (handle complex types like DECIMAL(10,2))
	typeEndIndex := colonIndex + 1

	// If the type has parentheses, find the matching closing parenthesis
	if colonIndex+2 < len(tokens) && tokens[colonIndex+2].Type == tok.OPENED_PARENS {
		parenCount := 1
		for i := colonIndex + 3; i < len(tokens) && parenCount > 0; i++ {
			token := tokens[i]
			if token.Type == tok.OPENED_PARENS {
				parenCount++
			} else if token.Type == tok.CLOSED_PARENS {
				parenCount--
				if parenCount == 0 {
					typeEndIndex = i
					break
				}
			}
		}
	} else {
		// For simple types, look for word boundaries
		for i := colonIndex + 1; i < len(tokens); i++ {
			token := tokens[i]

			// Stop at keywords, operators, or punctuation that would end a type
			if isTypeBoundary(token) {
				typeEndIndex = i - 1
				break
			}

			// If we reach the end, end at the last token
			if i == len(tokens)-1 {
				typeEndIndex = i
				break
			}
		}
	}

	// Extract expression and type tokens
	var (
		exprTokens []tok.Token
		typeTokens []tok.Token
	)

	for i := exprStartIndex; i < colonIndex; i++ {
		exprTokens = append(exprTokens, tokens[i])
	}

	for i := colonIndex + 1; i <= typeEndIndex; i++ {
		typeTokens = append(typeTokens, tokens[i])
	}

	// Build SQL fragments
	exprSQL := buildSQLFromTokens(exprTokens)
	typeSQL := buildSQLFromTokens(typeTokens)

	// Create a special conversion that indicates this should be split
	return &DialectConversion{
		StartTokenIndex: exprStartIndex,
		EndTokenIndex:   typeEndIndex,
		OriginalTokens:  tokens[exprStartIndex : typeEndIndex+1],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: fmt.Sprintf("CAST_SPLIT|%s|::|%s", exprSQL, typeSQL),
			},
			{
				Dialects:    []string{"mysql", "sqlite"},
				SqlFragment: fmt.Sprintf("CAST_SPLIT|%s|AS|%s", exprSQL, typeSQL),
			},
		},
	}
}

// detectNowFunction detects NOW() function
func detectNowFunction(tokens []tok.Token, startIndex int) *DialectConversion {
	// Expected pattern: NOW ( )
	if startIndex+2 >= len(tokens) {
		return nil
	}

	if strings.TrimSpace(tokens[startIndex+1].Value) != "(" ||
		strings.TrimSpace(tokens[startIndex+2].Value) != ")" {
		return nil
	}

	return &DialectConversion{
		StartTokenIndex: startIndex,
		EndTokenIndex:   startIndex + 2,
		OriginalTokens:  tokens[startIndex : startIndex+3],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"mysql"},
				SqlFragment: "NOW()",
			},
			{
				Dialects:    []string{"postgresql", "sqlite"},
				SqlFragment: "CURRENT_TIMESTAMP",
			},
		},
	}
}

// detectConcatFunction detects CONCAT(...) function and splits it into parts
func detectConcatFunction(tokens []tok.Token, startIndex int) *DialectConversion {
	// Expected pattern: CONCAT ( args )
	if startIndex+2 >= len(tokens) {
		return nil
	}

	if strings.TrimSpace(tokens[startIndex+1].Value) != "(" {
		return nil
	}

	// Find the matching closing parenthesis
	parenCount := 1
	closeParenIndex := -1

	for i := startIndex + 2; i < len(tokens) && parenCount > 0; i++ {
		token := tokens[i]
		if token.Type == tok.OPENED_PARENS {
			parenCount++
		} else if token.Type == tok.CLOSED_PARENS {
			parenCount--
			if parenCount == 0 {
				closeParenIndex = i
				break
			}
		}
	}

	if closeParenIndex == -1 {
		return nil
	}

	// Extract arguments
	var argTokens []tok.Token
	for i := startIndex + 2; i < closeParenIndex; i++ {
		argTokens = append(argTokens, tokens[i])
	}

	argsSQL := buildSQLFromTokens(argTokens)

	// Create a special conversion that indicates this should be split
	return &DialectConversion{
		StartTokenIndex: startIndex,
		EndTokenIndex:   closeParenIndex,
		OriginalTokens:  tokens[startIndex : closeParenIndex+1],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"mysql", "sqlite"},
				SqlFragment: fmt.Sprintf("CONCAT_SPLIT|CONCAT(%s|)", argsSQL),
			},
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: "CONCAT_SPLIT|" + convertConcatToPostgreSQL(argsSQL),
			},
		},
	}
}

// buildSQLFromTokens reconstructs SQL from tokens
func buildSQLFromTokens(tokens []tok.Token) string {
	var result strings.Builder
	for _, token := range tokens {
		result.WriteString(token.Value)
	}

	return strings.TrimSpace(result.String())
}

// convertConcatToPostgreSQL converts CONCAT arguments to PostgreSQL || syntax
func convertConcatToPostgreSQL(args string) string {
	// Simple implementation: split by comma and join with ||
	// This is a simplified version - a full implementation would need proper parsing
	parts := strings.Split(args, ",")

	var trimmedParts []string
	for _, part := range parts {
		trimmedParts = append(trimmedParts, strings.TrimSpace(part))
	}

	return strings.Join(trimmedParts, " || ")
}

// isExpressionBoundary checks if a token marks the boundary of an expression
func isExpressionBoundary(token tok.Token) bool {
	// Check token types first (most efficient)
	switch token.Type {
	case tok.SELECT, tok.FROM, tok.WHERE, tok.AND, tok.OR, tok.ORDER, tok.GROUP, tok.HAVING,
		tok.INSERT, tok.UPDATE, tok.DELETE, tok.SET, tok.VALUES, tok.INTO, tok.AS,
		tok.JOIN, tok.LEFT, tok.RIGHT, tok.INNER, tok.OUTER, tok.ON, tok.UNION,
		tok.LIMIT, tok.OFFSET, tok.DISTINCT, tok.ALL, tok.CASE, tok.WHEN,
		tok.THEN, tok.ELSE, tok.END:
		return true
	case tok.COMMA, tok.SEMICOLON, tok.EQUAL, tok.NOT_EQUAL, tok.LESS_THAN, tok.GREATER_THAN,
		tok.LESS_EQUAL, tok.GREATER_EQUAL, tok.PLUS, tok.MINUS,
		tok.MULTIPLY, tok.DIVIDE, tok.MODULO:
		return true
	}

	// Check for keywords that don't have dedicated token types
	upperToken := strings.ToUpper(strings.TrimSpace(token.Value))
	stringOnlyKeywords := []string{
		"EXCEPT", "INTERSECT", "IF", "ELSEIF", "FOR", "WHILE", "RETURN",
	}

	for _, keyword := range stringOnlyKeywords {
		if upperToken == keyword {
			return true
		}
	}

	return false
}

// isTypeBoundary checks if a token marks the boundary of a type specification
func isTypeBoundary(token tok.Token) bool {
	// Check token types first (most efficient)
	switch token.Type {
	case tok.AS, tok.FROM, tok.WHERE, tok.AND, tok.OR, tok.ORDER, tok.GROUP, tok.HAVING,
		tok.JOIN, tok.LEFT, tok.RIGHT, tok.INNER, tok.OUTER, tok.ON, tok.UNION,
		tok.LIMIT, tok.OFFSET, tok.DISTINCT, tok.ALL, tok.CASE, tok.WHEN,
		tok.THEN, tok.ELSE, tok.END:
		return true
	case tok.COMMA, tok.SEMICOLON, tok.EQUAL, tok.NOT_EQUAL, tok.LESS_THAN, tok.GREATER_THAN,
		tok.LESS_EQUAL, tok.GREATER_EQUAL, tok.PLUS, tok.MINUS,
		tok.MULTIPLY, tok.DIVIDE, tok.MODULO:
		return true
	}

	// Check for keywords that don't have dedicated token types
	upperToken := strings.ToUpper(strings.TrimSpace(token.Value))
	stringOnlyKeywords := []string{
		"EXCEPT", "INTERSECT", "IF", "ELSEIF", "FOR", "WHILE", "RETURN",
	}

	for _, keyword := range stringOnlyKeywords {
		if upperToken == keyword {
			return true
		}
	}

	return false
}

// insertDialectInstructions inserts dialect-specific instructions into the instruction stream
func insertDialectInstructions(instructions []Instruction, conversions []DialectConversion, dialect string) []Instruction {
	if len(conversions) == 0 {
		return instructions
	}

	var result []Instruction

	for _, instruction := range instructions {
		if instruction.Op == OpEmitStatic {
			processed := applyDialectConversions(instruction, conversions, dialect)
			result = append(result, processed...)
		} else {
			result = append(result, instruction)
		}
	}

	return mergeAdjacentStaticInstructions(result)
}

// applyDialectConversions applies all dialect conversions to a single instruction
func applyDialectConversions(instruction Instruction, conversions []DialectConversion, dialect string) []Instruction {
	if len(conversions) == 0 {
		return []Instruction{instruction}
	}

	remaining := instruction.Value
	if remaining == "" {
		return []Instruction{instruction}
	}
	// sort by start index
	sorted := make([]DialectConversion, len(conversions))
	copy(sorted, conversions)

	for i := range len(sorted) - 1 {
		for j := range len(sorted) - i - 1 {
			if sorted[j].StartTokenIndex > sorted[j+1].StartTokenIndex {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	var out []Instruction

	for _, conv := range sorted {
		original := buildSQLFromTokens(conv.OriginalTokens)

		idx := strings.Index(remaining, original)
		if idx == -1 {
			continue
		}

		if pre := remaining[:idx]; pre != "" {
			out = append(out, Instruction{Op: OpEmitStatic, Pos: instruction.Pos, Value: pre})
		}

		chosen := chooseVariant(conv.Variants, dialect)
		out = append(out, expandVariant(chosen, instruction.Pos))
		remaining = remaining[idx+len(original):]
	}

	if remaining != "" {
		out = append(out, Instruction{Op: OpEmitStatic, Pos: instruction.Pos, Value: remaining})
	}

	if len(out) == 0 {
		return []Instruction{instruction}
	}

	return out
}

func chooseVariant(vars []DialectVariant, dialect string) DialectVariant {
	if len(vars) == 0 {
		return DialectVariant{}
	}

	norm := func(s string) string {
		s = strings.ToLower(s)
		switch s {
		case "postgresql", "pg":
			return "postgres"
		case "sqlite3":
			return "sqlite"
		default:
			return s
		}
	}
	d := norm(dialect)

	for _, v := range vars {
		for _, vd := range v.Dialects {
			if norm(vd) == d {
				return v
			}
		}
	}

	return vars[0]
}

func expandVariant(v DialectVariant, pos string) Instruction {
	sql := v.SqlFragment
	if strings.HasPrefix(sql, "CAST_SPLIT|") {
		parts := strings.Split(strings.TrimPrefix(sql, "CAST_SPLIT|"), "|")
		if len(parts) >= 3 {
			expr := parts[0]
			op := parts[1]
			typePart := strings.Join(parts[2:], "|")

			switch op {
			case "AS":
				return Instruction{Op: OpEmitStatic, Pos: pos, Value: fmt.Sprintf("CAST(%s AS %s)", expr, typePart)}
			case "::":
				return Instruction{Op: OpEmitStatic, Pos: pos, Value: fmt.Sprintf("%s::%s", expr, typePart)}
			}
		}

		return Instruction{Op: OpEmitStatic, Pos: pos, Value: strings.ReplaceAll(strings.ReplaceAll(sql, "CAST_SPLIT|", ""), "|", "")}
	}

	if strings.HasPrefix(sql, "CONCAT_SPLIT|") {
		content := strings.TrimPrefix(sql, "CONCAT_SPLIT|")
		if strings.HasPrefix(content, "CONCAT(") && strings.HasSuffix(content, "|)") {
			inner := strings.TrimSuffix(strings.TrimPrefix(content, "CONCAT("), "|)")
			return Instruction{Op: OpEmitStatic, Pos: pos, Value: "CONCAT(" + inner + ")"}
		}

		return Instruction{Op: OpEmitStatic, Pos: pos, Value: content}
	}

	return Instruction{Op: OpEmitStatic, Pos: pos, Value: sql}
}

func normalizeInsertSelectSystemValues(instructions []Instruction) []Instruction {
	type pendingSystemField struct {
		instruction  Instruction
		separator    string
		separatorPos string
	}

	var (
		result         []Instruction
		pendingSystem  []pendingSystemField
		inInsert       bool
		isInsertSelect bool
		sawSelect      bool
	)

	appendPending := func(pos string) {
		if len(pendingSystem) == 0 {
			return
		}
		for _, pending := range pendingSystem {
			sep := pending.separator
			if sep == "" {
				sep = ", "
			}

			sepPos := pending.separatorPos
			if sepPos == "" {
				sepPos = pos
			}

			result = append(result, Instruction{Op: OpEmitStatic, Pos: sepPos, Value: sep})
			result = append(result, pending.instruction)
		}
		pendingSystem = nil
	}

	resetState := func() {
		inInsert = false
		isInsertSelect = false
		sawSelect = false
		pendingSystem = nil
	}

	for idx := 0; idx < len(instructions); idx++ {
		inst := instructions[idx]

		if inst.Op == OpEmitStatic {
			upperValue := strings.ToUpper(inst.Value)

			if strings.Contains(upperValue, "INSERT INTO") {
				if idx := strings.Index(upperValue, "INSERT INTO"); idx >= 0 {
					insertEnd := idx + len("INSERT INTO")
					j := insertEnd
					for j < len(inst.Value) && unicode.IsSpace(rune(inst.Value[j])) {
						j++
					}

					if j < len(inst.Value) && inst.Value[j] == ',' {
						remainder := inst.Value[j+1:]
						if len(remainder) > 0 && (remainder[0] == ' ' || remainder[0] == '\t') {
							remainder = remainder[1:]
						}

						inst.Value = inst.Value[:j] + remainder

						if j == insertEnd {
							inst.Value = inst.Value[:j] + " " + inst.Value[j:]
						}

						upperValue = strings.ToUpper(inst.Value)
					}
				}

				inInsert = true
				sawSelect = false
				pendingSystem = nil
				isInsertSelect = false

				for lookahead := idx; lookahead < len(instructions); lookahead++ {
					next := instructions[lookahead]
					if next.Op != OpEmitStatic {
						continue
					}

					nextUpper := strings.ToUpper(next.Value)
					if strings.Contains(nextUpper, "VALUES") {
						break
					}

					if strings.Contains(nextUpper, "SELECT") {
						isInsertSelect = true
						break
					}
				}
			}

			if inInsert && isInsertSelect {
				if strings.Contains(upperValue, "SELECT") {
					sawSelect = true
				}

				if sawSelect && len(pendingSystem) > 0 {
					if idxFrom := strings.Index(upperValue, "FROM"); idxFrom >= 0 {
						before := inst.Value[:idxFrom]
						after := inst.Value[idxFrom:]

						if before != "" {
							result = append(result, Instruction{Op: OpEmitStatic, Pos: inst.Pos, Value: before})
						}

						appendPending(inst.Pos)

						if after != "" {
							result = append(result, Instruction{Op: OpEmitStatic, Pos: inst.Pos, Value: after})
						}

						resetState()
						continue
					}
				}
			}

			result = append(result, inst)

			if inInsert {
				if !isInsertSelect && strings.Contains(upperValue, "VALUES") {
					resetState()
				} else if strings.Contains(upperValue, ";") {
					if len(pendingSystem) > 0 {
						appendPending(inst.Pos)
					}
					resetState()
				}
			}

			continue
		}

		if inInsert && isInsertSelect && !sawSelect && inst.Op == OpEmitSystemValue {
			separator := ""
			separatorPos := ""

			if len(result) > 0 {
				if last := result[len(result)-1]; last.Op == OpEmitStatic {
					if strings.TrimSpace(last.Value) == "," {
						separator = last.Value
						separatorPos = last.Pos
						result = result[:len(result)-1]
					}
				}
			}

			pendingSystem = append(pendingSystem, pendingSystemField{
				instruction:  inst,
				separator:    separator,
				separatorPos: separatorPos,
			})
			continue
		}

		result = append(result, inst)
	}

	if len(pendingSystem) > 0 {
		for _, pending := range pendingSystem {
			sep := pending.separator
			if sep == "" {
				sep = ", "
			}

			sepPos := pending.separatorPos
			if sepPos == "" {
				sepPos = pending.instruction.Pos
			}

			result = append(result, Instruction{Op: OpEmitStatic, Pos: sepPos, Value: sep})
			result = append(result, pending.instruction)
		}
	}

	return result
}

func mergeAdjacentStaticInstructions(ins []Instruction) []Instruction {
	if len(ins) == 0 {
		return ins
	}

	var (
		out []Instruction
		buf strings.Builder
		pos string
	)

	flush := func() {
		if buf.Len() > 0 {
			out = append(out, Instruction{Op: OpEmitStatic, Pos: pos, Value: buf.String()})
			buf.Reset()
		}
	}

	for _, in := range ins {
		if in.Op == OpEmitStatic {
			if buf.Len() == 0 {
				pos = in.Pos
			}

			buf.WriteString(in.Value)
		} else {
			flush()

			out = append(out, in)
		}
	}

	flush()

	return out
}

// detectRandFunction detects RAND() function
func detectRandFunction(tokens []tok.Token, startIndex int) *DialectConversion {
	// Expected pattern: RAND ( )
	if startIndex+2 >= len(tokens) {
		return nil
	}

	if strings.TrimSpace(tokens[startIndex+1].Value) != "(" ||
		strings.TrimSpace(tokens[startIndex+2].Value) != ")" {
		return nil
	}

	return &DialectConversion{
		StartTokenIndex: startIndex,
		EndTokenIndex:   startIndex + 2,
		OriginalTokens:  tokens[startIndex : startIndex+3],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"mysql", "sqlite"},
				SqlFragment: "RAND()",
			},
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: "RANDOM()",
			},
		},
	}
}

// detectRandomFunction detects RANDOM() function
func detectRandomFunction(tokens []tok.Token, startIndex int) *DialectConversion {
	// Expected pattern: RANDOM ( )
	if startIndex+2 >= len(tokens) {
		return nil
	}

	if strings.TrimSpace(tokens[startIndex+1].Value) != "(" ||
		strings.TrimSpace(tokens[startIndex+2].Value) != ")" {
		return nil
	}

	return &DialectConversion{
		StartTokenIndex: startIndex,
		EndTokenIndex:   startIndex + 2,
		OriginalTokens:  tokens[startIndex : startIndex+3],
		Variants: []DialectVariant{
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: "RANDOM()",
			},
			{
				Dialects:    []string{"mysql", "sqlite"},
				SqlFragment: "RAND()",
			},
		},
	}
}
