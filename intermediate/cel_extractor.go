package intermediate

import (
	"regexp"
	"slices"
	"strings"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

var simpleVarRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// EnvVar represents an environment variable
type EnvVar struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ExtractFromStatement extracts CEL expressions and environment variables from a parsed statement
func ExtractFromStatement(stmt parser.StatementNode) (expressions []string, envs [][]EnvVar) {
	expressions = make([]string, 0)
	envs = make([][]EnvVar, 0)
	envLevel := 0
	forLoopStack := make([]string, 0) // Stack of loop variable names

	// Process all tokens from the statement
	processTokens := func(tokens []tokenizer.Token) {
		for _, token := range tokens {
			// Check for directives
			if token.Directive != nil {
				switch token.Directive.Type {
				case "if", "elseif":
					if token.Directive.Condition != "" && !slices.Contains(expressions, token.Directive.Condition) {
						expressions = append(expressions, token.Directive.Condition)
					}

				case "for":
					// Parse "variable : collection" format
					parts := strings.Split(token.Directive.Condition, ":")
					if len(parts) == 2 {
						variable := strings.TrimSpace(parts[0])
						collection := strings.TrimSpace(parts[1])

						// Extract collection expression
						if collection != "" && !slices.Contains(expressions, collection) {
							expressions = append(expressions, collection)
						}

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
						// Extract variable name between /*= and */
						varExpr := strings.TrimSpace(token.Value[3 : len(token.Value)-2])
						if varExpr != "" && !slices.Contains(expressions, varExpr) {
							expressions = append(expressions, varExpr)
						}
					}
				}
			}
		}
	}

	// Process tokens from each clause
	for _, clause := range stmt.Clauses() {
		processTokens(clause.RawTokens())
	}

	// Process CTE tokens if available
	if cte := stmt.CTE(); cte != nil {
		processTokens(cte.RawTokens())
	}

	return
}
