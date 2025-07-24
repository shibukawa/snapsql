package intermediate

import (
	"strings"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// DetermineResponseType analyzes the statement and determines the response type
func DetermineResponseType(stmt parser.StatementNode, tableInfo map[string]map[string]string) *Response {
	// Default response type
	response := &Response{
		Name:   "Result",
		Fields: []Field{},
	}
	
	// Determine response type based on statement type
	switch stmt.Type() {
	case parsercommon.SELECT_STATEMENT:
		// For SELECT statements, extract fields from the SELECT clause
		selectStmt, ok := stmt.(*parsercommon.SelectStatement)
		if ok && selectStmt.Select != nil {
			response = extractFieldsFromSelectClause(selectStmt.Select, tableInfo)
		}
		
	case parsercommon.INSERT_INTO_STATEMENT:
		// For INSERT statements, check if it has a RETURNING clause
		insertStmt, ok := stmt.(*parsercommon.InsertIntoStatement)
		if ok && insertStmt.Returning != nil {
			response = extractFieldsFromReturningClause(insertStmt.Returning, tableInfo)
		}
		
	case parsercommon.UPDATE_STATEMENT:
		// For UPDATE statements, check if it has a RETURNING clause
		updateStmt, ok := stmt.(*parsercommon.UpdateStatement)
		if ok && updateStmt.Returning != nil {
			response = extractFieldsFromReturningClause(updateStmt.Returning, tableInfo)
		}
		
	case parsercommon.DELETE_FROM_STATEMENT:
		// For DELETE statements, check if it has a RETURNING clause
		deleteStmt, ok := stmt.(*parsercommon.DeleteFromStatement)
		if ok && deleteStmt.Returning != nil {
			response = extractFieldsFromReturningClause(deleteStmt.Returning, tableInfo)
		}
	}
	
	return response
}

// extractFieldsFromSelectClause extracts fields from a SELECT clause
func extractFieldsFromSelectClause(selectClause *parsercommon.SelectClause, tableInfo map[string]map[string]string) *Response {
	response := &Response{
		Name:   "Result",
		Fields: []Field{},
	}
	
	// If the SELECT clause is nil, return empty response type
	if selectClause == nil {
		return response
	}
	
	// Extract fields from the SELECT clause
	for _, item := range selectClause.Fields {
		field := Field{
			Name: item.FieldName,
		}
		
		// If the field has an explicit name, use it
		if item.ExplicitName {
			field.Name = item.FieldName
		} else {
			// Otherwise, use the original field name
			field.Name = item.OriginalField
		}
		
		// Determine the field type
		field.Type = inferTypeFromSelectField(item, tableInfo)
		
		// Set database tag if available
		if item.TableName != "" {
			field.DatabaseTag = item.TableName + "." + item.OriginalField
		} else {
			field.DatabaseTag = item.OriginalField
		}
		
		// Add the field to the response type
		response.Fields = append(response.Fields, field)
	}
	
	return response
}

// extractFieldsFromReturningClause extracts fields from a RETURNING clause
func extractFieldsFromReturningClause(returningClause *parsercommon.ReturningClause, tableInfo map[string]map[string]string) *Response {
	response := &Response{
		Name:   "Result",
		Fields: []Field{},
	}
	
	// If the RETURNING clause is nil, return empty response type
	if returningClause == nil {
		return response
	}
	
	// Extract fields from the RETURNING clause
	for _, item := range returningClause.Fields {
		field := Field{
			Name: item.FieldName,
		}
		
		// If the field has an explicit name, use it
		if item.ExplicitName {
			field.Name = item.FieldName
		} else {
			// Otherwise, use the original field name
			field.Name = item.OriginalField
		}
		
		// Determine the field type
		field.Type = inferTypeFromSelectField(item, tableInfo)
		
		// Set database tag if available
		if item.TableName != "" {
			field.DatabaseTag = item.TableName + "." + item.OriginalField
		} else {
			field.DatabaseTag = item.OriginalField
		}
		
		// Add the field to the response type
		response.Fields = append(response.Fields, field)
	}
	
	return response
}

// inferTypeFromSelectField infers the type of a SELECT field
func inferTypeFromSelectField(field parsercommon.SelectField, tableInfo map[string]map[string]string) string {
	// If the field has an explicit type, use it
	if field.ExplicitType {
		return field.TypeName
	}
	
	// Check if it's a table field and we have type information
	if field.TableName != "" && field.OriginalField != "" {
		if tableFields, ok := tableInfo[field.TableName]; ok {
			if fieldType, ok := tableFields[field.OriginalField]; ok {
				return fieldType
			}
		}
	}
	
	// Otherwise, infer the type based on the field kind
	switch field.FieldKind {
	case parsercommon.SingleField:
		// For single fields without table info, default to "string"
		return "string"
		
	case parsercommon.TableField:
		// For table fields without type info, default to "string"
		return "string"
		
	case parsercommon.FunctionField:
		// For function fields, infer the type based on the function name
		return inferTypeFromFunction(field.OriginalField)
		
	case parsercommon.LiteralField:
		// For literal fields, infer the type from the literal value
		return inferTypeFromLiteral(field.OriginalField)
		
	case parsercommon.ComplexField:
		// For complex fields (e.g., JSON paths), default to "any"
		return "any"
		
	default:
		// For unknown field kinds, default to "any"
		return "any"
	}
}

// inferTypeFromFunction infers the return type of a SQL function
func inferTypeFromFunction(functionName string) string {
	// Extract the function name from the expression
	funcName := strings.ToLower(functionName)
	
	// Check for common aggregate functions
	if strings.HasPrefix(funcName, "count(") {
		return "int"
	} else if strings.HasPrefix(funcName, "sum(") {
		return "number"
	} else if strings.HasPrefix(funcName, "avg(") {
		return "number"
	} else if strings.HasPrefix(funcName, "min(") || strings.HasPrefix(funcName, "max(") {
		// For min/max, we can't determine the type without knowing the column type
		return "any"
	} else if strings.HasPrefix(funcName, "json_") {
		// JSON functions typically return JSON or string
		return "any"
	} else if strings.HasPrefix(funcName, "to_char(") || strings.HasPrefix(funcName, "to_text(") {
		return "string"
	} else if strings.HasPrefix(funcName, "to_number(") || strings.HasPrefix(funcName, "to_decimal(") {
		return "number"
	} else if strings.HasPrefix(funcName, "to_date(") || strings.HasPrefix(funcName, "to_timestamp(") {
		return "datetime"
	} else if strings.HasPrefix(funcName, "coalesce(") {
		// For coalesce, we can't determine the type without knowing the column types
		return "any"
	}
	
	// Default to "any" for unknown functions
	return "any"
}

// inferTypeFromLiteral infers the type of a literal value
func inferTypeFromLiteral(literal string) string {
	// Check if it's a string literal
	if strings.HasPrefix(literal, "'") && strings.HasSuffix(literal, "'") {
		return "string"
	}
	
	// Check if it's a number literal
	if strings.ContainsAny(literal, "0123456789") {
		if strings.Contains(literal, ".") {
			return "number"
		}
		return "int"
	}
	
	// Check if it's a boolean literal
	if literal == "true" || literal == "false" {
		return "bool"
	}
	
	// Check if it's a NULL literal
	if strings.ToUpper(literal) == "NULL" {
		return "null"
	}
	
	// Default to "any" for unknown literals
	return "any"
}
