package intermediate

import (
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

var simpleVarRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// CELExtractor extracts CEL expressions and environment variables from parsed SQL
type CELExtractor struct {
	expressions []string        // 複雑なCEL式のリスト
	simpleVars  map[string]bool // 単純変数のセット
	envs        [][]EnvVar      // 環境変数のリスト
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// NewCELExtractor creates a new CEL extractor
func NewCELExtractor() *CELExtractor {
	return &CELExtractor{
		expressions: make([]string, 0),
		simpleVars:  make(map[string]bool),
		envs:        make([][]EnvVar, 0),
	}
}

// ExtractFromTokens extracts CEL expressions and environment variables from tokens
func (ce *CELExtractor) ExtractFromTokens(tokens []tokenizer.Token) {
	envLevel := 0
	forLoopStack := make([]string, 0) // Stack of loop variable names
	
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		
		// Check for directives
		if token.Directive != nil {
			switch token.Directive.Type {
			case "if", "elseif":
				// Extract condition expression
				ce.addExpression(token.Directive.Condition)
				
			case "for":
				// Parse "variable : collection" format
				parts := strings.Split(token.Directive.Condition, ":")
				if len(parts) == 2 {
					variable := strings.TrimSpace(parts[0])
					collection := strings.TrimSpace(parts[1])
					
					// Extract collection expression
					ce.addExpression(collection)
					
					// Increase environment level for the loop body
					envLevel++
					
					// Ensure we have enough environment levels
					for len(ce.envs) <= envLevel-1 {
						ce.envs = append(ce.envs, make([]EnvVar, 0))
					}
					
					// Add loop variable to the environment
					ce.envs[envLevel-1] = append(ce.envs[envLevel-1], EnvVar{
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
					variable := forLoopStack[len(forLoopStack)-1]
					forLoopStack = forLoopStack[:len(forLoopStack)-1]
					
					// Extract expressions for the loop variable at this level
					ce.addExpression(variable)
					
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
					ce.addExpression(varExpr)
				}
			}
		}
	}
}

// addExpression adds a CEL expression to the appropriate list
func (ce *CELExtractor) addExpression(expr string) {
	if expr == "" {
		return
	}

	// Check if it's a simple variable (matches regex pattern)
	if isSimpleVariable(expr) {
		ce.simpleVars[expr] = true
		return
	}

	// It's a complex CEL expression, add to expressions list if not already present
	for _, existingExpr := range ce.expressions {
		if existingExpr == expr {
			return
		}
	}
	ce.expressions = append(ce.expressions, expr)
}

// isSimpleVariable checks if an expression is a simple variable reference
func isSimpleVariable(expr string) bool {
	return simpleVarRegex.MatchString(expr)
}

// GetExpressions returns the extracted complex CEL expressions
func (ce *CELExtractor) GetExpressions() []string {
	return ce.expressions
}

// GetSimpleVars returns the extracted simple variables
func (ce *CELExtractor) GetSimpleVars() []string {
	vars := make([]string, 0, len(ce.simpleVars))
	for v := range ce.simpleVars {
		vars = append(vars, v)
	}
	return vars
}

// GetEnvs returns the environment variables by level
func (ce *CELExtractor) GetEnvs() [][]EnvVar {
	return ce.envs
}
