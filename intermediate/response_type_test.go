package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

func TestInferTypeFromFunction(t *testing.T) {
	tests := []struct {
		name         string
		functionName string
		expectedType string
	}{
		{
			name:         "Count",
			functionName: "count(*)",
			expectedType: "int",
		},
		{
			name:         "Sum",
			functionName: "sum(total)",
			expectedType: "number",
		},
		{
			name:         "Avg",
			functionName: "avg(score)",
			expectedType: "number",
		},
		{
			name:         "Min",
			functionName: "min(price)",
			expectedType: "any",
		},
		{
			name:         "Max",
			functionName: "max(price)",
			expectedType: "any",
		},
		{
			name:         "JsonFunction",
			functionName: "json_extract(data, '$.name')",
			expectedType: "any",
		},
		{
			name:         "ToChar",
			functionName: "to_char(created_at, 'YYYY-MM-DD')",
			expectedType: "string",
		},
		{
			name:         "ToNumber",
			functionName: "to_number(price_str)",
			expectedType: "number",
		},
		{
			name:         "ToDate",
			functionName: "to_date(date_str, 'YYYY-MM-DD')",
			expectedType: "datetime",
		},
		{
			name:         "Coalesce",
			functionName: "coalesce(name, 'Unknown')",
			expectedType: "any",
		},
		{
			name:         "UnknownFunction",
			functionName: "custom_function(arg1, arg2)",
			expectedType: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Infer type from function
			inferredType := inferTypeFromFunction(tt.functionName)

			// Verify type
			assert.Equal(t, tt.expectedType, inferredType, "Inferred type should match")
		})
	}
}

func TestInferTypeFromLiteral(t *testing.T) {
	tests := []struct {
		name         string
		literal      string
		expectedType string
	}{
		{
			name:         "StringLiteral",
			literal:      "'Hello, World!'",
			expectedType: "string",
		},
		{
			name:         "IntegerLiteral",
			literal:      "42",
			expectedType: "int",
		},
		{
			name:         "FloatLiteral",
			literal:      "3.14",
			expectedType: "number",
		},
		{
			name:         "BooleanLiteralTrue",
			literal:      "true",
			expectedType: "bool",
		},
		{
			name:         "BooleanLiteralFalse",
			literal:      "false",
			expectedType: "bool",
		},
		{
			name:         "NullLiteral",
			literal:      "NULL",
			expectedType: "null",
		},
		{
			name:         "UnknownLiteral",
			literal:      "unknown_literal",
			expectedType: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Infer type from literal
			inferredType := inferTypeFromLiteral(tt.literal)

			// Verify type
			assert.Equal(t, tt.expectedType, inferredType, "Inferred type should match")
		})
	}
}

// TestExtractFieldsFromSelectClause tests the extractFieldsFromSelectClause function directly
func TestExtractFieldsFromSelectClause(t *testing.T) {
	// Mock table information for type inference
	tableInfo := map[string]map[string]string{
		"users": {
			"id":         "int",
			"name":       "string",
			"email":      "string",
			"created_at": "datetime",
			"active":     "bool",
			"data":       "json",
			"score":      "number",
		},
	}

	// Create a mock SELECT clause
	selectClause := &parsercommon.SelectClause{
		Fields: []parsercommon.SelectField{
			{
				FieldKind:     parsercommon.SingleField,
				OriginalField: "id",
				FieldName:     "id",
				ExplicitName:  false,
			},
			{
				FieldKind:     parsercommon.TableField,
				TableName:     "users",
				OriginalField: "name",
				FieldName:     "name",
				ExplicitName:  false,
			},
			{
				FieldKind:     parsercommon.SingleField,
				OriginalField: "email",
				FieldName:     "user_email",
				ExplicitName:  true,
			},
		},
	}

	// Extract fields from the SELECT clause
	responseType := extractFieldsFromSelectClause(selectClause, tableInfo)

	// Verify fields
	assert.Equal(t, 3, len(responseType.Fields), "Number of fields should match")
	assert.Equal(t, "id", responseType.Fields[0].Name, "Field name should match")
	assert.Equal(t, "string", responseType.Fields[0].Type, "Field type should match")
	assert.Equal(t, "id", responseType.Fields[0].DatabaseTag, "Field database tag should match")
	
	assert.Equal(t, "name", responseType.Fields[1].Name, "Field name should match")
	assert.Equal(t, "string", responseType.Fields[1].Type, "Field type should match")
	assert.Equal(t, "users.name", responseType.Fields[1].DatabaseTag, "Field database tag should match")
	
	assert.Equal(t, "user_email", responseType.Fields[2].Name, "Field name should match")
	assert.Equal(t, "string", responseType.Fields[2].Type, "Field type should match")
	assert.Equal(t, "email", responseType.Fields[2].DatabaseTag, "Field database tag should match")
}

// TestExtractFieldsFromReturningClause tests the extractFieldsFromReturningClause function directly
func TestExtractFieldsFromReturningClause(t *testing.T) {
	// Mock table information for type inference
	tableInfo := map[string]map[string]string{
		"users": {
			"id":         "int",
			"name":       "string",
			"email":      "string",
			"created_at": "datetime",
		},
	}

	// Create a mock RETURNING clause
	returningClause := &parsercommon.ReturningClause{
		Fields: []parsercommon.SelectField{
			{
				FieldKind:     parsercommon.TableField,
				TableName:     "users",
				OriginalField: "id",
				FieldName:     "id",
				ExplicitName:  false,
			},
			{
				FieldKind:     parsercommon.TableField,
				TableName:     "users",
				OriginalField: "name",
				FieldName:     "name",
				ExplicitName:  false,
			},
		},
	}

	// Extract fields from the RETURNING clause
	responseType := extractFieldsFromReturningClause(returningClause, tableInfo)

	// Verify fields
	assert.Equal(t, 2, len(responseType.Fields), "Number of fields should match")
	assert.Equal(t, "id", responseType.Fields[0].Name, "Field name should match")
	assert.Equal(t, "int", responseType.Fields[0].Type, "Field type should match")
	assert.Equal(t, "users.id", responseType.Fields[0].DatabaseTag, "Field database tag should match")
	
	assert.Equal(t, "name", responseType.Fields[1].Name, "Field name should match")
	assert.Equal(t, "string", responseType.Fields[1].Type, "Field type should match")
	assert.Equal(t, "users.name", responseType.Fields[1].DatabaseTag, "Field database tag should match")
}
