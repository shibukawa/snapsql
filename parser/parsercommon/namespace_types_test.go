package parsercommon

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/google/cel-go/cel"
)

func TestInferCELTypeFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *cel.Type
	}{
		{"String", "hello", cel.StringType},
		{"Int", 42, cel.IntType},
		{"Int64", int64(42), cel.IntType},
		{"Float64", 3.14, cel.DoubleType},
		{"Bool", true, cel.BoolType},
		{"Time", time.Now(), cel.TimestampType},
		{"Nil", nil, cel.DynType},
		{"Unknown", []string{"test"}, cel.DynType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCELTypeFromValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferCELTypeFromStringType(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected *cel.Type
	}{
		{"String", "string", cel.StringType},
		{"Str", "str", cel.StringType},
		{"Integer", "int", cel.IntType},
		{"Integer Full", "integer", cel.IntType},
		{"Double", "double", cel.DoubleType},
		{"Float", "float", cel.DoubleType},
		{"Boolean", "bool", cel.BoolType},
		{"Boolean Full", "boolean", cel.BoolType},
		{"Timestamp", "timestamp", cel.TimestampType},
		{"Time", "time", cel.TimestampType},
		{"Unknown", "unknown", cel.DynType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCELTypeFromStringType(tt.typeStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNullableKey(t *testing.T) {
	tests := []struct {
		name             string
		key              string
		expectedKey      string
		expectedNullable bool
	}{
		{"Normal Key", "name", "name", false},
		{"Nullable Key", "age?", "age", true},
		{"Empty Key", "", "", false},
		{"Just Question Mark", "?", "", true},
		{"Multiple Question Marks", "test??", "test?", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, nullable := isNullableKey(tt.key)
			assert.Equal(t, tt.expectedKey, key)
			assert.Equal(t, tt.expectedNullable, nullable)
		})
	}
}

func TestCreateCELVariableDeclarations(t *testing.T) {
	parameters := map[string]any{
		"name":    "string",
		"age":     "int",
		"score?":  "double",
		"active":  "bool",
		"created": "timestamp",
		"data":    "unknown",
	}

	options := createCELVariableDeclarations(parameters)

	// Create a test environment to verify the options work
	envOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}
	envOptions = append(envOptions, options...)

	env, err := cel.NewEnv(envOptions...)
	assert.NoError(t, err)
	assert.True(t, env != nil)

	// Test that variables can be used in expressions
	testExpressions := []string{
		"name == 'test'",
		"age > 0",
		"score > 0.0",
		"active == true",
		"created > timestamp('2020-01-01T00:00:00Z')",
		"data != null",
	}

	for _, expr := range testExpressions {
		ast, issues := env.Compile(expr)
		assert.True(t, issues.Err() == nil, "Expression '%s' should compile: %v", expr, issues.Err())
		assert.True(t, ast != nil)
	}
}

func TestCreateCleanParameterMap(t *testing.T) {
	parameters := map[string]any{
		"name":      "John",
		"age?":      30,
		"email":     "test@example.com",
		"optional?": "value",
		"required":  "data",
	}

	clean := createCleanParameterMap(parameters)

	expected := map[string]any{
		"name":     "John",
		"email":    "test@example.com",
		"required": "data",
	}

	assert.Equal(t, expected, clean)

	// Verify nullable parameters are excluded
	_, hasAge := clean["age"]
	assert.False(t, hasAge)
	_, hasOptional := clean["optional"]
	assert.False(t, hasOptional)
}

func TestNamespaceWithTypedParameters(t *testing.T) {
	schema := &FunctionDefinition{
		Parameters: map[string]any{
			"name":   "string",
			"age":    "int",
			"score?": "double",
			"active": "bool",
		},
	}

	environment := map[string]any{
		"env_var": "test",
	}

	param := map[string]any{
		"name":   "John",
		"age":    30,
		"active": true,
	}

	ns := NewNamespace(schema, environment, param)
	assert.True(t, ns != nil)
	assert.True(t, ns.currentCEL != nil)

	// Test CEL evaluation with typed parameters
	expr := "name == 'John' && age > 25 && active == true"
	compiled, issues := ns.currentCEL.Compile(expr)
	assert.NoError(t, issues.Err())
	assert.True(t, compiled != nil)
}
