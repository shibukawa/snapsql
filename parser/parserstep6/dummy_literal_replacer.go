package parserstep6

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// replaceDummyLiterals replaces DUMMY_LITERAL tokens with actual literals based on FunctionDefinition
func replaceDummyLiterals(statement cmn.StatementNode, namespace *cmn.Namespace, functionDef cmn.FunctionDefinition, perr *cmn.ParseError) {
	// Step 1: Pre-scan for loop directives to register loop variables
	preRegisterLoopVariables(statement, namespace, perr)

	// Step 2: Process all clauses in the statement
	for _, clause := range statement.Clauses() {
		replaceDummyLiteralsInClause(clause, namespace, functionDef, perr)
	}
}

// replaceDummyLiteralsInClause replaces DUMMY_LITERAL tokens in a single clause
func replaceDummyLiteralsInClause(clause cmn.ClauseNode, namespace *cmn.Namespace, functionDef cmn.FunctionDefinition, perr *cmn.ParseError) {
	// Use reflection to access private fields
	clauseValue := reflect.ValueOf(clause)
	if clauseValue.Kind() == reflect.Ptr {
		clauseValue = clauseValue.Elem()
	}

	// Look for the embedded clauseBaseNode
	clauseBaseField := clauseValue.FieldByName("clauseBaseNode")
	if !clauseBaseField.IsValid() {
		fmt.Printf("[DEBUG] Could not find clauseBaseNode field\n")
		return
	}

	// Access headingTokens and bodyTokens fields
	headingTokensField := clauseBaseField.FieldByName("headingTokens")
	bodyTokensField := clauseBaseField.FieldByName("bodyTokens")

	if !headingTokensField.IsValid() || !bodyTokensField.IsValid() {
		fmt.Printf("[DEBUG] Could not access token fields\n")
		return
	}

	// Make fields accessible
	headingTokensField = reflect.NewAt(headingTokensField.Type(), unsafe.Pointer(headingTokensField.UnsafeAddr())).Elem()
	bodyTokensField = reflect.NewAt(bodyTokensField.Type(), unsafe.Pointer(bodyTokensField.UnsafeAddr())).Elem()

	// Process heading tokens
	headingTokens := headingTokensField.Interface().([]tokenizer.Token)
	for i := range headingTokens {
		if headingTokens[i].Type == tokenizer.DUMMY_LITERAL {
			replaceDummyLiteralToken(&headingTokens[i], namespace, functionDef, perr)
		}
	}

	// Process body tokens
	bodyTokens := bodyTokensField.Interface().([]tokenizer.Token)
	for i := range bodyTokens {
		if bodyTokens[i].Type == tokenizer.DUMMY_LITERAL {
			replaceDummyLiteralToken(&bodyTokens[i], namespace, functionDef, perr)
		}
	}

	// Update the fields
	headingTokensField.Set(reflect.ValueOf(headingTokens))
	bodyTokensField.Set(reflect.ValueOf(bodyTokens))
}

// replaceDummyLiteralToken replaces a single DUMMY_LITERAL token with actual literal
func replaceDummyLiteralToken(token *tokenizer.Token, namespace *cmn.Namespace, functionDef cmn.FunctionDefinition, perr *cmn.ParseError) {
	variableName := token.Value

	// Get parameter type from FunctionDefinition or Namespace loop variables
	paramType, err := getParameterTypeWithNamespace(variableName, functionDef.Parameters, namespace)
	if err != nil {
		// If parameter not found, log error but use default string type
		perr.Add(fmt.Errorf("parameter '%s' not found in FunctionDefinition or loop variables at %s: %w",
			variableName, token.Position.String(), err))
		paramType = "string"
	}

	// Generate actual literal value based on type
	literalValue := generateLiteralFromType(paramType)
	tokenType := inferTokenTypeFromLiteral(literalValue)

	// Replace the token
	token.Type = tokenType
	token.Value = literalValue
}

// getParameterType retrieves parameter type from FunctionDefinition parameters
// Supports dot notation for nested parameters (e.g., "user.name")
func getParameterType(variableName string, parameters map[string]any) (string, error) {
	parts := strings.Split(variableName, ".")
	current := parameters

	// Navigate through nested structure
	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return "", fmt.Errorf("parameter '%s' not found", variableName)
		}

		if i == len(parts)-1 {
			// Last part - extract type
			if paramMap, ok := value.(map[string]any); ok {
				if typeStr, ok := paramMap["type"].(string); ok {
					return typeStr, nil
				}
			}
			// If no type specified, try to infer from value
			return inferTypeFromValue(value), nil
		} else {
			// Intermediate part - navigate deeper
			if paramMap, ok := value.(map[string]any); ok {
				current = paramMap
			} else {
				return "", fmt.Errorf("parameter '%s' is not a nested object", strings.Join(parts[:i+1], "."))
			}
		}
	}

	return "", fmt.Errorf("parameter '%s' not found", variableName)
}

