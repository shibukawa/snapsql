package parserstep5

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Error definitions
var (
	ErrObjectInArrayContext = errors.New("object cannot be expanded in array context")
	ErrMissingObjectField   = errors.New("object missing required field for column expansion")
)

// ObjectExpansionInfo holds information about object expansion
type ObjectExpansionInfo struct {
	TokenIndex   int
	VariableName string
	ColumnOrder  []string
}

// expandArraysInValues expands array variables in VALUES clauses
// Converts: VALUES (/*= values */) -> VALUES (/*# for v : values *//*= v*/,/*# end */)
func expandArraysInValues(stmt cmn.StatementNode, funcDef *cmn.FunctionDefinition, gerr *GenerateError) {
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
	
	expandArraysInTokensWithNamespace(allTokens, namespace, gerr)
}

// expandArraysInTokensWithNamespace processes tokens using Namespace for dynamic type checking
func expandArraysInTokensWithNamespace(tokens []tokenizer.Token, namespace *cmn.Namespace, gerr *GenerateError) {
	fmt.Printf("DEBUG: expandArraysInTokensWithNamespace called with %d tokens\n", len(tokens))
	
	// Print token sequence for debugging
	fmt.Printf("DEBUG: Token sequence:\n")
	for i, token := range tokens {
		fmt.Printf("  [%d] %s: %s\n", i, token.Type, token.Value)
	}
	fmt.Printf("\n")
	
	// First pass: identify expansion markers and collect column information
	var expansionInfo []ObjectExpansionInfo
	var columnOrder []string
	
	// Extract column order from INSERT statement
	columnOrder = extractColumnOrderFromTokens(tokens)
	fmt.Printf("DEBUG: Extracted column order: %v\n", columnOrder)
	
	// Process tokens to find expansion opportunities
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
				} else if isObjectTypeWithNamespace(variableName, namespace) {
					// Check if this is in a context where object expansion is allowed
					if len(columnOrder) > 0 {
						fmt.Printf("DEBUG: Variable %s is object type - should expand\n", variableName)
						// Collect object expansion info
						expansionInfo = append(expansionInfo, ObjectExpansionInfo{
							TokenIndex:   i,
							VariableName: variableName,
							ColumnOrder:  columnOrder,
						})
						// Mark this token for object expansion
						tokens[i].Value = fmt.Sprintf("/*# EXPAND_OBJECT: %s */", variableName)
					} else {
						gerr.AddError(fmt.Errorf("%w: variable '%s' is an object but objects can only be expanded when column order is available (INSERT statements)", ErrObjectInArrayContext, variableName))
					}
				} else {
					fmt.Printf("DEBUG: Variable %s is not array or object type\n", variableName)
				}
			} else {
				fmt.Printf("DEBUG: Variable %s is not inside parentheses\n", variableName)
			}
		}
	}
	
	// Second pass: perform actual object expansions
	for _, info := range expansionInfo {
		expandObjectInTokens(tokens, info, namespace, gerr)
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

// isObjectTypeWithNamespace checks if a variable is an object type (map) using Namespace CEL evaluation
func isObjectTypeWithNamespace(variableName string, namespace *cmn.Namespace) bool {
	// Try to evaluate the variable in the current namespace context
	result, typeName, err := namespace.Eval(variableName)
	if err != nil {
		fmt.Printf("DEBUG: Failed to evaluate %s: %v\n", variableName, err)
		return false
	}
	
	fmt.Printf("DEBUG: Variable %s has CEL type: %s, Go value type: %T\n", variableName, typeName, result)
	
	// Use reflection to check if it's a map (object)
	if result != nil {
		rv := reflect.ValueOf(result)
		isObject := rv.Kind() == reflect.Map
		if isObject {
			fmt.Printf("DEBUG: Variable %s is a Go map/object (detected via reflection)\n", variableName)
		} else {
			fmt.Printf("DEBUG: Variable %s is not a Go map/object (kind: %s)\n", variableName, rv.Kind())
		}
		return isObject
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

// extractColumnOrderFromTokens extracts column order from INSERT statement tokens
func extractColumnOrderFromTokens(tokens []tokenizer.Token) []string {
	var columns []string
	inColumnList := false
	
	for i, token := range tokens {
		// Look for opening parenthesis after table name
		if token.Type == tokenizer.OPENED_PARENS && !inColumnList {
			// Check if this is the column list (not VALUES clause)
			// Look ahead to see if there's a VALUES keyword later
			hasValues := false
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == tokenizer.VALUES {
					hasValues = true
					break
				}
			}
			if hasValues {
				inColumnList = true
				continue
			}
		}
		
		// Collect column names
		if inColumnList && token.Type == tokenizer.IDENTIFIER {
			columns = append(columns, token.Value)
		}
		
		// End of column list
		if inColumnList && token.Type == tokenizer.CLOSED_PARENS {
			break
		}
	}
	
	return columns
}

// expandObjectInTokens performs the actual object expansion in tokens
func expandObjectInTokens(tokens []tokenizer.Token, info ObjectExpansionInfo, namespace *cmn.Namespace, gerr *GenerateError) {
	fmt.Printf("DEBUG: Expanding object %s at token index %d with columns: %v\n", 
		info.VariableName, info.TokenIndex, info.ColumnOrder)
	
	// Get the object value from namespace
	objectValue, _, err := namespace.Eval(info.VariableName)
	if err != nil {
		fmt.Printf("DEBUG: Failed to evaluate object %s: %v\n", info.VariableName, err)
		return
	}
	
	// Convert to map
	objectMap, ok := objectValue.(map[string]interface{})
	if !ok {
		fmt.Printf("DEBUG: Object %s is not a map: %T\n", info.VariableName, objectValue)
		return
	}
	
	// Validate that all required columns exist in the object
	for _, column := range info.ColumnOrder {
		if _, exists := objectMap[column]; !exists {
			availableFields := slices.Collect(maps.Keys(objectMap))
			gerr.AddError(fmt.Errorf("%w: object '%s' does not have required field '%s', available fields: %v", 
				ErrMissingObjectField, info.VariableName, column, availableFields))
			return
		}
	}
	
	// Generate field access expressions based on column order
	var fieldExpressions []string
	for _, column := range info.ColumnOrder {
		fieldExpressions = append(fieldExpressions, fmt.Sprintf("/*= %s.%s */", info.VariableName, column))
	}
	
	if len(fieldExpressions) > 0 {
		fmt.Printf("DEBUG: Replacing token with first field: %s\n", fieldExpressions[0])
		// Replace the EXPAND_OBJECT token with the first field expression
		tokens[info.TokenIndex].Value = fieldExpressions[0]
		
		// Create additional tokens for remaining fields
		var newTokens []tokenizer.Token
		for i := 1; i < len(fieldExpressions); i++ {
			// Add comma token
			newTokens = append(newTokens, tokenizer.Token{
				Type:  tokenizer.COMMA,
				Value: ",",
			})
			// Add space token for formatting
			newTokens = append(newTokens, tokenizer.Token{
				Type:  tokenizer.WHITESPACE,
				Value: " ",
			})
			// Add field expression token
			newTokens = append(newTokens, tokenizer.Token{
				Type:  tokenizer.BLOCK_COMMENT,
				Value: fieldExpressions[i],
			})
		}
		
		// Insert new tokens after the current token
		if len(newTokens) > 0 {
			// This is a simplified insertion - in practice, we'd need to modify the slice properly
			// For now, we'll just log what would be inserted
			fmt.Printf("DEBUG: Would insert %d new tokens after index %d\n", len(newTokens), info.TokenIndex)
			for _, token := range newTokens {
				fmt.Printf("DEBUG: New token: %s = %s\n", token.Type, token.Value)
			}
		}
	}
}
