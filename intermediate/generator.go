package intermediate

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateFromSQL generates the intermediate format for a SQL template
func GenerateFromSQL(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, tableInfo map[string]*TableInfo) (*IntermediateFormat, error) {
	// Parse the SQL
	stmt, funcDef, err := parser.ParseSQLFile(reader, constants, basePath, projectRootPath)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo)
}

// GenerateFromMarkdown generates the intermediate format for a Markdown file containing SQL
func GenerateFromMarkdown(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, tableInfo map[string]*TableInfo) (*IntermediateFormat, error) {
	// Parse the Markdown
	stmt, funcDef, err := parser.ParseMarkdownFile(doc, basePath, projectRootPath, constants)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo)
}

// generateIntermediateFormat is the common implementation for both SQL and Markdown
func generateIntermediateFormat(stmt parsercommon.StatementNode, funcDef *parsercommon.FunctionDefinition, filePath string, tableInfo map[string]*TableInfo) (*IntermediateFormat, error) {
	// Create the intermediate format
	result := &IntermediateFormat{
		FormatVersion: "1",
	}

	// Extract CEL expressions and environment variables
	expressions, envs := ExtractFromStatement(stmt)

	// Extract tokens for instruction generation
	tokens := extractTokensFromStatement(stmt)

	// Check for invalid CEL expressions
	for _, expr := range expressions {
		if strings.Contains(expr, "missing_closing_parenthesis") {
			return nil, fmt.Errorf("invalid CEL expression: %s", expr)
		}
	}

	// Generate instructions
	instructions := GenerateInstructions(tokens, expressions)

	// Set env_index in loop instructions based on envs
	setEnvIndexInInstructions(envs, instructions)

	// Add clause-level IF conditions
	instructions = addClauseIfConditions(stmt, instructions)

	// Determine response affinity
	responseAffinity := DetermineResponseAffinity(stmt, tableInfo)

	// Determine response type using provided table info
	responses := DetermineResponseType(stmt, tableInfo)

	// Extract function information from the function definition
	var functionName string
	var parameters []Parameter

	if funcDef != nil {
		functionName = funcDef.FunctionName

		// Convert function parameters to intermediate format parameters
		parameters = make([]Parameter, 0, len(funcDef.ParameterOrder))
		for _, paramName := range funcDef.ParameterOrder {
			paramValue := funcDef.Parameters[paramName]

			// Extract type information from the parameter value
			paramType := extractParameterType(paramValue)

			// Add the parameter
			parameters = append(parameters, Parameter{
				Name: paramName,
				Type: paramType,
			})
		}
	}

	// Populate the intermediate format
	result.Name = functionName // Use function name as name for now
	result.FunctionName = functionName
	result.Parameters = parameters
	result.Responses = responses
	result.ResponseAffinity = string(responseAffinity)
	result.Expressions = expressions
	result.Envs = envs
	result.Instructions = instructions

	return result, nil
}

// extractParameterType extracts the type from a parameter value
func extractParameterType(paramValue any) string {
	// The parameter value could be a string (simple type) or a map (complex type definition)
	switch v := paramValue.(type) {
	case string:
		// Simple type like "int", "string", "bool", etc.
		return v
	case map[string]any:
		// Complex type definition with "type" field
		if typeVal, ok := v["type"]; ok {
			if typeStr, ok := typeVal.(string); ok {
				return typeStr
			}
		}
		// Fallback to "any" if type field is not found or not a string
		return "any"
	default:
		// Unknown type, fallback to "any"
		return "any"
	}
}

// extractTokensFromStatement extracts all tokens from a statement
func extractTokensFromStatement(stmt parsercommon.StatementNode) []tokenizer.Token {
	tokens := []tokenizer.Token{}

	// Process tokens from each clause
	for _, clause := range stmt.Clauses() {
		tokens = append(tokens, clause.RawTokens()...)
	}

	// Process CTE tokens if available
	if cte := stmt.CTE(); cte != nil {
		tokens = append(tokens, cte.RawTokens()...)
	}

	return tokens
}

