package parserstep5

import (
	"fmt"
	"reflect"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// expandArraysInValues expands array variables in VALUES clauses
// Converts: VALUES (/*= values */) -> VALUES (/*# for v : values *//*= v*/,/*# end */)
func expandArraysInValues(stmt cmn.StatementNode, funcDef *cmn.FunctionDefinition) {
	// Only process INSERT statements
	if stmt.Type() != cmn.INSERT_INTO_STATEMENT {
		return
	}

	insertStmt, ok := stmt.(*cmn.InsertIntoStatement)
	if !ok {
		return
	}

	// Skip if no function definition available
	if funcDef == nil {
		return
	}

	// Create namespace for CEL evaluation
	namespace, err := cmn.NewNamespaceFromDefinition(funcDef)
	if err != nil {
		fmt.Printf("DEBUG: Failed to create namespace: %v\n", err)
		return
	}

	// Process all tokens in the INSERT statement (not just VALUES clause)
	// This allows us to handle FOR directives that span across clauses
	allTokens := []tokenizer.Token{}
	for _, clause := range insertStmt.Clauses() {
		allTokens = append(allTokens, clause.RawTokens()...)
	}
	
	expandArraysInTokensWithNamespace(allTokens, namespace)
}

// expandArraysInTokensWithNamespace processes tokens using Namespace for dynamic type checking
func expandArraysInTokensWithNamespace(tokens []tokenizer.Token, namespace *cmn.Namespace) {
	fmt.Printf("DEBUG: expandArraysInTokensWithNamespace called with %d tokens\n", len(tokens))
	
	// First pass: print all tokens for debugging
	fmt.Printf("DEBUG: Token sequence:\n")
	for i, token := range tokens {
		fmt.Printf("  [%d] %s: %s\n", i, token.Type, token.Value)
	}
	
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		fmt.Printf("DEBUG: Processing token [%d]: %s\n", i, token.Value)
		
		// Handle FOR directive - enter loop scope
		if isForDirective(token) {
			loopVar, arrayVar := parseForDirective(token)
			if loopVar != "" && arrayVar != "" {
				fmt.Printf("DEBUG: Entering loop scope: %s : %s\n", loopVar, arrayVar)
				
				// Evaluate the array variable to get its actual value
				arrayValue, _, err := namespace.Eval(arrayVar)
				if err != nil {
					fmt.Printf("DEBUG: Failed to evaluate array variable %s: %v\n", arrayVar, err)
					continue
				}
				
				// Convert to []any using reflection
				arraySlice, err := convertToAnySlice(arrayValue)
				if err != nil {
					fmt.Printf("DEBUG: Failed to convert %s to []any: %v\n", arrayVar, err)
					continue
				}
				
				// Enter loop with the actual array values
				err = namespace.EnterLoop(loopVar, arraySlice)
				if err != nil {
					fmt.Printf("DEBUG: Failed to enter loop: %v\n", err)
				}
			}
			continue
		}
		
		// Handle END directive - exit loop scope
		if isEndDirective(token) {
			fmt.Printf("DEBUG: Exiting loop scope\n")
			namespace.ExitLoop()
			continue
		}
		
		// Look for variable directive pattern: /*= variable_name */
		if isVariableDirective(token) {
			variableName := extractVariableName(token)
			fmt.Printf("DEBUG: Found variable directive: %s\n", variableName)
			
			// Check if this is inside parentheses (VALUES clause pattern)
			if isInsideParentheses(tokens, i) {
				fmt.Printf("DEBUG: Variable %s is inside parentheses\n", variableName)
				
				// Use Namespace to check if variable is an array type
				if isArrayTypeWithNamespace(variableName, namespace) {
					fmt.Printf("DEBUG: Variable %s is array type - should expand\n", variableName)
					// Mark this token for expansion
					tokens[i].Value = fmt.Sprintf("/*# EXPAND_ARRAY: %s */", variableName)
				} else {
					fmt.Printf("DEBUG: Variable %s is not array type\n", variableName)
				}
			} else {
				fmt.Printf("DEBUG: Variable %s is not inside parentheses\n", variableName)
			}
		}
	}
}

