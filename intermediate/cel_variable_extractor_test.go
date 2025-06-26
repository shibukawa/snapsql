package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestCELVariableExtractor_BasicExpressions(t *testing.T) {
	extractor, err := NewCELVariableExtractor()
	assert.NoError(t, err)

	tests := []struct {
		expression string
		expected   []string
	}{
		{"user", []string{"user"}},
		{"user.name", []string{"user"}},
		{"user.profile.email", []string{"user"}},
		{"active && enabled", []string{"active", "enabled"}},
		{"filters.active || filters.department", []string{"filters"}},
		{"!user.active", []string{"user"}},
		{"user.age > 18", []string{"user"}},
		{"", nil},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			vars, err := extractor.ExtractVariables(test.expression)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, vars)
		})
	}
}

func TestCELVariableExtractor_ComplexExpressions(t *testing.T) {
	extractor, err := NewCELVariableExtractor()
	assert.NoError(t, err)

	tests := []struct {
		expression string
		expected   []string
	}{
		{
			"user.active && (filters.department == 'engineering' || config.enabled)",
			[]string{"config", "filters", "user"},
		},
		{
			"!(filters.active || filters.department)",
			[]string{"filters"},
		},
		{
			"user.age >= config.min_age && user.department in filters.allowed_departments",
			[]string{"config", "filters", "user"},
		},
		{
			"size(items) > 0 && items[0].active",
			[]string{"items"},
		},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			vars, err := extractor.ExtractVariables(test.expression)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, vars)
		})
	}
}

func TestCELVariableExtractor_FunctionCalls(t *testing.T) {
	extractor, err := NewCELVariableExtractor()
	assert.NoError(t, err)

	tests := []struct {
		expression string
		expected   []string
	}{
		{"size(items)", []string{"items"}},
		{"user.name.upper()", []string{"user"}},
		{"has(filters.department)", []string{"filters"}},
		{"string(user.id)", []string{"user"}},
		{"user.name.startsWith(prefix)", []string{"prefix", "user"}},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			vars, err := extractor.ExtractVariables(test.expression)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, vars)
		})
	}
}

func TestCELVariableExtractor_InvalidExpressions(t *testing.T) {
	extractor, err := NewCELVariableExtractor()
	assert.NoError(t, err)

	// Invalid expressions should fall back to simple extraction
	tests := []struct {
		expression string
		expected   []string
	}{
		{"invalid..syntax", []string{"invalid"}},
		{"123invalid", []string{}},
		{"user-invalid", []string{"invalid", "user"}},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			vars, err := extractor.ExtractVariables(test.expression)
			// We expect an error for invalid expressions, but should still get fallback results
			if err != nil {
				t.Logf("Expected error for invalid expression: %v", err)
			}
			assert.Equal(t, test.expected, vars)
		})
	}
}

func TestEnhancedVariableExtractor_Instructions(t *testing.T) {
	extractor, err := NewEnhancedVariableExtractor()
	assert.NoError(t, err)

	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT * FROM users"},
		{Op: "JUMP_IF_EXP", Exp: "!(filters.active || filters.department)", Target: 8},
		{Op: "EMIT_LITERAL", Value: " WHERE 1=1"},
		{Op: "JUMP_IF_EXP", Exp: "!filters.active", Target: 6},
		{Op: "EMIT_LITERAL", Value: " AND active = "},
		{Op: "EMIT_EVAL", Exp: "filters.active", Placeholder: "true"},
		{Op: "JUMP_IF_EXP", Exp: "!filters.department", Target: 8},
		{Op: "EMIT_LITERAL", Value: " AND department = "},
		{Op: "EMIT_EVAL", Exp: "filters.department", Placeholder: "'sales'"},
	}

	deps, err := extractor.ExtractFromInstructions(instructions)
	assert.NoError(t, err)

	// Should extract "filters" as the root variable
	assert.Equal(t, []string{"filters"}, deps.AllVariables)
	assert.Equal(t, []string{"filters"}, deps.StructuralVariables)
	assert.Equal(t, []string{"filters"}, deps.ParameterVariables)
	assert.Equal(t, "filters", deps.CacheKeyTemplate)
}

