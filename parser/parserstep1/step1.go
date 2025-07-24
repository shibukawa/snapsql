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

	// Insert minimal dummy literals for /*= */ directives to ensure SQL parsing succeeds
	processedTokens := insertMinimalDummyLiterals(tokens)

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
