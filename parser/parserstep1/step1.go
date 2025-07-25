package parserstep1

import (
	"errors"
	"fmt"
	"strings"

	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors
var (
	ErrUnmatchedCloseParenthesis  = errors.New("unmatched close parenthesis")
	ErrUnmatchedOpenParenthesis   = errors.New("unmatched open parenthesis")
	ErrUnmatchedEndDirective      = errors.New("unmatched end directive")
	ErrDirectiveWithoutMatchingIf = errors.New("directive without matching 'if'")
	ErrEndWithoutMatchingIfOrFor  = errors.New("'end' without matching 'if' or 'for'")
	ErrUnmatchedDirective         = errors.New("unmatched directive")
	ErrSemicolonNotAtEnd          = errors.New("semicolon found in middle of SQL statement")
)

// validateParentheses checks that all parentheses are properly matched.
// Returns nil if all pairs are matched, otherwise returns an error.
func validateParentheses(tokens []tok.Token) error {
	var stack []tok.Token
	for _, t := range tokens {
		switch t.Type {
		case tok.OPENED_PARENS:
			stack = append(stack, t)
		case tok.CLOSED_PARENS:
			if len(stack) == 0 {
				return ErrUnmatchedCloseParenthesis
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		return ErrUnmatchedOpenParenthesis
	}
	return nil
}

// validateSnapSQLDirectives checks SnapSQL if/for/end directive matching.
// Returns nil if all pairs are matched, otherwise returns an error.
func validateSnapSQLDirectives(tokens []tok.Token) error {
	var stack []string
	for _, t := range tokens {
		if t.Type == tok.BLOCK_COMMENT && t.Directive != nil {
			dir := t.Directive.Type
			switch dir {
			case "if", "for":
				stack = append(stack, dir)
			case "else", "elseif":
				if len(stack) == 0 || stack[len(stack)-1] != "if" {
					return fmt.Errorf("%w: '%s'", ErrDirectiveWithoutMatchingIf, dir)
				}
				// else/elseifはpopしない
			case "end":
				if len(stack) == 0 {
					return ErrUnmatchedEndDirective
				}
				if stack[len(stack)-1] != "if" && stack[len(stack)-1] != "for" {
					return ErrEndWithoutMatchingIfOrFor
				}
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) > 0 {
		return fmt.Errorf("%w: %s", ErrUnmatchedDirective, stack[len(stack)-1])
	}
	return nil
}

// Execute receives a slice of tokenizer.Token, performs basic syntax validation,
// and inserts minimal dummy literals for /*= */ directives to ensure SQL syntax validity.
// Detailed variable validation and CEL expression processing remains in parserstep6.
func Execute(tokens []tok.Token) ([]tok.Token, error) {
	// Basic syntax validation
	if err := validateParentheses(tokens); err != nil {
		return tokens, err
	}
	if err := validateSnapSQLDirectives(tokens); err != nil {
		return tokens, err
	}

	// Validate and process semicolons
	processedTokens, err := processSemicolons(tokens)
	if err != nil {
		return tokens, err
	}

	// Insert minimal dummy literals for /*= */ directives to ensure SQL parsing succeeds
	processedTokens = insertMinimalDummyLiterals(processedTokens)

	return processedTokens, nil
}

// insertMinimalDummyLiterals inserts simple dummy literals for /*= */ directives
// to ensure SQL syntax validity. Does not perform type checking or CEL parsing.
func insertMinimalDummyLiterals(tokens []tok.Token) []tok.Token {
	var result []tok.Token

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		if token.Type == tok.BLOCK_COMMENT && isVariableDirective(token.Value) {
			// Always keep the original comment token first
			result = append(result, token)

			// Check if there's a whitespace token immediately after this comment
			shouldInsert := true
			if i+1 < len(tokens) {
				switch tokens[i+1].Type {
				case tok.NUMBER, tok.STRING, tok.BOOLEAN, tok.IDENTIFIER:
					shouldInsert = false
				}
			}
			if shouldInsert {
				// Insert DUMMY_LITERAL token immediately after the comment (before the whitespace)
				dummyLiteral := tok.Token{
					Type:     tok.DUMMY_LITERAL,
					Value:    extractVariableName(token.Value),
					Position: token.Position,
				}
				result = append(result, dummyLiteral)
			}
		} else {
			// Keep the original token
			result = append(result, token)
		}
	}

	return result
}

// isVariableDirective checks if comment is a variable directive /*= variable */ or /*$ variable */
func isVariableDirective(comment string) bool {
	trimmed := strings.TrimSpace(comment)
	return (strings.HasPrefix(trimmed, "/*=") || strings.HasPrefix(trimmed, "/*$")) && strings.HasSuffix(trimmed, "*/")
}

// extractVariableName extracts variable name from /*= variable */ or /*$ variable */ comment
func extractVariableName(comment string) string {
	trimmed := strings.TrimSpace(comment)
	// Remove /*= or /*$ and */
	var content string
	if strings.HasPrefix(trimmed, "/*=") {
		content = strings.TrimSpace(trimmed[3 : len(trimmed)-2])
	} else if strings.HasPrefix(trimmed, "/*$") {
		content = strings.TrimSpace(trimmed[3 : len(trimmed)-2])
	}
	return content
}

// processSemicolons validates semicolon placement and removes trailing semicolons.
// Only allows semicolons at the end of the SQL statement (followed only by whitespace and non-directive comments).
func processSemicolons(tokens []tok.Token) ([]tok.Token, error) {
	// Find all semicolon positions
	semicolonIndices := make([]int, 0)
	for i, token := range tokens {
		if token.Type == tok.SEMICOLON {
			semicolonIndices = append(semicolonIndices, i)
		}
	}

	// If no semicolons, return as-is
	if len(semicolonIndices) == 0 {
		return tokens, nil
	}

	// Check if there are multiple semicolons
	if len(semicolonIndices) > 1 {
		return tokens, fmt.Errorf("%w: multiple semicolons found at positions %v", ErrSemicolonNotAtEnd, semicolonIndices)
	}

	semicolonIndex := semicolonIndices[0]

	// Check if semicolon is at the end (followed only by whitespace and non-directive comments)
	if !isSemicolonAtEnd(tokens, semicolonIndex) {
		return tokens, fmt.Errorf("%w: semicolon at position %d is not at the end of statement", ErrSemicolonNotAtEnd, semicolonIndex)
	}

	// Remove the trailing semicolon
	result := make([]tok.Token, 0, len(tokens)-1)
	for i, token := range tokens {
		if i != semicolonIndex {
			result = append(result, token)
		}
	}

	return result, nil
}

// isSemicolonAtEnd checks if a semicolon at the given index is at the end of the statement.
// Returns true if the semicolon is followed only by whitespace and non-directive comments.
func isSemicolonAtEnd(tokens []tok.Token, semicolonIndex int) bool {
	// Check all tokens after the semicolon
	for i := semicolonIndex + 1; i < len(tokens); i++ {
		token := tokens[i]

		switch token.Type {
		case tok.WHITESPACE:
			// Whitespace is allowed after semicolon
			continue
		case tok.LINE_COMMENT:
			// Line comments are allowed after semicolon
			continue
		case tok.BLOCK_COMMENT:
			// Block comments are allowed, but not SnapSQL directives
			if isSnapSQLDirective(token.Value) {
				return false
			}
			continue
		case tok.EOF:
			// EOF is allowed after semicolon
			continue
		default:
			// Any other token means semicolon is not at the end
			return false
		}
	}

	return true
}

// isSnapSQLDirective checks if a comment is a SnapSQL directive
func isSnapSQLDirective(comment string) bool {
	trimmed := strings.TrimSpace(comment)
	return strings.HasPrefix(trimmed, "/*#") ||
		strings.HasPrefix(trimmed, "/*=") ||
		strings.HasPrefix(trimmed, "/*$")
}
