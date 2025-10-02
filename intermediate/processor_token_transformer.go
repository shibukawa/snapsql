package intermediate

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TokenTransformer transforms tokens based on statement type and system fields
type TokenTransformer struct{}

func (t *TokenTransformer) Name() string {
	return "TokenTransformer"
}

func (t *TokenTransformer) Process(ctx *ProcessingContext) error {
	// Transform tokens to include system fields without modifying StatementNode
	if len(ctx.ImplicitParams) > 0 {
		switch ctx.Statement.(type) {
		case *parser.InsertIntoStatement:
			ctx.Tokens = t.addSystemFieldsToInsertTokens(ctx.Tokens, ctx.ImplicitParams)
		case *parser.UpdateStatement:
			ctx.Tokens = t.addSystemFieldsToUpdateTokens(ctx.Tokens, ctx.ImplicitParams)
		}
	}

	return nil
}

// addSystemFieldsToInsertTokens adds system fields to INSERT statement tokens
func (t *TokenTransformer) addSystemFieldsToInsertTokens(tokens []tokenizer.Token, implicitParams []ImplicitParameter) []tokenizer.Token {
	if len(implicitParams) == 0 {
		return tokens
	}

	var result []tokenizer.Token

	// Find INSERT INTO clause and VALUES clause positions
	insertIntoEnd := -1
	valuesStart := -1
	valuesEnd := -1
	valuesNestLevel := 0

	for i, token := range tokens {
		if token.Type == tokenizer.CLOSED_PARENS && insertIntoEnd == -1 {
			// First closing parenthesis is likely the end of column list
			insertIntoEnd = i
		}

		if token.Type == tokenizer.VALUES && valuesStart == -1 {
			valuesStart = i
		}

		// Track parenthesis nesting after VALUES keyword
		if valuesStart != -1 && valuesEnd == -1 {
			switch token.Type {
			case tokenizer.OPENED_PARENS:
				valuesNestLevel++
			case tokenizer.CLOSED_PARENS:
				valuesNestLevel--
				// When nest level returns to 0, we found the matching closing parenthesis
				if valuesNestLevel == 0 {
					valuesEnd = i
				}
			}
		}
	}

	// Step 1: Add system columns to column list
	if insertIntoEnd != -1 {
		result = append(result, tokens[:insertIntoEnd]...)

		// Add system columns
		for _, param := range implicitParams {
			result = append(result, tokenizer.Token{
				Type:  tokenizer.COMMA,
				Value: ", ",
			})
			result = append(result, tokenizer.Token{
				Type:  tokenizer.IDENTIFIER,
				Value: param.Name,
			})
		}

		result = append(result, tokens[insertIntoEnd:]...)
		tokens = result
		result = nil

		// Adjust valuesEnd position due to added tokens
		valuesEnd += len(implicitParams) * 2
	}

	// Step 2: Add system values to VALUES clause
	if valuesEnd != -1 {
		result = append(result, tokens[:valuesEnd]...)

		// Add system values
		for _, param := range implicitParams {
			result = append(result, tokenizer.Token{
				Type:     tokenizer.COMMA,
				Value:    ", ",
				Position: tokenizer.Position{Line: 0, Column: 0}, // Auto-generated token
			})
			// Create a special comment that will be converted to EVAL_SYSTEM_VALUE instruction
			result = append(result, tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    fmt.Sprintf("/*# EMIT_SYSTEM_VALUE: %s */", param.Name),
				Position: tokens[valuesEnd].Position, // Use the same position as closing parenthesis
				Directive: &tokenizer.Directive{
					Type:        "system_value",
					SystemField: param.Name,
				},
			})
		}

		result = append(result, tokens[valuesEnd:]...)

		return result
	}

	return tokens
}