func TestEnhancedVariableExtractor_MixedVariables(t *testing.T) {
	extractor, err := NewEnhancedVariableExtractor()
	assert.NoError(t, err)

	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
		{Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
		{Op: "JUMP_IF_EXP", Exp: "include_email && user.active", Target: 5},
		{Op: "EMIT_LITERAL", Value: ", email"},
		{Op: "EMIT_EVAL", Exp: "user.name.upper()", Placeholder: "'JOHN'"},
		{Op: "LOOP_START", Variable: "field", Collection: "additional_fields", EndLabel: "end_loop"},
		{Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
		{Op: "LOOP_END", Variable: "field", Label: "end_loop"},
	}

	deps, err := extractor.ExtractFromInstructions(instructions)
	assert.NoError(t, err)

	// Check all variables
	expectedAll := []string{"additional_fields", "field", "include_email", "user", "user_id"}
	assert.Equal(t, expectedAll, deps.AllVariables)

	// Check structural variables (affect SQL structure)
	expectedStructural := []string{"additional_fields", "include_email", "user"}
	assert.Equal(t, expectedStructural, deps.StructuralVariables)

	// Check parameter variables (only affect values)
	expectedParameter := []string{"field", "user", "user_id"}
	assert.Equal(t, expectedParameter, deps.ParameterVariables)

	// Check cache key template
	assert.Equal(t, "additional_fields,include_email,user", deps.CacheKeyTemplate)
}

func TestEnhancedVariableExtractor_StaticSQL(t *testing.T) {
	extractor, err := NewEnhancedVariableExtractor()
	assert.NoError(t, err)

	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
		{Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
		{Op: "EMIT_LITERAL", Value: " AND name = "},
		{Op: "EMIT_EVAL", Exp: "user.name", Placeholder: "'John'"},
	}

	deps, err := extractor.ExtractFromInstructions(instructions)
	assert.NoError(t, err)

	// Only parameter variables, no structural variables
	assert.Equal(t, []string{"user", "user_id"}, deps.AllVariables)
	assert.Equal(t, 0, len(deps.StructuralVariables))
	assert.Equal(t, []string{"user", "user_id"}, deps.ParameterVariables)
	assert.Equal(t, "static", deps.CacheKeyTemplate) // Static SQL
}

func TestEnhancedVariableExtractor_ComplexCELExpressions(t *testing.T) {
	extractor, err := NewEnhancedVariableExtractor()
	assert.NoError(t, err)

	instructions := []Instruction{
		{Op: "JUMP_IF_EXP", Exp: "user.age >= config.min_age && user.department in filters.allowed_departments", Target: 5},
		{Op: "EMIT_LITERAL", Value: "SELECT * FROM users WHERE eligible = true"},
		{Op: "EMIT_EVAL", Exp: "user.profile.email.lower()", Placeholder: "'john@example.com'"},
	}

	deps, err := extractor.ExtractFromInstructions(instructions)
	assert.NoError(t, err)

	// Should extract all root variables
	expectedAll := []string{"config", "filters", "user"}
	assert.Equal(t, expectedAll, deps.AllVariables)

	// Structural variables from JUMP_IF_EXP
	expectedStructural := []string{"config", "filters", "user"}
	assert.Equal(t, expectedStructural, deps.StructuralVariables)

	// Parameter variables from EMIT_EVAL
	expectedParameter := []string{"user"}
	assert.Equal(t, expectedParameter, deps.ParameterVariables)
}

func TestCELVariableExtractor_EdgeCases(t *testing.T) {
	extractor, err := NewCELVariableExtractor()
	assert.NoError(t, err)

	tests := []struct {
		name       string
		expression string
		expected   []string
	}{
		{"Empty expression", "", nil},
		{"Literal only", "true", []string{}},
		{"Number literal", "42", []string{}},
		{"String literal", "'hello'", []string{}},
		{"List literal", "[1, 2, 3]", []string{}},
		{"Map literal", "{'key': 'value'}", []string{}},
		{"Mixed literals and variables", "user.name == 'John' && user.age > 25", []string{"user"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vars, err := extractor.ExtractVariables(test.expression)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, vars)
		})
	}
}
