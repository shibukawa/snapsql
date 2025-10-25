package gogen

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemColumnsIntegration(t *testing.T) {
	// Test intermediate format with system columns
	intermediateJSON := `{
		"cel_environments": [{"index": 0, "additional_variables": []}],
		"cel_expressions": [
			{"id": "expr_001", "expression": "name", "environment_index": 0, "position": {"line": 0, "column": 0}},
			{"id": "expr_002", "expression": "email", "environment_index": 0, "position": {"line": 0, "column": 0}}
		],
		"description": "Create a new user with automatic system column handling.",
		"format_version": "1",
		"function_name": "create_user_with_system_columns",
		"implicit_parameters": [
			{"name": "created_at", "type": "timestamp", "default": "NOW()"},
			{"name": "updated_at", "type": "timestamp", "default": "NOW()"},
			{"name": "created_by", "type": "int"},
			{"name": "version", "type": "int", "default": 1}
		],
		"instructions": [
			{"op": "EMIT_STATIC", "pos": "179:1", "value": "INSERT INTO users (name, email, created_at, updated_at, created_by, version) VALUES ("},
			{"op": "EMIT_EVAL", "pos": "181:9", "expr_index": 0},
			{"op": "EMIT_STATIC", "pos": "181:30", "value": ","},
			{"op": "EMIT_EVAL", "pos": "181:32", "expr_index": 1},
			{"op": "EMIT_STATIC", "pos": "0:0", "value": ", "},
			{"op": "EMIT_SYSTEM_VALUE", "pos": "181:62", "system_field": "created_at"},
			{"op": "EMIT_STATIC", "pos": "0:0", "value": ", "},
			{"op": "EMIT_SYSTEM_VALUE", "pos": "181:62", "system_field": "updated_at"},
			{"op": "EMIT_STATIC", "pos": "0:0", "value": ", "},
			{"op": "EMIT_SYSTEM_VALUE", "pos": "181:62", "system_field": "created_by"},
			{"op": "EMIT_STATIC", "pos": "0:0", "value": ", "},
			{"op": "EMIT_SYSTEM_VALUE", "pos": "181:62", "system_field": "version"},
			{"op": "EMIT_STATIC", "pos": "181:62", "value": ")"}
		],
		"parameters": [
			{"name": "name", "type": "string"},
			{"name": "email", "type": "string"}
		],
		"response_affinity": "none",
		"system_fields": [
			{"name": "created_at", "on_insert": {"default": "NOW()", "parameter": "implicit"}},
			{"name": "updated_at", "on_insert": {"default": "NOW()", "parameter": "implicit"}, "on_update": {"default": "NOW()", "parameter": "implicit"}},
			{"name": "created_by", "on_insert": {"parameter": "implicit"}},
			{"name": "version", "on_insert": {"default": 1, "parameter": "implicit"}, "on_update": {"parameter": "implicit"}}
		]
	}`

	var format intermediate.IntermediateFormat

	err := json.Unmarshal([]byte(intermediateJSON), &format)
	require.NoError(t, err)

	// Create generator
	generator := &Generator{
		PackageName: "testgen",
		Format:      &format,
		Dialect:     "postgres",
	}

	// Generate Go code
	var output strings.Builder

	err = generator.Generate(&output)
	require.NoError(t, err)

	generatedCode := output.String()

	// Verify generated code contains expected elements
	assert.Contains(t, generatedCode, "func CreateUserWithSystemColumns")
	assert.Contains(t, generatedCode, "sql.Result")
	assert.Contains(t, generatedCode, "ImplicitParamSpec")
	assert.Contains(t, generatedCode, "ExtractImplicitParams")
	assert.Contains(t, generatedCode, "systemValues")

	// Verify PostgreSQL placeholders
	assert.Contains(t, generatedCode, "$1,$2, $3, $4, $5, $6")

	// Verify system column handling with default values
	assert.Contains(t, generatedCode, `{Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()}`)
	assert.Contains(t, generatedCode, `{Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()}`)
	assert.Contains(t, generatedCode, `{Name: "created_by", Type: "int", Required: true}`)
	assert.Contains(t, generatedCode, `{Name: "version", Type: "int", Required: false, DefaultValue: 1}`)

	// Verify system values are used in arguments
	assert.Contains(t, generatedCode, `systemValues["created_at"]`)
	assert.Contains(t, generatedCode, `systemValues["updated_at"]`)
	assert.Contains(t, generatedCode, `systemValues["created_by"]`)
	assert.Contains(t, generatedCode, `systemValues["version"]`)
}