// addSystemFieldsToUpdateTokens adds system fields to UPDATE statement tokens
func (t *TokenTransformer) addSystemFieldsToUpdateTokens(tokens []tokenizer.Token, implicitParams []ImplicitParameter) []tokenizer.Token {
	if len(implicitParams) == 0 {
		return tokens
	}

	setIndex := -1
	insertPos := len(tokens)
	parenDepth := 0
	sawUpdate := false

	for i, token := range tokens {
		switch token.Type {
		case tokenizer.OPENED_PARENS:
			parenDepth++
		case tokenizer.CLOSED_PARENS:
			if parenDepth > 0 {
				parenDepth--
			}
		case tokenizer.UPDATE:
			if parenDepth == 0 {
				sawUpdate = true
			}
		case tokenizer.SET:
			if sawUpdate && parenDepth == 0 {
				setIndex = i
			}
		case tokenizer.WHERE, tokenizer.RETURNING:
			if sawUpdate && setIndex != -1 && parenDepth == 0 {
				insertPos = i
				goto build
			}
		case tokenizer.SEMICOLON:
			if sawUpdate && setIndex != -1 && parenDepth == 0 {
				insertPos = i
				goto build
			}
		}
	}

build:
	if setIndex == -1 {
		return tokens
	}

	working := append([]tokenizer.Token(nil), tokens...)
	replaced := make(map[string]bool)

	for _, param := range implicitParams {
		var replacedNow bool

		working, insertPos, replacedNow = replaceExistingAssignment(working, setIndex, insertPos, param.Name)
		if replacedNow {
			replaced[strings.ToLower(param.Name)] = true
		}
	}

	var position tokenizer.Position

	switch {
	case insertPos < len(working):
		position = working[insertPos].Position
	case insertPos > 0:
		position = working[insertPos-1].Position
	default:
		position = tokenizer.Position{Line: 0, Column: 0}
	}

	result := make([]tokenizer.Token, 0, len(working)+(len(implicitParams)*4))
	result = append(result, working[:insertPos]...)

	for _, param := range implicitParams {
		if replaced[strings.ToLower(param.Name)] {
			continue
		}

		result = append(result, tokenizer.Token{
			Type:     tokenizer.COMMA,
			Value:    ", ",
			Position: position,
		})
		result = append(result, tokenizer.Token{
			Type:     tokenizer.IDENTIFIER,
			Value:    param.Name,
			Position: position,
		})
		result = append(result, tokenizer.Token{
			Type:     tokenizer.EQUAL,
			Value:    " = ",
			Position: position,
		})
		result = append(result, tokenizer.Token{
			Type:      tokenizer.BLOCK_COMMENT,
			Value:     fmt.Sprintf("/*# EMIT_SYSTEM_VALUE: %s */", param.Name),
			Position:  position,
			Directive: &tokenizer.Directive{Type: "system_value", SystemField: param.Name},
		})
	}

	result = append(result, working[insertPos:]...)

	return result
}

func replaceExistingAssignment(tokens []tokenizer.Token, setIndex, insertPos int, paramName string) ([]tokenizer.Token, int, bool) {
	lowerName := strings.ToLower(paramName)
	parenDepth := 0

	var (
		equalIdx   int
		valueStart int
		valueEnd   int
		found      bool
	)

	for i := setIndex + 1; i < insertPos; i++ {
		tok := tokens[i]

		switch tok.Type {
		case tokenizer.OPENED_PARENS:
			parenDepth++
		case tokenizer.CLOSED_PARENS:
			if parenDepth > 0 {
				parenDepth--
			}
		case tokenizer.IDENTIFIER, tokenizer.RESERVED_IDENTIFIER, tokenizer.CONTEXTUAL_IDENTIFIER:
			if parenDepth != 0 {
				continue
			}

			if !strings.EqualFold(strings.TrimSpace(tok.Value), lowerName) {
				continue
			}

			equalIdx = nextNonWhitespaceIndex(tokens, i+1, insertPos)
			if equalIdx == -1 || tokens[equalIdx].Type != tokenizer.EQUAL {
				continue
			}

			valueStart = nextNonWhitespaceIndex(tokens, equalIdx+1, insertPos)
			if valueStart == -1 {
				valueStart = equalIdx + 1
			}

			valueEnd = valueStart
			valueDepth := 0

			for valueEnd < insertPos {
				t := tokens[valueEnd]
				switch t.Type {
				case tokenizer.OPENED_PARENS:
					valueDepth++
				case tokenizer.CLOSED_PARENS:
					if valueDepth > 0 {
						valueDepth--
					}
				case tokenizer.COMMA:
					if valueDepth == 0 {
						found = true
						goto exitLoops
					}
				}

				valueEnd++
			}

			found = true

			goto exitLoops
		}
	}

exitLoops:
	if !found {
		return tokens, insertPos, false
	}

	pos := tokens[equalIdx].Position
	if valueStart < len(tokens) {
		pos = tokens[valueStart].Position
	}

	directive := tokenizer.Token{
		Type:      tokenizer.BLOCK_COMMENT,
		Value:     fmt.Sprintf("/*# EMIT_SYSTEM_VALUE: %s */", paramName),
		Position:  pos,
		Directive: &tokenizer.Directive{Type: "system_value", SystemField: paramName},
	}

	tokens = append(tokens[:valueStart], append([]tokenizer.Token{directive}, tokens[valueEnd:]...)...)
	insertPos += 1 - (valueEnd - valueStart)

	return tokens, insertPos, true
}

func nextNonWhitespaceIndex(tokens []tokenizer.Token, start, limit int) int {
	for i := start; i < limit; i++ {
		if tokens[i].Type != tokenizer.WHITESPACE {
			return i
		}
	}

	return -1
}
