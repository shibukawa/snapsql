package intermediate

import (
	"fmt"
	"os"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateFromStatementNode generates intermediate format from a statement node
func GenerateFromStatementNode(
	stmt parser.StatementNode,
	functionDef *parser.FunctionDefinition,
	tokens []tokenizer.Token,
	tableInfo map[string]map[string]string,
) (*IntermediateFormat, error) {
	if stmt == nil {
		return nil, fmt.Errorf("statement node is required")
	}
	if functionDef == nil {
		return nil, fmt.Errorf("function definition is required")
	}

	// Create intermediate format
	format := &IntermediateFormat{
		InterfaceSchema: convertFunctionDefToInterfaceSchema(functionDef),
	}

	// Extract CEL expressions and environment variables using the new function
	expressions, envs := ExtractFromStatement(stmt)
	
	// TODO: Separate simple vars from complex expressions
	// For now, treat all as complex expressions
	format.CELExpressions = expressions
	format.SimpleVars = []string{} // This needs to be implemented
	format.Envs = envs
	
	// Generate instructions
	generateInstructions(format, stmt)
	
	// Determine response type and affinity (independently)
	format.ResponseType = determineResponseType(stmt, tableInfo)
	format.ResponseAffinity = string(determineResponseAffinity(stmt))

	return format, nil
}

// generateInstructions generates instructions based on the statement
func generateInstructions(format *IntermediateFormat, stmt parser.StatementNode) {
	// This is a placeholder implementation
	// In a real implementation, we would traverse the AST and generate instructions
	format.Instructions = []Instruction{
		{
			Op:    OpEmitLiteral,
			Value: "SELECT id, name, email FROM table",
			Pos:   []int{1, 1, 0},
		},
	}
}

// LoadFromFile loads an intermediate format from a JSON file
func LoadFromFile(filePath string) (*IntermediateFormat, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intermediate file %s: %w", filePath, err)
	}

	// Parse JSON
	format, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intermediate format: %w", err)
	}

	return format, nil
}

// convertFunctionDefToInterfaceSchema converts a function definition to an interface schema
func convertFunctionDefToInterfaceSchema(functionDef *parser.FunctionDefinition) *InterfaceSchema {
	if functionDef == nil {
		return nil
	}
	
	// Create interface schema
	schema := &InterfaceSchema{
		Name:         functionDef.FunctionName, // Use FunctionName for both fields
		FunctionName: functionDef.FunctionName,
	}
	
	// Convert parameters
	parameters := make([]Parameter, 0, len(functionDef.Parameters))
	for name, paramType := range functionDef.Parameters {
		parameters = append(parameters, Parameter{
			Name: name,
			Type: fmt.Sprintf("%v", paramType), // Convert type to string
		})
	}
	schema.Parameters = parameters
	
	return schema
}

// determineResponseType determines the response type for a statement
func determineResponseType(stmt parser.StatementNode, tableInfo map[string]map[string]string) *ResponseType {
	// This is a placeholder implementation
	// In a real implementation, we would analyze the statement and determine the response type
	return &ResponseType{
		Name: "User",
		Fields: []Field{
			{Name: "id", Type: "int", DatabaseTag: "id"},
			{Name: "name", Type: "string", DatabaseTag: "name"},
			{Name: "email", Type: "string", DatabaseTag: "email"},
		},
	}
}

// determineResponseAffinity determines the response affinity for a statement
func determineResponseAffinity(stmt parser.StatementNode) string {
	// This is a placeholder implementation
	// In a real implementation, we would analyze the statement and determine the response affinity
	return "many"
}
