package intermediate

import (
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// extractCELFromTokens extracts CEL expressions directly from tokens
func extractCELFromTokens(tokens []tokenizer.Token) ([]string, [][]EnvVar) {
	expressions := make([]string, 0)
	envs := make([][]EnvVar, 0)
	envLevel := 0
	forLoopStack := make([]string, 0) // Stack of loop variable names

	// Helper function to add an expression if it's not already in the list
	addExpression := func(expr string) {
		if expr != "" && !contains(expressions, expr) {
			expressions = append(expressions, expr)
		}
	}

	for _, token := range tokens {
		// Check for directives
		if token.Directive != nil {
			switch token.Directive.Type {
			case "if", "elseif":
				if token.Directive.Condition != "" {
					// Add the full condition expression
					addExpression(token.Directive.Condition)
				}

			case "for":
				// Parse "variable : collection" format
				parts := strings.Split(token.Directive.Condition, ":")
				if len(parts) == 2 {
					variable := strings.TrimSpace(parts[0])
					collection := strings.TrimSpace(parts[1])

					// Extract collection expression
					addExpression(collection)

					// Also add the loop variable as an expression
					addExpression(variable)

					// Increase environment level for the loop body
					envLevel++

					// Ensure we have enough environment levels
					for len(envs) <= envLevel-1 {
						envs = append(envs, make([]EnvVar, 0))
					}

					// Add loop variable to the environment
					envs[envLevel-1] = append(envs[envLevel-1], EnvVar{
						Name: variable,
						Type: "any", // Default type, can be refined later
					})

					// Push loop variable to stack
					forLoopStack = append(forLoopStack, variable)
				}

			case "end":
				// Check if we're ending a for loop
				if len(forLoopStack) > 0 {
					// Pop the last loop variable
					forLoopStack = forLoopStack[:len(forLoopStack)-1]
					// Decrease environment level
					if envLevel > 0 {
						envLevel--
					}
				}

			case "variable":
				// Extract variable expression from the token value
				// The format is /*= variable_name */
				if token.Value != "" && strings.HasPrefix(token.Value, "/*=") && strings.HasSuffix(token.Value, "*/") {
					// Extract variable expression between /*= and */
					varExpr := strings.TrimSpace(token.Value[3 : len(token.Value)-2])

					// Add the full expression, not just simple variables
					addExpression(varExpr)
				}
			}
		}
	}

	return expressions, envs
}

// extractDirectives extracts directives from tokens
func extractDirectives(tokens []tokenizer.Token) []map[string]interface{} {
	directives := []map[string]interface{}{}

	for _, token := range tokens {
		if token.Directive != nil {
			directive := map[string]interface{}{
				"type": token.Directive.Type,
				"position": map[string]interface{}{
					"line":   token.Position.Line,
					"column": token.Position.Column,
					"offset": token.Position.Offset,
				},
			}

			if token.Directive.Condition != "" {
				directive["condition"] = token.Directive.Condition
			}

			directives = append(directives, directive)
		}
	}

	return directives
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