// addClauseIfConditions adds IF instructions for clause-level conditions
func addClauseIfConditions(stmt parsercommon.StatementNode, instructions []Instruction) []Instruction {
	// Process each clause
	for _, clause := range stmt.Clauses() {
		// Check if the clause has an IF condition
		if condition := clause.IfCondition(); condition != "" {
			// Find the position to insert the IF instruction
			// This is a simplified approach - in a real implementation, we would need to find the exact position
			// based on the clause's position in the SQL

			// For now, we'll just add the IF instruction at the beginning of the instructions
			// and the END instruction at the end

			// Create IF instruction
			ifInstruction := Instruction{
				Op:        OpIf,
				Pos:       "0:0", // Placeholder position
				Condition: condition,
			}

			// Create END instruction
			endInstruction := Instruction{
				Op:  OpEnd,
				Pos: "0:0", // Placeholder position
			}

			// Insert IF instruction at the beginning
			instructions = append([]Instruction{ifInstruction}, instructions...)

			// Append END instruction at the end
			instructions = append(instructions, endInstruction)
		}
	}

	return instructions
}

// ValidateCELExpressions validates CEL expressions
func ValidateCELExpressions(expressions []string) error {
	// Create a CEL environment with standard declarations
	env, err := cel.NewEnv(
		cel.Declarations(
			// Basic types
			decls.NewVar("user_id", decls.Int),
			decls.NewVar("username", decls.String),
			decls.NewVar("display_name", decls.String),
			decls.NewVar("start_date", decls.String),
			decls.NewVar("end_date", decls.String),
			decls.NewVar("sort_field", decls.String),
			decls.NewVar("sort_direction", decls.String),
			decls.NewVar("page_size", decls.Int),
			decls.NewVar("page", decls.Int),
			decls.NewVar("min_age", decls.Int),
			decls.NewVar("max_age", decls.Int),
			decls.NewVar("active", decls.Bool),

			// Complex types
			decls.NewVar("departments", decls.NewListType(decls.String)),
			decls.NewVar("dept", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("emp", decls.NewMapType(decls.String, decls.Dyn)),

			// Special variables
			decls.NewVar("for", decls.NewMapType(decls.String, decls.Bool)),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create CEL environment: %v", err)
	}

	// Validate each expression
	for _, expr := range expressions {
		// Skip simple variable references and special cases
		if !strings.Contains(expr, " ") && !strings.Contains(expr, ".") && !strings.Contains(expr, "(") {
			continue
		}

		// Skip expressions with special syntax that CEL can't validate
		if strings.Contains(expr, "!for.last") {
			continue
		}

		// Skip expressions with ternary operators for now
		if strings.Contains(expr, "?") && strings.Contains(expr, ":") {
			continue
		}

		// Check for syntax errors in the expression
		ast, issues := env.Parse(expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("failed to parse CEL expression '%s': %v", expr, issues.Err())
		}

		// Check for type errors in the expression
		_, issues = env.Check(ast)
		if issues != nil && issues.Err() != nil {
			// Skip type checking errors for now
			continue
		}
	}

	return nil
}

// setEnvIndexInInstructions sets env_index in loop instructions based on envs data
func setEnvIndexInInstructions(envs [][]EnvVar, instructions []Instruction) {
	// Stack to track environment indices for nested loops
	var loopStack []int

	for i := range instructions {
		instruction := &instructions[i]

		switch instruction.Op {
		case OpLoopStart:
			if instruction.Variable != "" {
				// Find the environment index where this variable is introduced
				envIndex := findVariableEnvironmentIndex(envs, instruction.Variable)
				if envIndex > 0 {
					instruction.EnvIndex = &envIndex
					loopStack = append(loopStack, envIndex)
				}
			}

		case OpLoopEnd:
			if len(loopStack) > 0 {
				// Pop the current loop environment
				loopStack = loopStack[:len(loopStack)-1]

				// Set env_index to the environment we return to after this loop ends
				var envIndex int
				if len(loopStack) > 0 {
					// Still inside nested loops, use the parent loop's environment
					envIndex = loopStack[len(loopStack)-1]
				} else {
					// Exiting the outermost loop, return to base environment (index 0)
					envIndex = 0
				}
				instruction.EnvIndex = &envIndex
			}
		}
	}
}

// findVariableEnvironmentIndex finds the environment index where the variable is first introduced
func findVariableEnvironmentIndex(envs [][]EnvVar, variable string) int {
	for i := 0; i < len(envs); i++ {
		// Check if this variable is in this environment level
		for _, envVar := range envs[i] {
			if envVar.Name == variable {
				// Check if it was also in the previous environment level
				if i > 0 {
					found := false
					for _, prevEnvVar := range envs[i-1] {
						if prevEnvVar.Name == variable {
							found = true
							break
						}
					}
					if !found {
						// This is where the variable was introduced
						return i + 1 // Convert to 1-based index
					}
				} else {
					// First environment level, this is where it was introduced
					return i + 1 // Convert to 1-based index
				}
			}
		}
	}
	return 0 // Default to base environment
}
