package parserstep1

import (
	"errors"

	tokenizer "github.com/shibukawa/snapsql/tokenizer"
)

// validateParentheses checks that all parentheses are properly matched.
// Returns nil if all pairs are matched, otherwise returns an error.
func validateParentheses(tokens []tokenizer.Token) error {
	var stack []tokenizer.Token
	for _, tok := range tokens {
		switch tok.Type {
		case tokenizer.OPENED_PARENS:
			stack = append(stack, tok)
		case tokenizer.CLOSED_PARENS:
			if len(stack) == 0 {
				return errors.New("unmatched close parenthesis")
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		return errors.New("unmatched open parenthesis")
	}
	return nil
}

// validateSnapSQLDirectives checks SnapSQL if/for/end directive matching.
// Returns nil if all pairs are matched, otherwise returns an error.
func validateSnapSQLDirectives(tokens []tokenizer.Token) error {
	var stack []string
	for _, tok := range tokens {
		if tok.Type == tokenizer.BLOCK_COMMENT && tok.Directive != nil {
			dir := tok.Directive.Type
			switch dir {
			case "if", "for":
				stack = append(stack, dir)
			case "else", "elseif":
				if len(stack) == 0 || stack[len(stack)-1] != "if" {
					return errors.New("'" + dir + "' without matching 'if'")
				}
				// else/elseifはpopしない
			case "end":
				if len(stack) == 0 {
					return errors.New("unmatched end directive")
				}
				if stack[len(stack)-1] != "if" && stack[len(stack)-1] != "for" {
					return errors.New("'end' without matching 'if' or 'for'")
				}
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) > 0 {
		return errors.New("unmatched directive: " + stack[len(stack)-1])
	}
	return nil
}

// Execute receives a slice of tokenizer.Token and returns error if parentheses or SnapSQL directives are not matched.
func Execute(tokens []tokenizer.Token) error {
	if err := validateParentheses(tokens); err != nil {
		return err
	}
	if err := validateSnapSQLDirectives(tokens); err != nil {
		return err
	}
	// ...他の検証もここで呼ぶ予定...
	return nil
}
