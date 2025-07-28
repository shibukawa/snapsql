package intermediate

import (
	"fmt"

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

	for i, token := range tokens {
		if token.Type == tokenizer.CLOSED_PARENS && insertIntoEnd == -1 {
			// First closing parenthesis is likely the end of column list
			insertIntoEnd = i
		}
		if token.Type == tokenizer.VALUES && valuesStart == -1 {
			valuesStart = i
		}
		if token.Type == tokenizer.CLOSED_PARENS && valuesStart != -1 && valuesEnd == -1 {
			// Closing parenthesis after VALUES
			valuesEnd = i
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
	var result []tokenizer.Token

	// Find the end of SET clause (before WHERE, or end of statement)
	setEnd := len(tokens)
	for i, token := range tokens {
		if token.Type == tokenizer.WHERE {
			setEnd = i
			break
		}
	}

	// Add system fields to SET clause
	result = append(result, tokens[:setEnd]...)

	// Add system fields
	for _, param := range implicitParams {
		result = append(result, tokenizer.Token{
			Type:     tokenizer.COMMA,
			Value:    ", ",
			Position: tokenizer.Position{Line: 0, Column: 0}, // Auto-generated token
		})
		result = append(result, tokenizer.Token{
			Type:     tokenizer.IDENTIFIER,
			Value:    param.Name,
			Position: tokenizer.Position{Line: 0, Column: 0}, // Auto-generated token
		})
		result = append(result, tokenizer.Token{
			Type:     tokenizer.EQUAL,
			Value:    " = ",
			Position: tokenizer.Position{Line: 0, Column: 0}, // Auto-generated token
		})
		result = append(result, tokenizer.Token{
			Type:     tokenizer.BLOCK_COMMENT,
			Value:    fmt.Sprintf("/*= %s */NOW()", param.Name),
			Position: tokenizer.Position{Line: 0, Column: 0}, // Auto-generated token
		})
	}

	result = append(result, tokens[setEnd:]...)

	return result
}
