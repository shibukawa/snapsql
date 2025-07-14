package parsercommon

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestNamespace(t *testing.T) {
	ifs := &FunctionDefinition{
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"active":      "bool",
				"departments": []any{"str"},
			},
		},
	}

	environment := map[string]any{
		"table_suffix": "prod",
		"tenant_id":    "12345",
	}

	ns := NewNamespace(ifs, environment, nil)

	// Environment constant evaluation
	result, err := ns.EvaluateEnvironmentExpression("table_suffix")
	assert.NoError(t, err)
	assert.Equal(t, "prod", result)

	// Non-existent environment constant
	_, err = ns.EvaluateEnvironmentExpression("nonexistent")
	assert.Error(t, err)

	// Parameter evaluation
	result, err = ns.EvaluateParameterExpression("user_id")
	assert.NoError(t, err)
	assert.Equal(t, 0, result) // Dummy value for int

	// Nested parameter evaluation
	result, err = ns.EvaluateParameterExpression("filters.active")
	assert.NoError(t, err)
	assert.Equal(t, false, result) // Dummy value for bool

	// Non-existent parameter
	_, err = ns.EvaluateParameterExpression("nonexistent_param")
	assert.Error(t, err)
}

func TestValueToLiteral(t *testing.T) {
	ns := NewNamespace(nil, map[string]any{}, nil)

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "string",
			value:    "test",
			expected: "'test'",
		},
		{
			name:     "string with single quote",
			value:    "test's value",
			expected: "'test''s value'",
		},
		{
			name:     "integer",
			value:    123,
			expected: "123",
		},
		{
			name:     "floating point number",
			value:    123.45,
			expected: "123.45",
		},
		{
			name:     "boolean value (true)",
			value:    true,
			expected: "true",
		},
		{
			name:     "boolean value (false)",
			value:    false,
			expected: "false",
		},
		{
			name:     "string array",
			value:    []string{"admin", "user"},
			expected: "'admin', 'user'",
		},
		{
			name:     "any array",
			value:    []any{"admin", 123, true},
			expected: "'admin', 123, true",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ns.valueToLiteral(test.value)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestLoopVariableManagement(t *testing.T) {
	schema := &FunctionDefinition{
		Parameters: map[string]any{
			"fields": []map[string]any{
				{
					"name":  "test_field",
					"type":  "str",
					"users": []string{"dummy"},
				},
			},
		},
	}

	ns := NewNamespace(schema, map[string]any{}, nil)

	// Initially should have base variables
	result, err := ns.EvaluateParameterExpression("fields")
	assert.NoError(t, err)
	assert.Equal(t, []string{"dummy"}, result.([]string))

	// Enter loop - add loop variable
	ns.EnterLoop("field", []any{"str"})

	// Should be able to access loop variable
	result, err = ns.EvaluateParameterExpression("field")
	assert.NoError(t, err)
	assert.Equal(t, "test_field", result)

	// Should still be able to access original variables
	result, err = ns.EvaluateParameterExpression("fields")
	assert.NoError(t, err)
	assert.Equal(t, []string{"dummy"}, result.([]string))

	// Enter nested loop
	ns.EnterLoop("users", []any{"dummy"})

	// Should be able to access both loop variables
	result, err = ns.EvaluateParameterExpression("field")
	assert.NoError(t, err)
	assert.Equal(t, "test_field", result)

	result, err = ns.EvaluateParameterExpression("user")
	assert.NoError(t, err)
	assert.Equal(t, "dummy", result)

	// Leave nested loop
	ns.LeaveLoop()

	// Should still have first loop variable but not second
	result, err = ns.EvaluateParameterExpression("field")
	assert.NoError(t, err)
	assert.Equal(t, "test_field", result)

	_, err = ns.EvaluateParameterExpression("user")
	assert.Error(t, err) // Should no longer be accessible

	// Leave first loop
	ns.LeaveLoop()

	// Should be back to original state
	_, err = ns.EvaluateParameterExpression("field")
	assert.Error(t, err) // Should no longer be accessible

	result, err = ns.EvaluateParameterExpression("fields")
	assert.NoError(t, err)
	assert.Equal(t, []string{"dummy"}, result.([]string))
}

func TestExtractElementFromList(t *testing.T) {
	ns := NewNamespace(nil, map[string]any{}, nil)

	tests := []struct {
		name          string
		listResult    any
		expectedValue any
		expectedType  string
		expectError   bool
	}{
		{
			name:          "string list",
			listResult:    []string{"hello", "world"},
			expectedValue: "hello",
			expectedType:  "str",
			expectError:   false,
		},
		{
			name:          "integer list",
			listResult:    []int{1, 2, 3},
			expectedValue: 1,
			expectedType:  "int",
			expectError:   false,
		},
		{
			name:          "empty string list",
			listResult:    []string{},
			expectedValue: "",
			expectedType:  "str",
			expectError:   false,
		},
		{
			name:          "any type list",
			listResult:    []any{"test", 123},
			expectedValue: "test",
			expectedType:  "str",
			expectError:   false,
		},
		{
			name:        "non-list value",
			listResult:  "not a list",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, typeStr, err := ns.extractElementFromList(tt.listResult)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedType, typeStr)
		})
	}
}

func TestDummyDataGeneration(t *testing.T) {
	tests := []struct {
		name     string
		schema   *FunctionDefinition
		expected map[string]any
	}{
		{
			name: "basic type dummy data generation",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"name":    "str",
					"age":     "int",
					"active":  "bool",
					"score":   "float",
					"fields":  []any{"str"},
					"numbers": []any{"int"},
				},
			},
			expected: map[string]any{
				"name":    "",
				"age":     0,
				"active":  false,
				"score":   0.0,
				"fields":  []string{"dummy"},
				"numbers": []int{0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDummyDataFromSchema(tt.schema)

			for key, expectedValue := range tt.expected {
				_, exists := result[key]
				assert.True(t, exists, "key '%s' should be included in result", key)
				assert.Equal(t, expectedValue, result[key])
			}
		})
	}
}

func TestEnvironmentAndParameterSeparation(t *testing.T) {
	schema := &FunctionDefinition{
		Parameters: map[string]any{
			"user_id": "int",
			"name":    "str",
		},
	}

	environment := map[string]any{
		"table_name": "users",
		"env_flag":   true,
	}

	ns := NewNamespace(schema, environment, nil)

	// Test environment variable evaluation
	result, err := ns.EvaluateEnvironmentExpression("table_name")
	assert.NoError(t, err)
	assert.Equal(t, "users", result)

	result, err = ns.EvaluateEnvironmentExpression("env_flag")
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	// Test parameter evaluation
	result, err = ns.EvaluateParameterExpression("user_id")
	assert.NoError(t, err)
	assert.Equal(t, 0, result) // Dummy value

	result, err = ns.EvaluateParameterExpression("name")
	assert.NoError(t, err)
	assert.Equal(t, "", result) // Dummy value

	// Environment variables should not be accessible from parameter evaluation
	_, err = ns.EvaluateParameterExpression("table_name")
	assert.Error(t, err)

	// Parameters should not be accessible from environment evaluation
	_, err = ns.EvaluateEnvironmentExpression("user_id")
	assert.Error(t, err)
}
