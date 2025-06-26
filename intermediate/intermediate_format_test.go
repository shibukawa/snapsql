package intermediate

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestVariableExtractor_BasicExtraction(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
		{Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
		{Op: "JUMP_IF_EXP", Exp: "include_email", Target: 5},
		{Op: "EMIT_LITERAL", Value: ", email"},
		{Op: "EMIT_LITERAL", Value: " FROM users"},
	}

	extractor := NewVariableExtractor()
	deps := extractor.ExtractFromInstructions(instructions)

	// Check all variables
	assert.Equal(t, []string{"include_email", "user_id"}, deps.AllVariables)

	// Check structural variables (affect SQL structure)
	assert.Equal(t, []string{"include_email"}, deps.StructuralVariables)

	// Check parameter variables (only affect values)
	assert.Equal(t, []string{"user_id"}, deps.ParameterVariables)

	// Check cache key template
	assert.Equal(t, "include_email", deps.CacheKeyTemplate)
}

func TestVariableExtractor_ComplexExpressions(t *testing.T) {
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

	extractor := NewVariableExtractor()
	deps := extractor.ExtractFromInstructions(instructions)

	// Should extract "filters" as the root variable
	assert.Equal(t, []string{"filters"}, deps.AllVariables)
	assert.Equal(t, []string{"filters"}, deps.StructuralVariables)
	assert.Equal(t, []string{"filters"}, deps.ParameterVariables)
	assert.Equal(t, "filters", deps.CacheKeyTemplate)
}

func TestVariableExtractor_LoopVariables(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT "},
		{Op: "LOOP_START", Variable: "field", Collection: "fields", EndLabel: "end_loop"},
		{Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
		{Op: "EMIT_LITERAL", Value: ", "},
		{Op: "LOOP_NEXT", StartLabel: "loop_start"},
		{Op: "LOOP_END", Variable: "field", Label: "end_loop"},
		{Op: "EMIT_LITERAL", Value: "1 FROM users"},
	}

	extractor := NewVariableExtractor()
	deps := extractor.ExtractFromInstructions(instructions)

	// "fields" is structural (affects loop), "field" is parameter (loop variable)
	assert.Equal(t, []string{"field", "fields"}, deps.AllVariables)
	assert.Equal(t, []string{"fields"}, deps.StructuralVariables)
	assert.Equal(t, []string{"field"}, deps.ParameterVariables)
	assert.Equal(t, "fields", deps.CacheKeyTemplate)
}

func TestVariableExtractor_NoStructuralVariables(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
		{Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
		{Op: "EMIT_LITERAL", Value: " AND name = "},
		{Op: "EMIT_EVAL", Exp: "user.name", Placeholder: "'John'"},
	}

	extractor := NewVariableExtractor()
	deps := extractor.ExtractFromInstructions(instructions)

	// Only parameter variables, no structural variables
	assert.Equal(t, []string{"user", "user_id"}, deps.AllVariables)
	assert.Equal(t, 0, len(deps.StructuralVariables))
	assert.Equal(t, []string{"user", "user_id"}, deps.ParameterVariables)
	assert.Equal(t, "static", deps.CacheKeyTemplate) // Static SQL
}

func TestVariableDependencies_CacheKeyGeneration(t *testing.T) {
	deps := VariableDependencies{
		StructuralVariables: []string{"filters", "include_email"},
		CacheKeyTemplate:    "filters,include_email",
	}

	params1 := map[string]any{
		"filters": map[string]any{
			"active":     true,
			"department": "engineering",
		},
		"include_email": true,
	}

	params2 := map[string]any{
		"filters": map[string]any{
			"active":     true,
			"department": "engineering",
		},
		"include_email": true,
	}

	params3 := map[string]any{
		"filters": map[string]any{
			"active":     false,
			"department": "sales",
		},
		"include_email": false,
	}

	key1 := deps.GenerateCacheKey(params1)
	key2 := deps.GenerateCacheKey(params2)
	key3 := deps.GenerateCacheKey(params3)

	// Same parameters should generate same key
	assert.Equal(t, key1, key2)

	// Different parameters should generate different keys
	assert.NotEqual(t, key1, key3)

	// Keys should be reasonable length (16 characters)
	assert.Equal(t, 16, len(key1))
}

func TestVariableDependencies_StaticCacheKey(t *testing.T) {
	deps := VariableDependencies{
		CacheKeyTemplate: "static",
	}

	params := map[string]any{
		"user_id": 123,
		"name":    "John",
	}

	key := deps.GenerateCacheKey(params)
	assert.Equal(t, "static", key)
}

func TestIntermediateFormat_JSONSerialization(t *testing.T) {
	format := &IntermediateFormat{
		Source: SourceInfo{
			File:    "test.snap.sql",
			Content: "SELECT * FROM users WHERE id = /*= user_id */123",
			Hash:    "abc123",
		},
		InterfaceSchema: &InterfaceSchema{
			Name:         "GetUser",
			FunctionName: "getUser",
			Parameters: []Parameter{
				{Name: "user_id", Type: "int", Optional: false},
			},
		},
		Instructions: []Instruction{
			{Op: "EMIT_LITERAL", Pos: []int{1, 1, 0}, Value: "SELECT * FROM users WHERE id = "},
			{Op: "EMIT_PARAM", Pos: []int{1, 33, 32}, Param: "user_id", Placeholder: "123"},
		},
		Dependencies: VariableDependencies{
			AllVariables:        []string{"user_id"},
			StructuralVariables: []string{},
			ParameterVariables:  []string{"user_id"},
			CacheKeyTemplate:    "static",
		},
		Metadata: FormatMetadata{
			Version:     "1.0.0",
			GeneratedAt: time.Now().Format(time.RFC3339),
			Generator:   "snapsql-compiler",
			SchemaURL:   "https://github.com/shibukawa/snapsql/schemas/intermediate-format.json",
		},
	}

	// Test JSON serialization
	jsonData, err := format.ToJSON()
	assert.NoError(t, err)
	assert.True(t, len(jsonData) > 0)

	// Test JSON deserialization
	parsedFormat, err := FromJSON(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, format.Source.File, parsedFormat.Source.File)
	assert.Equal(t, format.InterfaceSchema.Name, parsedFormat.InterfaceSchema.Name)
	assert.Equal(t, len(format.Instructions), len(parsedFormat.Instructions))
	assert.Equal(t, format.Dependencies.CacheKeyTemplate, parsedFormat.Dependencies.CacheKeyTemplate)
}

func TestVariableExtractor_EdgeCases(t *testing.T) {
	extractor := NewVariableExtractor()

	// Test empty expression
	vars := extractor.extractVariablesFromCEL("")
	assert.Equal(t, 0, len(vars))

	// Test invalid variable names
	vars = extractor.extractVariablesFromCEL("123invalid")
	assert.Equal(t, 0, len(vars))

	// Test complex nested expression
	vars = extractor.extractVariablesFromCEL("!(user.active && (filters.department || config.enabled))")
	expected := []string{"config", "user"}
	assert.Equal(t, expected, vars)
}

func TestIsValidVariableName(t *testing.T) {
	// Valid names
	assert.True(t, isValidVariableName("user"))
	assert.True(t, isValidVariableName("user_id"))
	assert.True(t, isValidVariableName("_private"))
	assert.True(t, isValidVariableName("User123"))

	// Invalid names
	assert.False(t, isValidVariableName(""))
	assert.False(t, isValidVariableName("123user"))
	assert.False(t, isValidVariableName("user-id"))
	assert.False(t, isValidVariableName("user.id"))
}
