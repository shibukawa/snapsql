package intermediate

import (
	"fmt"
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
	_, envs := extractFromStatement(ctx.Statement)

	// Convert [][]EnvVar to []string for now (simplified)
	var envStrings []string

	for _, envGroup := range envs {
		for _, env := range envGroup {
			envStrings = append(envStrings, env.Name)
		}
	}

	ctx.Environments = envStrings

	// Extract enhanced CEL information with proper environment mapping
	celExpressions, celEnvironments := c.extractEnhancedCELInfo(ctx.Statement)
	ctx.CELExpressions = celExpressions
	ctx.CELEnvironments = celEnvironments

	return nil
}

// extractEnhancedCELInfo extracts CEL expressions with environment mapping
func (c *CELExpressionExtractor) extractEnhancedCELInfo(stmt parser.StatementNode) ([]CELExpression, []CELEnvironment) {
	expressions := make([]CELExpression, 0)
	environments := make([]CELEnvironment, 0)

	expressionCounter := 0
	currentEnvIndex := 0

	// Create base environment (index 0) - uses parameters from interface_schema
	baseEnv := CELEnvironment{
		Index:               0,
		AdditionalVariables: []CELVariableInfo{}, // Empty - uses parameters
	}
	environments = append(environments, baseEnv)

	// Helper function to create CEL expression
	createExpression := func(expr string, line int) CELExpression {
		expressionCounter++

		return CELExpression{
			ID:               fmt.Sprintf("expr_%03d", expressionCounter),
			Expression:       expr,
			EnvironmentIndex: currentEnvIndex,
			Position: Position{
				Line:   line,
				Column: 0,
			},
		}
	}

	// Process all tokens from the statement
	processTokens := func(tokens []tokenizer.Token) {
		for _, token := range tokens {
			if token.Directive != nil {
				switch token.Directive.Type {
				case "if", "elseif":
					if token.Directive.Condition != "" {
						expr := createExpression(token.Directive.Condition, 0)
						expressions = append(expressions, expr)
					}

				case "for":
					// Parse "variable : collection" format
					parts := strings.Split(token.Directive.Condition, ":")
					if len(parts) == 2 {
						variable := strings.TrimSpace(parts[0])
						collection := strings.TrimSpace(parts[1])

						// Add collection expression to current environment
						collectionExpr := createExpression(collection, 0)
						expressions = append(expressions, collectionExpr)

						// Create new environment for loop body
						currentEnvIndex++
						loopEnv := CELEnvironment{
							Index: currentEnvIndex,
							AdditionalVariables: []CELVariableInfo{
								{
									Name: variable,
									Type: "any", // Could be inferred from collection type
								},
							},
						}
						environments = append(environments, loopEnv)
					}

				case "end":
					// Return to parent environment
					if currentEnvIndex > 0 {
						currentEnvIndex--
					}

				case "variable":
					var varExpr string
					if token.Directive.Condition != "" {
						varExpr = token.Directive.Condition
					} else if token.Value != "" && strings.HasPrefix(token.Value, "/*=") && strings.HasSuffix(token.Value, "*/") {
						varExpr = strings.TrimSpace(token.Value[3 : len(token.Value)-2])
					}

					if varExpr != "" {
						expr := createExpression(varExpr, 0)
						expressions = append(expressions, expr)
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

	return expressions, environments
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
					// Use directive field if available (more efficient)
					if token.Directive != nil && token.Directive.Condition != "" {
						addExpression(token.Directive.Condition)
					} else {
						// Fallback to parsing token value (legacy support)
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
	}

	// Process tokens from each clause
	for _, clause := range stmt.Clauses() {
		processTokens(clause.RawTokens())
	}

	// Process CTE tokens if available
	if cte := stmt.CTE(); cte != nil {
		processTokens(cte.RawTokens())
	}

	return expressions, envs
}
