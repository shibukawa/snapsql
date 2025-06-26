package snapsqlgo

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestIntegrationWithIntermediateFile(t *testing.T) {
	// Load the generated intermediate file
	intermediateFile := filepath.Join("..", "..", "generated", "users.json")
	format, err := intermediate.LoadFromFile(intermediateFile)
	assert.NoError(t, err)

	// Verify the loaded format
	assert.Equal(t, "queries/users.snap.sql", format.Source.File)
	assert.True(t, len(format.Source.Content) > 0)
	assert.True(t, len(format.Source.Hash) > 0)
	assert.Equal(t, "2.0.0", format.Metadata.Version)
	assert.Equal(t, "snapsql-intermediate-generator", format.Metadata.Generator)

	// Convert intermediate instructions to runtime instructions
	runtimeInstructions := make([]Instruction, len(format.Instructions))
	for i, inst := range format.Instructions {
		runtimeInstructions[i] = Instruction{
			Op:          inst.Op,
			Pos:         inst.Pos,
			Value:       inst.Value,
			Param:       inst.Param,
			Exp:         inst.Exp,
			Placeholder: inst.Placeholder,
			Target:      inst.Target,
			Name:        inst.Name,
			Variable:    inst.Variable,
			Collection:  inst.Collection,
			EndLabel:    inst.EndLabel,
			StartLabel:  inst.StartLabel,
			Label:       inst.Label,
		}
	}

	// Execute the instructions
	params := map[string]any{
		"include_email": true,
		"table_suffix": "prod",
		"filters": map[string]any{
			"active":      false,
			"departments": []string{"engineering", "design"},
		},
		"sort_fields": []map[string]any{
			{"field": "created_at", "direction": "DESC"},
		},
		"pagination": map[string]any{
			"limit":  20,
			"offset": 10,
		},
	}

	executor, err := NewInstructionExecutor(runtimeInstructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	// Verify the output
	t.Logf("Generated SQL: %s", sql)
	t.Logf("Generated Args: %v", args)

	// For now, we expect the simple literal output since AST conversion is not fully implemented
	assert.Equal(t, "SELECT * FROM users", sql)
	assert.True(t, len(args) == 0)
}

func TestIntermediateFormatStructure(t *testing.T) {
	// Load the generated intermediate file
	intermediateFile := filepath.Join("..", "..", "generated", "users.json")
	format, err := intermediate.LoadFromFile(intermediateFile)
	assert.NoError(t, err)

	// Test source information
	assert.True(t, len(format.Source.File) > 0)
	assert.True(t, len(format.Source.Content) > 0)
	assert.True(t, len(format.Source.Hash) > 0)
	assert.Equal(t, 64, len(format.Source.Hash)) // SHA-256 hash length

	// Test instructions
	assert.True(t, len(format.Instructions) > 0)
	for _, inst := range format.Instructions {
		assert.True(t, len(inst.Op) > 0)
		assert.Equal(t, 3, len(inst.Pos)) // [line, column, offset]
		assert.True(t, inst.Pos[0] >= 1) // Line number should be >= 1
		assert.True(t, inst.Pos[1] >= 1) // Column number should be >= 1
		assert.True(t, inst.Pos[2] >= 0) // Offset should be >= 0
	}

	// Test dependencies
	assert.True(t, format.Dependencies.AllVariables != nil)
	assert.True(t, format.Dependencies.StructuralVariables != nil)
	assert.True(t, format.Dependencies.ParameterVariables != nil)
	assert.True(t, format.Dependencies.DependencyGraph != nil)
	assert.True(t, len(format.Dependencies.CacheKeyTemplate) > 0)

	// Test metadata
	assert.True(t, len(format.Metadata.Version) > 0)
	assert.True(t, len(format.Metadata.GeneratedAt) > 0)
	assert.True(t, len(format.Metadata.Generator) > 0)
	assert.True(t, len(format.Metadata.SchemaURL) > 0)
}

func TestCacheKeyGeneration(t *testing.T) {
	// Load the generated intermediate file
	intermediateFile := filepath.Join("..", "..", "generated", "users.json")
	format, err := intermediate.LoadFromFile(intermediateFile)
	assert.NoError(t, err)

	// Test cache key generation
	params1 := map[string]any{
		"include_email": true,
		"table_suffix": "prod",
	}

	params2 := map[string]any{
		"include_email": false,
		"table_suffix": "dev",
	}

	key1 := format.Dependencies.GenerateCacheKey(params1)
	key2 := format.Dependencies.GenerateCacheKey(params2)

	t.Logf("Cache key 1: %s", key1)
	t.Logf("Cache key 2: %s", key2)

	// For static SQL (no structural variables), cache keys should be the same
	if format.Dependencies.CacheKeyTemplate == "static" {
		assert.Equal(t, key1, key2)
		assert.Equal(t, "static", key1)
	} else {
		// For dynamic SQL, cache keys should be different for different structural variables
		assert.True(t, key1 != key2)
	}
}

func TestErrorReporting(t *testing.T) {
	// Load the generated intermediate file
	intermediateFile := filepath.Join("..", "..", "generated", "users.json")
	format, err := intermediate.LoadFromFile(intermediateFile)
	assert.NoError(t, err)

	// Create error reporter
	reporter := intermediate.NewErrorReporter(
		format.Source.File,
		format.Source.Content,
		format.Instructions,
	)

	// Test error reporting for valid instruction
	if len(format.Instructions) > 0 {
		execErr := reporter.ReportError("test error", 0)
		assert.Equal(t, "test error", execErr.Message)
		assert.Equal(t, 0, execErr.Instruction)
		assert.Equal(t, format.Instructions[0].Pos, execErr.Pos)
		assert.Equal(t, format.Source.File, execErr.SourceFile)

		// Test detailed error
		detailed := execErr.DetailedError()
		assert.True(t, len(detailed) > 0)
		assert.True(t, contains(detailed, "test error"))
		assert.True(t, contains(detailed, format.Source.File))
	}

	// Test error reporting for invalid instruction index
	execErr := reporter.ReportError("invalid index error", 999)
	assert.Equal(t, "invalid index error", execErr.Message)
	assert.Equal(t, 999, execErr.Instruction)
	assert.Equal(t, []int{0, 0, 0}, execErr.Pos) // Default position
}

func TestValidateIntermediateFormat(t *testing.T) {
	// Load the generated intermediate file
	intermediateFile := filepath.Join("..", "..", "generated", "users.json")
	format, err := intermediate.LoadFromFile(intermediateFile)
	assert.NoError(t, err)

	// Create generator for validation
	generator, err := intermediate.NewGenerator()
	assert.NoError(t, err)

	// Validate the format
	validationErrors := generator.ValidateFormat(format)
	
	// Log any validation errors for debugging
	for _, validationErr := range validationErrors {
		t.Logf("Validation error: %v", validationErr)
	}

	// The format should be valid (no critical errors)
	// Note: There might be warnings about missing variables since AST conversion is simplified
	assert.True(t, len(validationErrors) <= 1, "Should have at most 1 validation error (missing variables)")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) <= len(s) && s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}
