package intermediate

import (
	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/typeinference"
)

// DetermineResponseType analyzes the statement and determines the response type
func DetermineResponseType(stmt parser.StatementNode, tableInfo map[string]*TableInfo) []Response {
	// Convert tableInfo to DatabaseSchema format for typeinference package
	schemas := convertTableInfoToSchemas(tableInfo)

	// Use typeinference package for complete type inference
	inferredFields, err := typeinference.InferFieldTypes(schemas, stmt, nil)
	if err != nil {
		// Log error but continue with empty response
		// In production, you might want to handle this differently
		return []Response{}
	}

	// Convert InferredFieldInfo to Response format
	var fields []Response
	for _, field := range inferredFields {
		response := Response{
			Name: field.Name,
		}

		// Convert TypeInfo to string type
		if field.Type != nil {
			response.Type = field.Type.BaseType
		} else {
			response.Type = "any"
		}

		fields = append(fields, response)
	}

	return fields
}

// convertTableInfoToSchemas converts intermediate.TableInfo to typeinference.DatabaseSchema
func convertTableInfoToSchemas(tableInfo map[string]*TableInfo) []DatabaseSchema {
	if len(tableInfo) == 0 {
		return nil
	}

	// Create a single database schema with all tables
	schema := DatabaseSchema{
		Name:   "default", // Default database name
		Tables: make([]*TableInfo, 0, len(tableInfo)),
	}

	// TableInfo is already in the correct format, just add to schema
	for _, info := range tableInfo {
		schema.Tables = append(schema.Tables, info)
	}

	return []DatabaseSchema{schema}
}
