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
	ErrObjectInInClause     = errors.New("object cannot be used in IN clause")
)

// ObjectExpansionInfo holds information about object expansion
type ObjectExpansionInfo struct {
	TokenIndex   int
	VariableName string
	ColumnOrder  []string
}

// ObjectArrayExpansionInfo holds information about object array expansion
type ObjectArrayExpansionInfo struct {
	TokenIndex   int
	VariableName string
	ColumnOrder  []string
}

// expandArraysInValues expands array variables in VALUES clauses
// Converts: VALUES (/*= values */) -> VALUES (/*# for v : values *//*= v*/,/*# end */)
func expandArraysInValues(stmt cmn.StatementNode, funcDef *cmn.FunctionDefinition, gerr *GenerateError) {
	// Only process INSERT statements
	if stmt.Type() != cmn.INSERT_INTO_STATEMENT {
		// For non-INSERT statements, check for object usage in IN clauses
		checkObjectUsageInStatement(stmt, funcDef, gerr)
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
	// First pass: identify expansion markers and collect column information
	var expansionInfo []ObjectExpansionInfo
	var objectArrayExpansionInfo []ObjectArrayExpansionInfo
	var columnOrder []string

	// Extract column order from INSERT statement
	columnOrder = extractColumnOrderFromTokens(tokens)

	// Process tokens to find expansion opportunities
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// Handle FOR directive - enter loop scope
		if isForDirective(token) {
			loopVar, arrayVar := parseForDirective(token)
			if loopVar != "" && arrayVar != "" {

				// Evaluate the array variable to get its actual value
				arrayValue, _, err := namespace.Eval(arrayVar)
				if err != nil {
					continue
				}

				// Convert to []any using reflection
				arraySlice, err := convertToAnySlice(arrayValue)
				if err != nil {
					continue
				}

				// Enter loop with the actual array values
				err = namespace.EnterLoop(loopVar, arraySlice)
				if err != nil {
				}
			}
			continue
		}

		// Handle END directive - exit loop scope
		if isEndDirective(token) {
			namespace.ExitLoop()
			continue
		}

		// Look for variable directive pattern: /*= variable_name */
		if isVariableDirective(token) {
			variableName := extractVariableName(token)

			// Check if this is inside parentheses (VALUES clause pattern)
			if isInsideParentheses(tokens, i) {

				// Use Namespace to check variable type in priority order
				if isObjectArrayTypeWithNamespace(variableName, namespace) {
					// Check if this is inside an IN clause - object arrays are not allowed in IN clauses
					if isInsideInClause(tokens, i) {
						gerr.AddError(fmt.Errorf("%w: variable '%s' is an object array but object arrays cannot be used in IN clauses", ErrObjectInInClause, variableName))
					} else if len(columnOrder) > 0 {
						// Check if this is in a context where object array expansion is allowed
						// Collect object array expansion info
						objectArrayExpansionInfo = append(objectArrayExpansionInfo, ObjectArrayExpansionInfo{
							TokenIndex:   i,
							VariableName: variableName,
							ColumnOrder:  columnOrder,
						})
						// Mark this token for object array expansion
						tokens[i].Value = fmt.Sprintf("/*# EXPAND_OBJECT_ARRAY: %s */", variableName)
					} else {
						gerr.AddError(fmt.Errorf("%w: variable '%s' is an object array but objects can only be expanded when column order is available (INSERT statements)", ErrObjectInArrayContext, variableName))
					}
				} else if isArrayTypeWithNamespace(variableName, namespace) {
					// Mark this token for expansion
					tokens[i].Value = fmt.Sprintf("/*# EXPAND_ARRAY: %s */", variableName)
				} else if isObjectTypeWithNamespace(variableName, namespace) {
					// Check if this is inside an IN clause - objects are not allowed in IN clauses
					if isInsideInClause(tokens, i) {
						gerr.AddError(fmt.Errorf("%w: variable '%s' is an object but objects cannot be used in IN clauses", ErrObjectInInClause, variableName))
					} else if len(columnOrder) > 0 {
						// Check if this is in a context where object expansion is allowed
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
				}
			} else {
			}
		}
	}

	// Second pass: perform actual object expansions
	for _, info := range expansionInfo {
		expandObjectInTokens(tokens, info, namespace, gerr)
	}

	// Third pass: perform actual object array expansions
	for _, info := range objectArrayExpansionInfo {
		expandObjectArrayInTokens(tokens, info, namespace, gerr)
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
	result, _, err := namespace.Eval(variableName)
	if err != nil {
		return false
	}

	// Use reflection to check if it's a slice or array
	if result != nil {
		rv := reflect.ValueOf(result)
		isArray := rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
		if isArray {
		} else {
		}
		return isArray
	}

	return false
}

// isObjectTypeWithNamespace checks if a variable is an object type (map) using Namespace CEL evaluation
func isObjectTypeWithNamespace(variableName string, namespace *cmn.Namespace) bool {
	// Try to evaluate the variable in the current namespace context
	result, _, err := namespace.Eval(variableName)
	if err != nil {
		return false
	}

	// Use reflection to check if it's a map (object)
	if result != nil {
		rv := reflect.ValueOf(result)
		isObject := rv.Kind() == reflect.Map
		if isObject {
		} else {
		}
		return isObject
	}

	return false
}

// isObjectArrayTypeWithNamespace checks if a variable is an array of objects using Namespace CEL evaluation
func isObjectArrayTypeWithNamespace(variableName string, namespace *cmn.Namespace) bool {
	// Try to evaluate the variable in the current namespace context
	result, _, err := namespace.Eval(variableName)
	if err != nil {
		return false
	}

	// Use reflection to check if it's a slice or array
	if result != nil {
		rv := reflect.ValueOf(result)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			// Check if the array has elements and the first element is an object
			if rv.Len() > 0 {
				firstElement := rv.Index(0).Interface()
				if firstElement != nil {
					firstRv := reflect.ValueOf(firstElement)
					isObjectArray := firstRv.Kind() == reflect.Map
					if isObjectArray {
					} else {
					}
					return isObjectArray
				}
			}
			return false
		}
		return false
	}

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

// isInsideInClause checks if a token position is inside an IN clause parentheses
func isInsideInClause(tokens []tokenizer.Token, tokenIndex int) bool {
	// Look backwards to find if there's an IN keyword before the opening parenthesis
	parenDepth := 0

	for i := tokenIndex - 1; i >= 0; i-- {
		token := tokens[i]

		if token.Type == tokenizer.CLOSED_PARENS {
			parenDepth++
		} else if token.Type == tokenizer.OPENED_PARENS {
			if parenDepth == 0 {
				// Look backwards from the opening parenthesis to find IN keyword
				for j := i - 1; j >= 0; j-- {
					prevToken := tokens[j]
					if prevToken.Type == tokenizer.WHITESPACE {
						continue
					}
					if (prevToken.Type == tokenizer.IDENTIFIER || prevToken.Type == tokenizer.RESERVED_IDENTIFIER) && strings.ToUpper(prevToken.Value) == "IN" {
						return true
					}
					// If we hit a non-whitespace token that's not IN, stop looking
					break
				}
				break
			} else {
				parenDepth--
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
	// Get the object value from namespace
	objectValue, _, err := namespace.Eval(info.VariableName)
	if err != nil {
		return
	}

	// Convert to map
	objectMap, ok := objectValue.(map[string]interface{})
	if !ok {
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
			for range newTokens {
			}
		}
	}
}

// expandObjectArrayInTokens performs the actual object array expansion in tokens
func expandObjectArrayInTokens(tokens []tokenizer.Token, info ObjectArrayExpansionInfo, namespace *cmn.Namespace, gerr *GenerateError) {
	// Get the array value from namespace
	arrayValue, _, err := namespace.Eval(info.VariableName)
	if err != nil {
		return
	}

	// Convert to slice
	rv := reflect.ValueOf(arrayValue)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return
	}

	arrayLength := rv.Len()
	if arrayLength == 0 {
		return
	}

	// Validate that all objects in the array have required fields
	for i := 0; i < arrayLength; i++ {
		element := rv.Index(i).Interface()
		objectMap, ok := element.(map[string]interface{})
		if !ok {
			gerr.AddError(fmt.Errorf("array element at index %d is not an object: %T", i, element))
			return
		}

		// Check all required columns exist in this object
		for _, column := range info.ColumnOrder {
			if _, exists := objectMap[column]; !exists {
				availableFields := slices.Collect(maps.Keys(objectMap))
				gerr.AddError(fmt.Errorf("%w: object at index %d in array '%s' does not have required field '%s', available fields: %v",
					ErrMissingObjectField, i, info.VariableName, column, availableFields))
				return
			}
		}
	}

	// Generate VALUES clauses for each object in the array
	var valuesClauses []string
	for i := 0; i < arrayLength; i++ {
		var fieldExpressions []string
		for _, column := range info.ColumnOrder {
			fieldExpressions = append(fieldExpressions, fmt.Sprintf("/*= %s[%d].%s */", info.VariableName, i, column))
		}
		valuesClauses = append(valuesClauses, fmt.Sprintf("(%s)", strings.Join(fieldExpressions, ", ")))
	}

	if len(valuesClauses) > 0 {
		// Replace the EXPAND_OBJECT_ARRAY token with the first VALUES clause
		tokens[info.TokenIndex].Value = valuesClauses[0]

		// Create additional tokens for remaining VALUES clauses
		var newTokens []tokenizer.Token
		for i := 1; i < len(valuesClauses); i++ {
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
			// Add VALUES clause token
			newTokens = append(newTokens, tokenizer.Token{
				Type:  tokenizer.BLOCK_COMMENT,
				Value: valuesClauses[i],
			})
		}

		// Insert new tokens after the current token
		if len(newTokens) > 0 {
			// This is a simplified insertion - in practice, we'd need to modify the slice properly
			// For now, we'll just log what would be inserted
			for range newTokens {
			}
		}
	}
}

// checkObjectUsageInStatement checks for invalid object usage in non-INSERT statements
func checkObjectUsageInStatement(stmt cmn.StatementNode, funcDef *cmn.FunctionDefinition, gerr *GenerateError) {
	if funcDef == nil {
		return
	}

	// Extract tokens from the statement using the same method as generator.go
	tokens := []tokenizer.Token{}
	for _, clause := range stmt.Clauses() {
		tokens = append(tokens, clause.RawTokens()...)
	}
	if cte := stmt.CTE(); cte != nil {
		tokens = append(tokens, cte.RawTokens()...)
	}
	tokens = append(tokens, stmt.LeadingTokens()...)

	if len(tokens) == 0 {
		return
	}

	// Create namespace for type checking
	namespace, err := cmn.NewNamespaceFromDefinition(funcDef)
	if err != nil {
		return
	}

	// Check each token for variable directives
	for i, token := range tokens {
		if isVariableDirective(token) {
			variableName := extractVariableName(token)

			// Check if this is an object type
			if isObjectTypeWithNamespace(variableName, namespace) {
				// Check if this is inside an IN clause
				if isInsideInClause(tokens, i) {
					gerr.AddError(fmt.Errorf("%w: variable '%s' is an object but objects cannot be used in IN clauses", ErrObjectInInClause, variableName))
				}
			} else if isObjectArrayTypeWithNamespace(variableName, namespace) {
				// Check if this is inside an IN clause
				if isInsideInClause(tokens, i) {
					gerr.AddError(fmt.Errorf("%w: variable '%s' is an object array but object arrays cannot be used in IN clauses", ErrObjectInInClause, variableName))
				}
			}
		}
	}
}
