package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/testhelper"
)

func TestInferTypeFromFunction(t *testing.T) {
	tests := []struct {
		name         string
		functionName string
		expectedType string
	}{
		{
			name:         "Count" + testhelper.GetCaller(t),
			functionName: "count(*)",
			expectedType: "int",
		},
		{
			name:         "Sum" + testhelper.GetCaller(t),
			functionName: "sum(total)",
			expectedType: "number",
		},
		{
			name:         "Avg" + testhelper.GetCaller(t),
			functionName: "avg(score)",
			expectedType: "number",
		},
		{
			name:         "Min" + testhelper.GetCaller(t),
			functionName: "min(price)",
			expectedType: "any",
		},
		{
			name:         "Max" + testhelper.GetCaller(t),
			functionName: "max(price)",
			expectedType: "any",
		},
		{
			name:         "JsonFunction" + testhelper.GetCaller(t),
			functionName: "json_extract(data, '$.name')",
			expectedType: "any",
		},
		{
			name:         "ToChar" + testhelper.GetCaller(t),
			functionName: "to_char(created_at, 'YYYY-MM-DD')",
			expectedType: "string",
		},
		{
			name:         "ToNumber" + testhelper.GetCaller(t),
			functionName: "to_number(price_str)",
			expectedType: "number",
		},
		{
			name:         "ToDate" + testhelper.GetCaller(t),
			functionName: "to_date(date_str, 'YYYY-MM-DD')",
			expectedType: "datetime",
		},
		{
			name:         "Coalesce" + testhelper.GetCaller(t),
			functionName: "coalesce(name, 'Unknown')",
			expectedType: "any",
		},
		{
			name:         "UnknownFunction" + testhelper.GetCaller(t),
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
			name:         "StringLiteral" + testhelper.GetCaller(t),
			literal:      "'Hello, World!'",
			expectedType: "string",
		},
		{
			name:         "IntegerLiteral" + testhelper.GetCaller(t),
			literal:      "42",
			expectedType: "int",
		},
		{
			name:         "FloatLiteral" + testhelper.GetCaller(t),
			literal:      "3.14",
			expectedType: "number",
		},
		{
			name:         "BooleanLiteralTrue" + testhelper.GetCaller(t),
			literal:      "true",
			expectedType: "bool",
		},
		{
			name:         "BooleanLiteralFalse" + testhelper.GetCaller(t),
			literal:      "false",
			expectedType: "bool",
		},
		{
			name:         "NullLiteral" + testhelper.GetCaller(t),
			literal:      "NULL",
			expectedType: "null",
		},
		{
			name:         "UnknownLiteral" + testhelper.GetCaller(t),
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
	response := extractFieldsFromSelectClause(selectClause, tableInfo)

	// Verify fields
	assert.Equal(t, 3, len(response), "Number of fields should match")
	assert.Equal(t, "id", response[0].Name, "Field name should match")
	assert.Equal(t, "int", response[0].Type, "Field type should match") // Changed from "string" to "int"
	assert.Equal(t, "id", response[0].DatabaseTag, "Field database tag should match")

	assert.Equal(t, "name", response[1].Name, "Field name should match")
	assert.Equal(t, "string", response[1].Type, "Field type should match")
	assert.Equal(t, "users.name", response[1].DatabaseTag, "Field database tag should match")

	assert.Equal(t, "user_email", response[2].Name, "Field name should match")
	assert.Equal(t, "string", response[2].Type, "Field type should match") // email field should be found in table info
	assert.Equal(t, "email", response[2].DatabaseTag, "Field database tag should match")
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
	response := extractFieldsFromReturningClause(returningClause, tableInfo)

	// Verify fields
	assert.Equal(t, 2, len(response), "Number of fields should match")
	assert.Equal(t, "id", response[0].Name, "Field name should match")
	assert.Equal(t, "int", response[0].Type, "Field type should match")
	assert.Equal(t, "users.id", response[0].DatabaseTag, "Field database tag should match")

	assert.Equal(t, "name", response[1].Name, "Field name should match")
	assert.Equal(t, "string", response[1].Type, "Field type should match")
	assert.Equal(t, "users.name", response[1].DatabaseTag, "Field database tag should match")
}
