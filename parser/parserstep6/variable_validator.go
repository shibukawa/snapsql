package parserstep6

import (
	"fmt"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// validateVariables validates template variables and directives in a parsed statement
func validateVariables(statement cmn.StatementNode, paramNamespace *cmn.Namespace, constNamespace *cmn.Namespace, perr *cmn.ParseError) {
	// Process all clauses in the statement
	for _, clause := range statement.Clauses() {
		tokens := clause.RawTokens()
		processTokens(tokens, paramNamespace, constNamespace, perr)
	}
}

// processTokens processes a sequence of tokens for directive validation
func processTokens(tokens []tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	i := 0

	for i < len(tokens) {
		token := tokens[i]
		if token.Directive != nil {
			switch token.Directive.Type {
			case "variable":
				validateVariableDirective(token, paramNs, perr)
			case "const":
				validateConstDirective(token, constNs, perr)
			case "if":
				validateIfDirective(token, paramNs, constNs, perr)
			case "for":
				// Handle for loop - process the loop body with extended namespace
				ok, endIndex := processForLoop(tokens, i, paramNs, constNs, perr)
				if ok {
					// Process tokens inside the loop with the extended namespace
					loopTokens := tokens[i+1 : endIndex]
					processTokens(loopTokens, paramNs, constNs, perr)
					i = endIndex // Skip to after the end directive
					paramNs.ExitLoop()
					continue
				}
			case "elseif":
				validateElseIfDirective(token, paramNs, constNs, perr)
			case "else":
				// No specific validation needed for else
			case "end":
				// No specific validation needed for end
			}
		}
		i++
	}
}

// validateVariableDirective validates a variable directive
func validateVariableDirective(token tokenizer.Token, paramNs *cmn.Namespace, perr *cmn.ParseError) {
	expression := extractExpressionFromDirective(token.Value, "/*=", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid variable directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Validate the expression using parameter CEL
	if _, _, err := paramNs.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", expression, err, token.Position.String()))
	}
}

// validateConstDirective validates a const directive
func validateConstDirective(token tokenizer.Token, constNs *cmn.Namespace, perr *cmn.ParseError) {
	expression := extractExpressionFromDirective(token.Value, "/*$", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid const directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}
	// Validate as environment expression
	if _, _, err := constNs.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in environment expression '%s': %w at %s", expression, err, token.Position.String()))
	}
}

// validateIfDirective validates an if directive
func validateIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: if directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in if directive '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// validateElseIfDirective validates an elseif directive
func validateElseIfDirective(token tokenizer.Token, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) {
	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: elseif directive missing condition", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Try to evaluate with parameter namespace first
	_, _, err := paramNs.Eval(condition)
	if err != nil {
		perr.Add(fmt.Errorf("invalid condition in elseif directive '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// processForLoop processes a for loop directive and returns the end index
func processForLoop(tokens []tokenizer.Token, startIndex int, paramNs *cmn.Namespace, constNs *cmn.Namespace, perr *cmn.ParseError) (bool, int) {
	if startIndex >= len(tokens) || tokens[startIndex].Directive == nil || tokens[startIndex].Directive.Type != "for" {
		return false, startIndex
	}

	forToken := tokens[startIndex]
	forDirective := forToken.Directive

	// Parse the for directive: "for item : items"
	parts := strings.Split(forDirective.Condition, ":")
	if len(parts) != 2 {
		perr.Add(fmt.Errorf("%w at %s: invalid for directive format, expected 'for item : items'", cmn.ErrInvalidForSnapSQL, forToken.Position.String()))
		return false, startIndex
	}

	itemName := strings.TrimSpace(parts[0])
	itemsExpr := strings.TrimSpace(parts[1])

	// Find the matching end directive
	endIndex := findMatchingEndDirective(tokens, startIndex)
	if endIndex == -1 {
		perr.Add(fmt.Errorf("%w at %s: missing end directive for loop", cmn.ErrInvalidForSnapSQL, forToken.Position.String()))
		return false, startIndex
	}

	// Try to evaluate the items expression with parameter namespace first
	itemsValue, _, err := paramNs.Eval(itemsExpr)
	if err != nil {
		perr.Add(fmt.Errorf("invalid items expression in for directive '%s': %w at %s", itemsExpr, err, forToken.Position.String()))
		return false, startIndex
	}

	// Enter the loop with the first item (if available)
	items, ok := itemsValue.([]any)
	if !ok {
		perr.Add(fmt.Errorf("%w at %s: items expression '%s' must evaluate to a list", cmn.ErrInvalidForSnapSQL, forToken.Position.String(), itemsExpr))
		return false, startIndex
	}

	if len(items) == 0 {
		perr.Add(fmt.Errorf("%w at %s: items expression '%s' evaluates to an empty list", cmn.ErrInvalidForSnapSQL, forToken.Position.String(), itemsExpr))
		return false, startIndex
	}

	// Enter the loop with the first item
	if err := paramNs.EnterLoop(itemName, items); err != nil {
		perr.Add(fmt.Errorf("error entering loop: %w at %s", err, forToken.Position.String()))
		return false, startIndex
	}

	return true, endIndex
}

// findMatchingEndDirective finds the matching end directive for a control structure
func findMatchingEndDirective(tokens []tokenizer.Token, startIndex int) int {
	depth := 0
	for i := startIndex + 1; i < len(tokens); i++ {
		token := tokens[i]
		if token.Directive == nil {
			continue
		}

		switch token.Directive.Type {
		case "if", "for":
			depth++
		case "end":
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// extractExpressionFromDirective extracts the expression from a directive comment
func extractExpressionFromDirective(value string, prefix string, suffix string) string {
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) {
		return ""
	}
	return strings.TrimSpace(value[len(prefix) : len(value)-len(suffix)])
}
