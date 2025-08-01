package intermediate

import (
	"fmt"
	"strings"

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

	// Detect SQL patterns that need dialect-specific handling
	dialectConversions := detectDialectPatterns(ctx.Tokens)

	// Insert dialect-specific instructions where needed
	instructions = insertDialectInstructions(instructions, dialectConversions)

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

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

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
	var exprTokens []tok.Token
	var typeTokens []tok.Token

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
				SqlFragment: fmt.Sprintf("CAST_SPLIT|CAST(%s|AS|%s)", exprSQL, typeSQL),
			},
			{
				Dialects:    []string{"postgresql"},
				SqlFragment: fmt.Sprintf("CAST_SPLIT|(%s|::|%s)", exprSQL, typeSQL),
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
	var exprTokens []tok.Token
	var typeTokens []tok.Token

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
				SqlFragment: fmt.Sprintf("CAST_SPLIT|CAST(%s|AS|%s)", exprSQL, typeSQL),
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
				SqlFragment: fmt.Sprintf("CONCAT_SPLIT|%s", convertConcatToPostgreSQL(argsSQL)),
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

// createSplitCastInstructions creates multiple EMIT_IF_DIALECT instructions for split CAST syntax
func createSplitCastInstructions(variant DialectVariant, pos string) []Instruction {
	var instructions []Instruction

	// Remove the "CAST_SPLIT|" prefix
	content := strings.TrimPrefix(variant.SqlFragment, "CAST_SPLIT|")

	// Split by "|" to get individual parts
	parts := strings.Split(content, "|")

	if len(parts) < 3 {
		// Fallback to single instruction if split format is invalid
		return []Instruction{
			{
				Op:          OpEmitIfDialect,
				Pos:         pos,
				SqlFragment: strings.TrimPrefix(variant.SqlFragment, "CAST_SPLIT|"),
				Dialects:    variant.Dialects,
			},
		}
	}

	// Create separate instructions for each part
	for i, part := range parts {
		if part != "" {
			// Skip the expression part (index 1 for CAST, index 0 for PostgreSQL)
			// The expression will be processed separately and may contain other dialect code
			isExpressionPart := false

			if len(parts) == 4 { // CAST(expr|AS|type) format
				isExpressionPart = (i == 1) // Skip the expression part
			} else if len(parts) == 3 { // expr|::|type format
				isExpressionPart = (i == 0) // Skip the expression part
			}

			if !isExpressionPart {
				instructions = append(instructions, Instruction{
					Op:          OpEmitIfDialect,
					Pos:         pos,
					SqlFragment: part,
					Dialects:    variant.Dialects,
				})
			}
		}
	}

	return instructions
}

// createSplitConcatInstructions creates multiple EMIT_IF_DIALECT instructions for split CONCAT syntax
func createSplitConcatInstructions(variant DialectVariant, pos string) []Instruction {
	var instructions []Instruction

	// Remove the "CONCAT_SPLIT|" prefix
	content := strings.TrimPrefix(variant.SqlFragment, "CONCAT_SPLIT|")

	// For MySQL/SQLite: CONCAT(args|)
	// For PostgreSQL: args with || operators
	if strings.HasPrefix(content, "CONCAT(") && strings.HasSuffix(content, "|)") {
		// MySQL/SQLite format: CONCAT(args|)
		instructions = append(instructions, Instruction{
			Op:          OpEmitIfDialect,
			Pos:         pos,
			SqlFragment: "CONCAT(",
			Dialects:    variant.Dialects,
		})

		// The arguments will be processed separately and may contain other dialect code
		// We only emit the CONCAT( and ) parts

		instructions = append(instructions, Instruction{
			Op:          OpEmitIfDialect,
			Pos:         pos,
			SqlFragment: ")",
			Dialects:    variant.Dialects,
		})
	} else {
		// PostgreSQL format: args with || operators
		// Split by || and create instructions for each operator
		parts := strings.Split(content, " || ")
		for i := 0; i < len(parts)-1; i++ {
			// Add || operator between arguments
			instructions = append(instructions, Instruction{
				Op:          OpEmitIfDialect,
				Pos:         pos,
				SqlFragment: " || ",
				Dialects:    variant.Dialects,
			})
		}
	}

	return instructions
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
func insertDialectInstructions(instructions []Instruction, conversions []DialectConversion) []Instruction {
	if len(conversions) == 0 {
		return instructions
	}

	var result []Instruction

	for _, instruction := range instructions {
		// Check if this instruction contains SQL that needs dialect conversion
		if instruction.Op == OpEmitStatic {
			// Apply all conversions to this instruction
			processedInstructions := applyDialectConversions(instruction, conversions)
			result = append(result, processedInstructions...)
		} else {
			// No conversion needed, keep original instruction
			result = append(result, instruction)
		}
	}

	return result
}

// applyDialectConversions applies all dialect conversions to a single instruction
func applyDialectConversions(instruction Instruction, conversions []DialectConversion) []Instruction {
	if len(conversions) == 0 {
		return []Instruction{instruction}
	}

	var result []Instruction
	currentValue := instruction.Value

	// Sort conversions by start position (ascending) to process from left to right
	// This ensures we process outer constructs before inner ones
	sortedConversions := make([]DialectConversion, len(conversions))
	copy(sortedConversions, conversions)

	// Simple bubble sort by start position (ascending)
	for i := 0; i < len(sortedConversions)-1; i++ {
		for j := 0; j < len(sortedConversions)-i-1; j++ {
			if sortedConversions[j].StartTokenIndex > sortedConversions[j+1].StartTokenIndex {
				sortedConversions[j], sortedConversions[j+1] = sortedConversions[j+1], sortedConversions[j]
			}
		}
	}

	// Process conversions from left to right
	remainingValue := currentValue
	var lastConversionEndPos string

	// Calculate position information for each conversion
	calculatePositionForConversion := func(conversion DialectConversion) string {
		// Use the position of the first token in the conversion
		if len(conversion.OriginalTokens) > 0 {
			firstToken := conversion.OriginalTokens[0]
			return fmt.Sprintf("%d:%d", firstToken.Position.Line, firstToken.Position.Column)
		}
		// Fallback to original instruction position
		return instruction.Pos
	}

	// Calculate position after a conversion
	calculatePositionAfterConversion := func(conversion DialectConversion) string {
		// Use the position after the last token in the conversion
		if len(conversion.OriginalTokens) > 0 {
			lastToken := conversion.OriginalTokens[len(conversion.OriginalTokens)-1]
			// Calculate position after the token (approximate)
			endColumn := lastToken.Position.Column + len(lastToken.Value)
			return fmt.Sprintf("%d:%d", lastToken.Position.Line, endColumn)
		}
		// Fallback to original instruction position
		return instruction.Pos
	}

	for _, conversion := range sortedConversions {
		originalText := buildSQLFromTokens(conversion.OriginalTokens)

		if strings.Contains(remainingValue, originalText) {
			// Split around the conversion point
			parts := strings.Split(remainingValue, originalText)

			// Calculate position for this conversion
			conversionPos := calculatePositionForConversion(conversion)

			// Add the part before the conversion
			if len(parts) > 0 && parts[0] != "" {
				// Use the position after the last conversion, or original position if this is the first
				beforePos := instruction.Pos
				if lastConversionEndPos != "" {
					beforePos = lastConversionEndPos
				}

				result = append(result, Instruction{
					Op:    OpEmitStatic,
					Pos:   beforePos,
					Value: parts[0],
				})
			}

			// Add dialect-specific instructions
			for _, variant := range conversion.Variants {
				if strings.HasPrefix(variant.SqlFragment, "CAST_SPLIT|") {
					// Handle split CAST instructions
					splitInstructions := createSplitCastInstructions(variant, conversionPos)
					result = append(result, splitInstructions...)
				} else if strings.HasPrefix(variant.SqlFragment, "CONCAT_SPLIT|") {
					// Handle split CONCAT instructions
					splitInstructions := createSplitConcatInstructions(variant, conversionPos)
					result = append(result, splitInstructions...)
				} else {
					// Handle regular instructions
					result = append(result, Instruction{
						Op:          OpEmitIfDialect,
						Pos:         conversionPos,
						SqlFragment: variant.SqlFragment,
						Dialects:    variant.Dialects,
					})
				}
			}

			// Update remainingValue to the part after the conversion
			if len(parts) > 1 {
				remainingValue = strings.Join(parts[1:], originalText)
				// Update the position for the remaining content
				lastConversionEndPos = calculatePositionAfterConversion(conversion)
			} else {
				remainingValue = ""
			}
		}
	}

	// Add any remaining part
	if remainingValue != "" {
		// Use the position after the last conversion, or original position if no conversions
		remainingPos := instruction.Pos
		if lastConversionEndPos != "" {
			remainingPos = lastConversionEndPos
		}

		result = append(result, Instruction{
			Op:    OpEmitStatic,
			Pos:   remainingPos,
			Value: remainingValue,
		})
	}

	// If no conversions were applied, return the original instruction
	if len(result) == 0 {
		result = append(result, instruction)
	}

	return result
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
