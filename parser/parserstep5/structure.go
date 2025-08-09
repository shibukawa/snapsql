package parserstep5

import (
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// validateAndLinkDirectives validates SnapSQL directive structure and builds linking information
func validateAndLinkDirectives(stmt cmn.StatementNode, parseErr *cmn.ParseError) {
	// Validate directives in each clause separately
	for _, clause := range stmt.Clauses() {
		validateClause(clause.RawTokens(), parseErr)
	}
}

// validateClause validates directives within a single clause
func validateClause(tokens []tokenizer.Token, parseErr *cmn.ParseError) {
	var stack []DirectiveStackItem

	parenLevel := 0

	for i, token := range tokens {
		// Track parentheses nesting level
		switch token.Type {
		case tokenizer.OPENED_PARENS:
			parenLevel++
		case tokenizer.CLOSED_PARENS:
			parenLevel--
		}

		if token.Directive == nil {
			continue
		}

		// Only validate control flow directives, skip variable/const substitutions
		if !IsControlFlowDirective(token.Directive.Type) {
			continue
		}

		switch token.Directive.Type {
		case "if", "for":
			stack = append(stack, DirectiveStackItem{
				Type:       token.Directive.Type,
				TokenIndex: i,
				ParenLevel: parenLevel,
			})
		case "elseif", "else":
			if len(stack) == 0 {
				parseErr.Add(fmt.Errorf("%w: unexpected directive '%s' at line %d, column %d", cmn.ErrInvalidForSnapSQL,
					token.Directive.Type, token.Position.Line, token.Position.Column))

				continue
			}

			// elseif and else can only follow if or elseif
			stackTop := stack[len(stack)-1]
			if stackTop.Type != "if" && stackTop.Type != "elseif" {
				parseErr.Add(fmt.Errorf("%w: unexpected directive '%s' at line %d, column %d", cmn.ErrInvalidForSnapSQL,
					token.Directive.Type, token.Position.Line, token.Position.Column))

				continue
			}

			// Check parentheses level consistency
			if stackTop.ParenLevel != parenLevel {
				parseErr.Add(fmt.Errorf("%w: directive '%s' at line %d, column %d crosses parentheses boundary (started at level %d, now at level %d)", cmn.ErrInvalidForSnapSQL,
					token.Directive.Type, token.Position.Line, token.Position.Column, stackTop.ParenLevel, parenLevel))

				continue
			}

			// Link previous directive to current
			linkDirectives(tokens, stackTop.TokenIndex, i)
			// Update stack top
			stack[len(stack)-1] = DirectiveStackItem{
				Type:       token.Directive.Type,
				TokenIndex: i,
				ParenLevel: parenLevel,
			}
		case "end":
			if len(stack) == 0 {
				parseErr.Add(fmt.Errorf("%w: unexpected 'end' directive at line %d, column %d", cmn.ErrInvalidForSnapSQL,
					token.Position.Line, token.Position.Column))

				continue
			}

			// Check parentheses level consistency
			stackTop := stack[len(stack)-1]
			if stackTop.ParenLevel != parenLevel {
				parseErr.Add(fmt.Errorf("%w: directive 'end' at line %d, column %d crosses parentheses boundary (started at level %d, now at level %d)", cmn.ErrInvalidForSnapSQL,
					token.Position.Line, token.Position.Column, stackTop.ParenLevel, parenLevel))

				continue
			}

			// Link last directive to end
			linkDirectives(tokens, stack[len(stack)-1].TokenIndex, i)
			stack = stack[:len(stack)-1] // Pop from stack
		}
	}

	// Check for unclosed blocks
	for _, item := range stack {
		parseErr.Add(fmt.Errorf("%w: unclosed directive block '%s'", cmn.ErrInvalidForSnapSQL, item.Type))
	}
}

// IsControlFlowDirective checks if a directive type is a control flow directive
func IsControlFlowDirective(directiveType string) bool {
	switch directiveType {
	case "if", "elseif", "else", "for", "end":
		return true
	default:
		return false
	}
}

// DirectiveStackItem represents an item in the directive stack
type DirectiveStackItem struct {
	Type       string
	TokenIndex int
	ParenLevel int // Parentheses nesting level when this directive was encountered
}

// linkDirectives sets the NextIndex field to create directive chain
func linkDirectives(tokens []tokenizer.Token, fromIndex, toIndex int) {
	if fromIndex >= 0 && fromIndex < len(tokens) && toIndex >= 0 && toIndex < len(tokens) {
		if tokens[fromIndex].Directive != nil {
			tokens[fromIndex].Directive.NextIndex = tokens[toIndex].Index
		}
	}
}
