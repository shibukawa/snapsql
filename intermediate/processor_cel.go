package intermediate

import (
	"slices"
	"strings"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// CELExpressionExtractor extracts CEL expressions and environment variables
type CELExpressionExtractor struct{}

func (c *CELExpressionExtractor) Name() string {
	return "CELExpressionExtractor"
}

func (c *CELExpressionExtractor) Process(ctx *ProcessingContext) error {
	// Extract CEL expressions and environment variables from the statement
	expressions, envs := extractFromStatement(ctx.Statement)

	ctx.Expressions = expressions

	// Convert [][]EnvVar to []string for now (simplified)
	var envStrings []string
	for _, envGroup := range envs {
		for _, env := range envGroup {
			envStrings = append(envStrings, env.Name)
		}
	}
	ctx.Environments = envStrings

	return nil
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// extractFromStatement extracts CEL expressions and environment variables from a parsed statement
func extractFromStatement(stmt parser.StatementNode) (expressions []string, envs [][]EnvVar) {
	expressions = make([]string, 0)
	envs = make([][]EnvVar, 0)
	envLevel := 0
	forLoopStack := make([]string, 0) // Stack of loop variable names

	// Helper function to add an expression if it's not already in the list
	addExpression := func(expr string) {
		if expr != "" && !slices.Contains(expressions, expr) {
			expressions = append(expressions, expr)
		}
	}

	// Process all tokens from the statement
	processTokens := func(tokens []tokenizer.Token) {
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

						// Add all variables from the current loop stack to this environment level
						// This includes variables from outer loops plus the current loop variable
						currentEnv := make([]EnvVar, len(forLoopStack)+1)

						// Copy all existing loop variables from the stack
						for i, stackVar := range forLoopStack {
							currentEnv[i] = EnvVar{
								Name: stackVar,
								Type: "any",
							}
						}

						// Add the current loop variable
						currentEnv[len(forLoopStack)] = EnvVar{
							Name: variable,
							Type: "any", // Default type, can be refined later
						}

						// Set the environment for this level
						envs[envLevel-1] = currentEnv

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
