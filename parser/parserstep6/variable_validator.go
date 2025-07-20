package parserstep6

import (
	"fmt"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// validateVariables validates template variables and directives in a parsed statement
func validateVariables(statement cmn.StatementNode, namespace *cmn.Namespace, perr *cmn.ParseError) {
	// Process all clauses in the statement
	processStatement(statement, namespace, perr)
}

// processStatement processes a statement and all its clauses
func processStatement(statement cmn.StatementNode, ns *cmn.Namespace, perr *cmn.ParseError) {
	for _, clause := range statement.Clauses() {
		processClause(clause, ns, perr)
	}
}

// processClause processes a single clause and its tokens
func processClause(clause cmn.ClauseNode, ns *cmn.Namespace, perr *cmn.ParseError) {
	tokens := clause.RawTokens()
	processTokens(tokens, ns, perr)
}

// processTokens processes a sequence of tokens for directive validation
func processTokens(tokens []tokenizer.Token, ns *cmn.Namespace, perr *cmn.ParseError) {
	i := 0

	for i < len(tokens) {
		token := tokens[i]
		if token.Directive != nil {
			switch token.Directive.Type {
			case "variable":
				validateVariableDirective(token, ns, perr)
			case "const":
				validateConstDirective(token, ns, perr)
			case "if":
				validateIfDirective(token, ns, perr)
			case "for":
				// Handle for loop - process the loop body with extended namespace
				ok, endIndex := processForLoop(tokens, i, ns, perr)
				if ok {
					// Process tokens inside the loop with the extended namespace
					loopTokens := tokens[i+1 : endIndex]
					processTokens(loopTokens, ns, perr)
					i = endIndex // Skip to after the end directive
					ns.ExitLoop()
					continue
				}
			case "elseif":
				validateElseIfDirective(token, ns, perr)
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
func validateVariableDirective(token tokenizer.Token, ns *cmn.Namespace, perr *cmn.ParseError) {
	expression := extractExpressionFromDirective(token.Value, "/*=", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid variable directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Validate the expression using parameter CEL
	if _, _, err := ns.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", expression, err, token.Position.String()))
	}
}

// validateConstDirective validates a const directive
func validateConstDirective(token tokenizer.Token, ns *cmn.Namespace, perr *cmn.ParseError) {
	expression := extractExpressionFromDirective(token.Value, "/*$", "*/")
	if expression == "" {
		perr.Add(fmt.Errorf("%w at %s: invalid const directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}
	// Validate as environment expression
	if _, _, err := ns.Eval(expression); err != nil {
		perr.Add(fmt.Errorf("undefined variable in environment expression '%s': %w at %s", expression, err, token.Position.String()))
	}
}

// validateIfDirective validates an if directive
func validateIfDirective(token tokenizer.Token, ns *cmn.Namespace, perr *cmn.ParseError) {
	if token.Directive == nil || token.Directive.Type != "if" {
		return
	}

	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: empty condition in if directive", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Validate the condition expression using CEL
	if _, _, err := ns.Eval(condition); err != nil {
		perr.Add(fmt.Errorf("undefined variable in condition '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// validateElseIfDirective validates an elseif directive
func validateElseIfDirective(token tokenizer.Token, ns *cmn.Namespace, perr *cmn.ParseError) {
	if token.Directive == nil || token.Directive.Type != "elseif" {
		return
	}

	condition := token.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: empty condition in elseif directive", cmn.ErrInvalidForSnapSQL, token.Position.String()))
		return
	}

	// Validate the condition expression using CEL
	if _, _, err := ns.Eval(condition); err != nil {
		perr.Add(fmt.Errorf("undefined variable in condition '%s': %w at %s", condition, err, token.Position.String()))
	}
}

// processForLoop processes a for loop and returns the extended namespace and end index
func processForLoop(tokens []tokenizer.Token, startIndex int, ns *cmn.Namespace, perr *cmn.ParseError) (bool, int) {
	if startIndex >= len(tokens) {
		return false, startIndex
	}

	forToken := tokens[startIndex]
	if forToken.Directive == nil || forToken.Directive.Type != "for" {
		return false, startIndex
	}

	// Validate the for condition
	condition := forToken.Directive.Condition
	if condition == "" {
		perr.Add(fmt.Errorf("%w at %s: empty condition in for directive", cmn.ErrInvalidForSnapSQL, forToken.Position.String()))
		return false, startIndex
	}

	// Parse the for condition: "item in items"
	parts := strings.Fields(condition)
	if len(parts) != 3 || parts[1] != ":" {
		perr.Add(fmt.Errorf("%w at %s: invalid for condition format, expected 'item : items'", cmn.ErrInvalidForSnapSQL, forToken.Position.String()))
		return false, startIndex
	}

	loopVar := parts[0]
	listExpr := parts[2]

	// Validate that the list expression exists
	if loopTarget, _, err := ns.Eval(listExpr); err != nil {
		perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", listExpr, err, forToken.Position.String()))
		return false, startIndex
	} else if loopTarget2, ok := loopTarget.([]any); ok {
		endIndex := findMatchingEnd(tokens, startIndex)
		err := ns.EnterLoop(loopVar, loopTarget2)
		if err != nil {
			perr.Add(fmt.Errorf("failed to enter loop: %w at %s", err, forToken.Position.String()))
			return false, startIndex
		}
		return true, endIndex
	} else {
		perr.Add(fmt.Errorf("%w at %s: loop target '%s' is not list: %v", cmn.ErrInvalidForSnapSQL, forToken.Position.String(), listExpr, loopTarget))
		return false, startIndex
	}
}

// findMatchingEnd finds the matching end directive for a given start index
func findMatchingEnd(tokens []tokenizer.Token, startIndex int) int {
	level := 1
	for i := startIndex + 1; i < len(tokens); i++ {
		token := tokens[i]
		if token.Directive != nil {
			switch token.Directive.Type {
			case "if", "for":
				level++
			case "end":
				level--
				if level == 0 {
					return i
				}
			}
		}
	}
	return -1 // No matching end found
}

// extractExpressionFromDirective extracts expression from directive content
func extractExpressionFromDirective(content, prefix, suffix string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, prefix) || !strings.HasSuffix(content, suffix) {
		return ""
	}

	expression := content[len(prefix) : len(content)-len(suffix)]
	return strings.TrimSpace(expression)
}
