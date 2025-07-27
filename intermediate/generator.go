package intermediate

import (
	"fmt"
	"io"
	"strings"

	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateFromSQL generates the intermediate format for a SQL template
func GenerateFromSQL(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
	// Parse the SQL
	stmt, funcDef, err := parser.ParseSQLFile(reader, constants, basePath, projectRootPath)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo, config)
}

// GenerateFromMarkdown generates the intermediate format for a Markdown file containing SQL
func GenerateFromMarkdown(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
	// Parse the Markdown
	stmt, funcDef, err := parser.ParseMarkdownFile(doc, basePath, projectRootPath, constants)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo, config)
}

// generateIntermediateFormat is the common implementation for both SQL and Markdown
func generateIntermediateFormat(stmt parsercommon.StatementNode, funcDef *parsercommon.FunctionDefinition, filePath string, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
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
			// Use OriginalParameters for parameter type names (preserves common type names)
			originalParamValue := funcDef.OriginalParameters[paramName]

			var paramType string
			if originalParamValue != nil {
				// For common types, use the original type name (e.g., "User", "Department[]")
				paramType = extractParameterTypeFromOriginal(originalParamValue)
			} else {
				// Fallback to normalized type if original is not available
				paramValue := funcDef.Parameters[paramName]
				paramType = extractParameterType(paramValue)
			}

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

	// Add system fields information from config
	if config != nil {
		result.SystemFields = extractSystemFieldsInfo(config, stmt)

		// Perform system field validation and get implicit parameters
		implicitParams, err := CheckSystemFields(stmt, config, parameters)
		if err != nil {
			return nil, err
		}
		result.ImplicitParameters = implicitParams

		// For UPDATE statements, add system fields to SET clause
		if len(implicitParams) > 0 {
			err = AddSystemFieldsToUpdate(stmt, implicitParams)
			if err != nil {
				return nil, fmt.Errorf("failed to add system fields to UPDATE statement: %w", err)
			}
		}
	}

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

// extractSystemFieldsInfo extracts system fields information from config
func extractSystemFieldsInfo(config *Config, stmt parsercommon.StatementNode) []SystemFieldInfo {
	var systemFields []SystemFieldInfo

	// Get all system fields from config
	for _, field := range config.System.Fields {
		systemFieldInfo := SystemFieldInfo{
			Name:              field.Name,
			ExcludeFromSelect: field.ExcludeFromSelect,
		}

		// Convert OnInsert configuration
		if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
			systemFieldInfo.OnInsert = &SystemFieldOperationInfo{
				Default:   field.OnInsert.Default,
				Parameter: string(field.OnInsert.Parameter),
			}
		}

		// Convert OnUpdate configuration
		if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
			systemFieldInfo.OnUpdate = &SystemFieldOperationInfo{
				Default:   field.OnUpdate.Default,
				Parameter: string(field.OnUpdate.Parameter),
			}
		}

		systemFields = append(systemFields, systemFieldInfo)
	}

	return systemFields
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

// extractParameterTypeFromOriginal extracts type string from original parameter value (for common types)
func extractParameterTypeFromOriginal(value any) string {
	switch v := value.(type) {
	case string:
		return v // Common type names like "User", "Department[]", "api/users/User"
	default:
		// Fallback to regular extraction
		return extractParameterType(value)
	}
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