func TestSystemColumnsCodeExecution(t *testing.T) {
	// Test that the generated code logic works correctly
	now := time.Now()

	// Set up context with system values
	ctx := snapsqlgo.WithSystemValue(t.Context(), "created_by", 123)
	ctx = snapsqlgo.WithSystemValue(ctx, "created_at", now)
	ctx = snapsqlgo.WithSystemValue(ctx, "updated_at", now)
	ctx = snapsqlgo.WithSystemValue(ctx, "version", 1)

	// Test implicit parameter extraction (simulating generated code)
	implicitSpecs := []snapsqlgo.ImplicitParamSpec{
		{Name: "created_at", Type: "time.Time", Required: false},
		{Name: "updated_at", Type: "time.Time", Required: false},
		{Name: "created_by", Type: "int", Required: true},
		{Name: "version", Type: "int", Required: false},
	}

	systemValues := snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)

	// Verify extracted values
	assert.Equal(t, now, systemValues["created_at"])
	assert.Equal(t, now, systemValues["updated_at"])
	assert.Equal(t, 123, systemValues["created_by"])
	assert.Equal(t, 1, systemValues["version"])

	// Test SQL argument construction (simulating generated code)
	args := []any{
		"John Doe",                 // name parameter
		"john@example.com",         // email parameter
		systemValues["created_at"], // system value
		systemValues["updated_at"], // system value
		systemValues["created_by"], // system value
		systemValues["version"],    // system value
	}

	// Verify arguments are correctly constructed
	assert.Equal(t, "John Doe", args[0])
	assert.Equal(t, "john@example.com", args[1])
	assert.Equal(t, now, args[2])
	assert.Equal(t, now, args[3])
	assert.Equal(t, 123, args[4])
	assert.Equal(t, 1, args[5])
}

func TestSystemColumnsWithMissingRequiredValue(t *testing.T) {
	// Don't set required created_by value
	ctx := snapsqlgo.WithSystemValue(t.Context(), "version", 1)

	implicitSpecs := []snapsqlgo.ImplicitParamSpec{
		{Name: "created_by", Type: "int", Required: true},
		{Name: "version", Type: "int", Required: false},
	}

	// Should panic when required parameter is missing
	assert.Panics(t, func() {
		snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)
	})
}

func TestSystemColumnsWithDefaults(t *testing.T) {
	// Only set some values, let others use defaults
	ctx := snapsqlgo.WithSystemValue(t.Context(), "created_by", 123)

	implicitSpecs := []snapsqlgo.ImplicitParamSpec{
		{Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "created_by", Type: "int", Required: true},
		{Name: "version", Type: "int", Required: false, DefaultValue: 1},
	}

	systemValues := snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)

	// Verify values
	assert.Equal(t, 123, systemValues["created_by"])
	assert.Equal(t, 1, systemValues["version"])  // default value from spec
	assert.NotNil(t, systemValues["created_at"]) // time.Now() from spec
	assert.NotNil(t, systemValues["updated_at"]) // time.Now() from spec
}

func TestOptimizedInstructionsWithSystemColumns(t *testing.T) {
	// Test that optimizer correctly handles system column instructions
	instructions := []intermediate.Instruction{
		{Op: intermediate.OpEmitStatic, Value: "INSERT INTO users (name, "},
		{Op: intermediate.OpEmitEval, ExprIndex: intPtr(0)},
		{Op: intermediate.OpEmitStatic, Value: ", created_at) VALUES ("},
		{Op: intermediate.OpEmitSystemValue, SystemField: "created_at"},
		{Op: intermediate.OpEmitStatic, Value: ")"},
	}

	optimized, err := codegenerator.OptimizeInstructions(instructions, "postgres")
	require.NoError(t, err)

	// Verify optimization results based on actual output
	require.Equal(t, 5, len(optimized), "Should have 5 optimized instructions")

	assert.Equal(t, "EMIT_STATIC", optimized[0].Op)
	assert.Contains(t, optimized[0].Value, "INSERT INTO users (name, ")

	assert.Equal(t, "ADD_PARAM", optimized[1].Op)
	assert.NotNil(t, optimized[1].ExprIndex)
	assert.Equal(t, 0, *optimized[1].ExprIndex)

	assert.Equal(t, "EMIT_STATIC", optimized[2].Op)
	assert.Contains(t, optimized[2].Value, ", created_at) VALUES (")

	assert.Equal(t, "ADD_SYSTEM_PARAM", optimized[3].Op)
	assert.Equal(t, "created_at", optimized[3].SystemField)

	assert.Equal(t, "EMIT_STATIC", optimized[4].Op)
	assert.Equal(t, ")", optimized[4].Value)
}

func intPtr(i int) *int {
	return &i
}
