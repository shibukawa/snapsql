package parserstep5

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// detectDummyRanges detects dummy elements that should be removed when directive values are output.
// Dummy elements are tokens that immediately follow /*= var */, /*$ env */, or /*# end */ directives
// without any whitespace in between.
//
// Rules:
// - Single primitive elements (identifiers, strings, numbers) are marked as single dummy elements
// - Parentheses-enclosed elements are marked from opening parenthesis to matching closing parenthesis
//
// Example:
//
//	VALUES /*# for user : users */ (/*= user.id */'1', /*= user.name */'name'), /*# end */('1', 'name')
//	The last ('1', 'name') should be marked as dummy range from opening '(' to closing ')'
func detectDummyRanges(stmt cmn.StatementNode) {
	// Process all clauses in the statement
	for _, clause := range stmt.Clauses() {
		// Use RawTokens() which includes directive tokens
		detectDummyRangesInTokens(clause.RawTokens())
	}
}

// detectDummyRangesInTokens detects dummy ranges in a token slice
func detectDummyRangesInTokens(tokens []tokenizer.Token) {
	for i := range tokens {
		if tokens[i].Directive == nil {
			continue
		}

		// Check for directives that can have dummy elements
		if isDummyableDirective(tokens[i].Directive.Type) {
			dummyRange := findDummyRange(tokens, i)
			if len(dummyRange) > 0 {
				tokens[i].Directive.DummyRange = dummyRange
			}
		}
	}
}

// isDummyableDirective checks if the directive type can have dummy elements
func isDummyableDirective(directiveType string) bool {
	switch directiveType {
	case "variable", "const", "end":
		return true
	default:
		return false
	}
}

// findDummyRange finds the range of dummy tokens immediately following a directive
func findDummyRange(tokens []tokenizer.Token, directiveIndex int) []int {
	if directiveIndex >= len(tokens)-1 {
		return nil
	}

	// Look for the next non-whitespace token
	nextIndex := directiveIndex + 1
	for nextIndex < len(tokens) && tokens[nextIndex].Type == tokenizer.WHITESPACE {
		nextIndex++
	}

	if nextIndex >= len(tokens) {
		return nil
	}

	// Check if the directive token and the next token are adjacent (no whitespace between them)
	if !areTokensAdjacent(tokens[directiveIndex], tokens[nextIndex]) {
		return nil
	}

	// Determine the dummy range based on the token type
	if tokens[nextIndex].Type == tokenizer.OPENED_PARENS {
		// Find matching closing parenthesis
		return findParenthesesRange(tokens, nextIndex)
	} else if isPrimitiveToken(tokens[nextIndex].Type) {
		// Single primitive token
		return []int{nextIndex}
	}

	return nil
}

// areTokensAdjacent checks if two tokens are adjacent without whitespace between them
func areTokensAdjacent(first, second tokenizer.Token) bool {
	// Calculate the end position of the first token
	firstEnd := first.Position.Offset + len(first.Value)
	// Check if the second token starts immediately after the first one
	return firstEnd == second.Position.Offset
}

// isPrimitiveToken checks if the token type represents a primitive element
func isPrimitiveToken(tokenType tokenizer.TokenType) bool {
	switch tokenType {
	case tokenizer.STRING, tokenizer.IDENTIFIER, tokenizer.NUMBER, tokenizer.BOOLEAN, tokenizer.NULL:
		return true
	default:
		return false
	}
}

// findParenthesesRange finds the range from opening parenthesis to matching closing parenthesis
func findParenthesesRange(tokens []tokenizer.Token, startIndex int) []int {
	if startIndex >= len(tokens) || tokens[startIndex].Type != tokenizer.OPENED_PARENS {
		return nil
	}

	parenLevel := 0

	for i := startIndex; i < len(tokens); i++ {
		switch tokens[i].Type {
		case tokenizer.OPENED_PARENS:
			parenLevel++
		case tokenizer.CLOSED_PARENS:
			parenLevel--
			if parenLevel == 0 {
				// Found matching closing parenthesis
				result := make([]int, i-startIndex+1)
				for j := 0; j < len(result); j++ {
					result[j] = startIndex + j
				}

				return result
			}
		}
	}

	// No matching closing parenthesis found
	return nil
}
