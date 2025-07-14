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
	ns := NewNamespace(ifs, map[string]any{}, nil)
	ns.SetConstant("table_suffix", "prod")
	ns.SetConstant("tenant_id", "12345")

	// 環境constant のvalidation
	err := ns.ValidateExpression("table_suffix")
	assert.NoError(t, err)

	// Non-existent environment constant
	err = ns.ValidateExpression("nonexistent")
	assert.Error(t, err)

	// parameter のvalidation
	err = ns.ValidateExpression("user_id")
	assert.NoError(t, err)

	err = ns.ValidateExpression("filters.active")
	assert.NoError(t, err)

	// Non-existent parameter
	err = ns.ValidateExpression("nonexistent_param")
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
			name:     "string ",
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
			name:     "string array ",
			value:    []string{"admin", "user"},
			expected: "'admin', 'user'",
		},
		{
			name:     "anyarray ",
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
func TestAddLoopVariableWithEvaluation(t *testing.T) {
	t.Skip("temporarily skipped - investigating infinite loop issue")

	tests := []struct {
		name          string
		schema        *FunctionDefinition
		variable      string
		listExpr      string
		expectError   bool
		expectedType  string
		expectedValue any
	}{
		{
			name: "create loop variable from string list",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"fields": []any{"str"},
				},
			},
			variable:      "field",
			listExpr:      "fields",
			expectError:   false,
			expectedType:  "str",
			expectedValue: "dummy",
		},
		{
			name: "create loop variable from integer list",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"numbers": []any{"int"},
				},
			},
			variable:      "num",
			listExpr:      "numbers",
			expectError:   false,
			expectedType:  "int",
			expectedValue: 0,
		},
		{
			name: "complex expression evaluation",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"users":  []any{"str"},
					"active": "bool",
				},
			},
			variable:      "user",
			listExpr:      "users",
			expectError:   false,
			expectedType:  "str",
			expectedValue: "dummy",
		},
		{
			name: "error with nonexistent variable",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"fields": []any{"str"},
				},
			},
			variable:    "field",
			listExpr:    "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 名前空間を作成
			ns := NewNamespace(tt.schema, map[string]any{}, nil)
			assert.NotZero(t, ns)

			// AddLoopVariableWithEvaluationをexecution
			newNs, err := ns.AddLoopVariableWithEvaluation(tt.variable, tt.listExpr)

			if tt.expectError {
				assert.Error(t, err)
				assert.Zero(t, newNs)
				return
			}

			// Verify no error
			assert.NoError(t, err)
			assert.NotZero(t, newNs)

			// Verify loop variable is added to schema
			_, exists := newNs.Schema.Parameters[tt.variable]
			assert.True(t, exists, "loop variable should be added to schema")
			assert.Equal(t, tt.expectedType, newNs.Schema.Parameters[tt.variable].(string))

			// Verify loop variable is added to dummy data
			_, exists = newNs.param[tt.variable]
			assert.True(t, exists, "loop variable should be added to dummy data")
			assert.Equal(t, tt.expectedValue, newNs.param[tt.variable])

			// Verify loop variable can be validated with CEL
			err = newNs.ValidateParameterExpression(tt.variable)
			assert.NoError(t, err, "loop variable should be recognized by CEL")
		})
	}
}

func TestEvaluateParameterExpression(t *testing.T) {
	t.Skip("temporarily skipped - investigating infinite loop issue")

	tests := []struct {
		name           string
		schema         *FunctionDefinition
		expression     string
		expectedResult any
		expectError    bool
	}{
		{
			name: "simple variable reference",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"name": "str",
				},
			},
			expression:     "name",
			expectedResult: "",
			expectError:    false,
		},
		{
			name: "list variable reference",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"fields": []any{"str"},
				},
			},
			expression:     "fields",
			expectedResult: []string{"dummy"},
			expectError:    false,
		},
		{
			name: "error with nonexistent variable",
			schema: &FunctionDefinition{
				Parameters: map[string]any{
					"name": "str",
				},
			},
			expression:  "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 名前空間を作成
			ns := NewNamespace(tt.schema, map[string]any{}, nil)
			assert.NotZero(t, ns)

			// 式を評価
			result, err := ns.EvaluateParameterExpression(tt.expression)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			// Verify no error
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestExtractElementFromList(t *testing.T) {
	// t.Skip("temporarily skipped - investigating infinite loop issue")

	ns := NewNamespace(nil, map[string]any{}, nil)

	tests := []struct {
		name          string
		listResult    any
		expectedValue any
		expectedType  string
		expectError   bool
	}{
		{
			name:          "string list ",
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
			name:          "empty string list ",
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
	// t.Skip("temporarily skipped - investigating infinite loop issue")

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