// getParameterTypeWithNamespace retrieves parameter type from FunctionDefinition parameters or Namespace loop variables
// First checks FunctionDefinition, then checks current loop variables in Namespace
func getParameterTypeWithNamespace(variableName string, parameters map[string]any, namespace *cmn.Namespace) (string, error) {
	// First try to get from FunctionDefinition parameters
	paramType, err := getParameterType(variableName, parameters)
	if err == nil {
		return paramType, nil
	}

	// If not found in parameters, check loop variables in Namespace
	if namespace != nil {
		if loopVarType, found := namespace.GetLoopVariableType(variableName); found {
			return loopVarType, nil
		}
	}

	// Not found in either parameters or loop variables
	return "", fmt.Errorf("parameter '%s' not found in FunctionDefinition or loop variables", variableName)
}

// inferTypeFromValue infers type from parameter value
func inferTypeFromValue(value any) string {
	switch value.(type) {
	case int, int64, int32:
		return "int"
	case float64, float32:
		return "float"
	case bool:
		return "bool"
	case string:
		return "string"
	default:
		return "string"
	}
}

// generateLiteralFromType generates appropriate literal value based on parameter type
func generateLiteralFromType(paramType string) string {
	switch strings.ToLower(paramType) {
	case "int", "integer", "long":
		return "1"
	case "float", "double", "decimal", "number":
		return "1.0"
	case "bool", "boolean":
		return "true"
	case "string", "text":
		return "'dummy'"
	case "date":
		return "'2024-01-01'"
	case "datetime", "timestamp":
		return "'2024-01-01 00:00:00'"
	case "email":
		return "'user@example.com'"
	case "uuid":
		return "'00000000-0000-0000-0000-000000000000'"
	case "json":
		return "'{}'"
	case "array":
		return "'[]'"
	default:
		return "'dummy'" // Default to string
	}
}

// inferTokenTypeFromLiteral infers token type from literal value
func inferTokenTypeFromLiteral(literalValue string) tokenizer.TokenType {
	switch {
	case strings.HasPrefix(literalValue, "'"):
		return tokenizer.STRING
	case literalValue == "true" || literalValue == "false":
		return tokenizer.BOOLEAN
	case strings.Contains(literalValue, "."):
		return tokenizer.NUMBER // Floating-point number
	default:
		return tokenizer.NUMBER // Integer
	}
}

// preRegisterLoopVariables scans for loop directives and pre-registers loop variables in the namespace
func preRegisterLoopVariables(statement cmn.StatementNode, namespace *cmn.Namespace, perr *cmn.ParseError) {
	for _, clause := range statement.Clauses() {
		preRegisterLoopVariablesInClause(clause, namespace, perr)
	}
}

// preRegisterLoopVariablesInClause scans a single clause for loop directives
func preRegisterLoopVariablesInClause(clause cmn.ClauseNode, namespace *cmn.Namespace, perr *cmn.ParseError) {
	// Use reflection to access private fields
	clauseValue := reflect.ValueOf(clause)
	if clauseValue.Kind() == reflect.Ptr {
		clauseValue = clauseValue.Elem()
	}

	clauseBaseField := clauseValue.FieldByName("clauseBaseNode")
	if !clauseBaseField.IsValid() {
		return
	}

	headingTokensField := clauseBaseField.FieldByName("headingTokens")
	bodyTokensField := clauseBaseField.FieldByName("bodyTokens")

	if !headingTokensField.IsValid() || !bodyTokensField.IsValid() {
		return
	}

	// Make fields accessible
	headingTokensField = reflect.NewAt(headingTokensField.Type(), unsafe.Pointer(headingTokensField.UnsafeAddr())).Elem()
	bodyTokensField = reflect.NewAt(bodyTokensField.Type(), unsafe.Pointer(bodyTokensField.UnsafeAddr())).Elem()

	// Process both heading and body tokens
	headingTokens := headingTokensField.Interface().([]tokenizer.Token)
	bodyTokens := bodyTokensField.Interface().([]tokenizer.Token)

	preRegisterLoopVariablesInTokens(headingTokens, namespace, perr)
	preRegisterLoopVariablesInTokens(bodyTokens, namespace, perr)
}

// preRegisterLoopVariablesInTokens scans tokens for loop directives and registers loop variables
func preRegisterLoopVariablesInTokens(tokens []tokenizer.Token, namespace *cmn.Namespace, perr *cmn.ParseError) {
	for _, token := range tokens {
		if token.Directive != nil && token.Directive.Type == "for" {
			// Parse the for condition to extract loop variable and target
			condition := token.Directive.Condition
			if condition == "" {
				continue
			}

			// Parse the for condition: "item : items"
			parts := strings.Fields(condition)
			if len(parts) != 3 || parts[1] != ":" {
				continue
			}

			loopVar := parts[0]
			listExpr := parts[2]

			// Try to evaluate the list expression to get the loop target
			if loopTarget, err := namespace.EvaluateParameterExpression(listExpr); err == nil {
				if loopTarget2, ok := loopTarget.([]any); ok {
					// Register the loop variable in the namespace
					namespace.EnterLoop(loopVar, loopTarget2)

					// Note: We don't need to call LeaveLoop here since this is just
					// for DUMMY_LITERAL resolution. The actual loop processing in
					// validateVariables will handle proper loop lifecycle.
				}
			}
		}
	}
}