// isForDirective checks if token is a FOR directive
func isForDirective(token tokenizer.Token) bool {
	if token.Type != tokenizer.BLOCK_COMMENT {
		return false
	}
	content := strings.TrimSpace(token.Value)
	return strings.HasPrefix(content, "/*#") && 
		   strings.Contains(content, "for ") &&
		   strings.Contains(content, " : ") &&
		   strings.HasSuffix(content, "*/")
}

// isEndDirective checks if token is an END directive
func isEndDirective(token tokenizer.Token) bool {
	if token.Type != tokenizer.BLOCK_COMMENT {
		return false
	}
	content := strings.TrimSpace(token.Value)
	return content == "/*# end */"
}

// isVariableDirective checks if token is a variable directive
func isVariableDirective(token tokenizer.Token) bool {
	return token.Type == tokenizer.BLOCK_COMMENT && 
		   strings.HasPrefix(token.Value, "/*=") && 
		   strings.HasSuffix(token.Value, "*/")
}

// extractVariableName extracts variable name from /*= variable */ directive
func extractVariableName(token tokenizer.Token) string {
	content := strings.TrimSpace(token.Value[3 : len(token.Value)-2])
	return strings.TrimSpace(content)
}

// parseForDirective parses FOR directive and returns loop variable and array variable
// Example: "/*# for item : items */" -> ("item", "items")
func parseForDirective(token tokenizer.Token) (string, string) {
	content := strings.TrimSpace(token.Value)
	if !strings.HasPrefix(content, "/*#") || !strings.HasSuffix(content, "*/") {
		return "", ""
	}
	
	// Remove /*# and */ markers
	content = strings.TrimSpace(content[3 : len(content)-2])
	
	// Parse "for loopVar : arrayVar"
	if !strings.HasPrefix(content, "for ") {
		return "", ""
	}
	
	forContent := strings.TrimSpace(content[4:]) // Remove "for "
	parts := strings.Split(forContent, " : ")
	if len(parts) != 2 {
		return "", ""
	}
	
	loopVar := strings.TrimSpace(parts[0])
	arrayVar := strings.TrimSpace(parts[1])
	
	return loopVar, arrayVar
}

// convertToAnySlice converts any slice type to []any using reflection
func convertToAnySlice(value any) ([]any, error) {
	if value == nil {
		return nil, fmt.Errorf("value is nil")
	}
	
	rv := reflect.ValueOf(value)
	
	// Check if it's a slice or array
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("value is not a slice or array, got %s", rv.Kind())
	}
	
	// Convert to []any
	length := rv.Len()
	result := make([]any, length)
	for i := 0; i < length; i++ {
		result[i] = rv.Index(i).Interface()
	}
	
	return result, nil
}

// isArrayTypeWithNamespace checks if a variable is an array type using Namespace CEL evaluation
func isArrayTypeWithNamespace(variableName string, namespace *cmn.Namespace) bool {
	// Try to evaluate the variable in the current namespace context
	result, typeName, err := namespace.Eval(variableName)
	if err != nil {
		fmt.Printf("DEBUG: Failed to evaluate %s: %v\n", variableName, err)
		return false
	}
	
	fmt.Printf("DEBUG: Variable %s has CEL type: %s, Go value type: %T\n", variableName, typeName, result)
	
	// Use reflection to check if it's a slice or array
	if result != nil {
		rv := reflect.ValueOf(result)
		isArray := rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
		if isArray {
			fmt.Printf("DEBUG: Variable %s is a Go slice/array (detected via reflection)\n", variableName)
		} else {
			fmt.Printf("DEBUG: Variable %s is not a Go slice/array (kind: %s)\n", variableName, rv.Kind())
		}
		return isArray
	}
	
	fmt.Printf("DEBUG: Variable %s has nil value\n", variableName)
	return false
}

// isInsideParentheses checks if the token at index i is inside parentheses
func isInsideParentheses(tokens []tokenizer.Token, index int) bool {
	parenCount := 0
	
	// Look backwards for opening parenthesis
	for i := index - 1; i >= 0; i-- {
		if tokens[i].Type == tokenizer.CLOSED_PARENS {
			parenCount++
		} else if tokens[i].Type == tokenizer.OPENED_PARENS {
			parenCount--
			if parenCount < 0 {
				// Found unmatched opening parenthesis
				return true
			}
		}
	}
	
	return false
}
