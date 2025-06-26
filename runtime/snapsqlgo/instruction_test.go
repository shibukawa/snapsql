package snapsqlgo

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestInstructionExecutor_BasicEmit(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
		{Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
		{Op: "EMIT_LITERAL", Value: " AND name = "},
		{Op: "EMIT_EVAL", Exp: "user.name", Placeholder: "'John'"},
	}

	params := map[string]any{
		"user_id": 42,
		"user": map[string]any{
			"name": "Alice",
		},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT id, name FROM users WHERE id = ? AND name = ?"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, []any{42, "Alice"}, args)
}

func TestInstructionExecutor_ConditionalJump(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT id, name"},
		{Op: "JUMP_IF_EXP", Exp: "!include_email", Target: 4}, // Jump if NOT include_email
		{Op: "EMIT_LITERAL", Value: ", email"},
		{Op: "LABEL", Name: "end_email"},
		{Op: "EMIT_LITERAL", Value: " FROM users"},
	}

	// Test with include_email = true (should include email field)
	params := map[string]any{
		"include_email": true,
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT id, name, email FROM users"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, 0, len(args))

	// Test with include_email = false (should skip email field)
	params["include_email"] = false
	executor, err = NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err = executor.Execute()
	assert.NoError(t, err)

	expectedSQL = "SELECT id, name FROM users"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, 0, len(args))
}

func TestInstructionExecutor_Loop(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT "},
		{Op: "LOOP_START", Variable: "field", Collection: "fields", EndLabel: "end_loop"},
		{Op: "LABEL", Name: "loop_start"},
		{Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
		{Op: "EMIT_LITERAL", Value: ", "},
		{Op: "LOOP_NEXT", StartLabel: "loop_start"},
		{Op: "LOOP_END", Variable: "field", Label: "end_loop"},
		{Op: "EMIT_LITERAL", Value: "1 FROM users"},
	}

	params := map[string]any{
		"fields": []any{"id", "name", "email"},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT ?, ?, ?, 1 FROM users"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, []any{"id", "name", "email"}, args)
}

func TestInstructionExecutor_EmptyLoop(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT "},
		{Op: "LOOP_START", Variable: "field", Collection: "fields", EndLabel: "end_loop"},
		{Op: "LABEL", Name: "loop_start"},
		{Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
		{Op: "EMIT_LITERAL", Value: ", "},
		{Op: "LOOP_NEXT", StartLabel: "loop_start"},
		{Op: "LOOP_END", Variable: "field", Label: "end_loop"},
		{Op: "EMIT_LITERAL", Value: "1 FROM users"},
	}

	params := map[string]any{
		"fields": []any{}, // Empty collection
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT 1 FROM users"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, 0, len(args))
}

func TestInstructionExecutor_NestedConditions(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT * FROM users"},                             // 0
		{Op: "JUMP_IF_EXP", Exp: "!(filters.active || filters.department)", Target: 9}, // 1 -> jump to 9 (end_where)
		{Op: "EMIT_LITERAL", Value: " WHERE 1=1"},                                      // 2
		{Op: "JUMP_IF_EXP", Exp: "!filters.active", Target: 6},                         // 3 -> jump to 6
		{Op: "EMIT_LITERAL", Value: " AND active = "},                                  // 4
		{Op: "EMIT_EVAL", Exp: "filters.active", Placeholder: "true"},                  // 5
		{Op: "JUMP_IF_EXP", Exp: "!filters.department", Target: 9},                     // 6 -> jump to 9 (end_where)
		{Op: "EMIT_LITERAL", Value: " AND department = "},                              // 7
		{Op: "EMIT_EVAL", Exp: "filters.department", Placeholder: "'sales'"},           // 8
		{Op: "LABEL", Name: "end_where"},                                               // 9
	}

	// Test with both filters
	params := map[string]any{
		"filters": map[string]any{
			"active":     true,
			"department": "engineering",
		},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT * FROM users WHERE 1=1 AND active = ? AND department = ?"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, []any{true, "engineering"}, args)

	// Test with no filters
	params = map[string]any{
		"filters": map[string]any{},
	}

	executor, err = NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err = executor.Execute()
	assert.NoError(t, err)

	expectedSQL = "SELECT * FROM users"
	assert.Equal(t, expectedSQL, sql)
	assert.Equal(t, 0, len(args))
}

func TestInstructionExecutor_VariableScoping(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Value: "SELECT "},
		{Op: "LOOP_START", Variable: "table", Collection: "tables", EndLabel: "end_table_loop"},
		{Op: "LABEL", Name: "table_loop_start"},
		{Op: "EMIT_PARAM", Param: "table", Placeholder: "table_name"},
		{Op: "EMIT_LITERAL", Value: "."},
		{Op: "LOOP_START", Variable: "field", Collection: "fields", EndLabel: "end_field_loop"},
		{Op: "LABEL", Name: "field_loop_start"},
		{Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
		{Op: "EMIT_LITERAL", Value: ", "},
		{Op: "LOOP_NEXT", StartLabel: "field_loop_start"},
		{Op: "LOOP_END", Variable: "field", Label: "end_field_loop"},
		{Op: "LOOP_NEXT", StartLabel: "table_loop_start"},
		{Op: "LOOP_END", Variable: "table", Label: "end_table_loop"},
		{Op: "EMIT_LITERAL", Value: "1 FROM dual"},
	}

	params := map[string]any{
		"tables": []any{"users", "orders"},
		"fields": []any{"id", "name"},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	sql, args, err := executor.Execute()
	assert.NoError(t, err)

	expectedSQL := "SELECT ?.?, ?, ?.?, ?, 1 FROM dual"
	assert.Equal(t, expectedSQL, sql)
	// Loop variables should shadow the collection parameters
	assert.Equal(t, []any{"users", "id", "name", "orders", "id", "name"}, args)
}

func TestInstructionExecutor_CELEvaluation(t *testing.T) {
	instructions := []Instruction{
		{Op: "NOP"}, // Just to test CEL evaluation
	}

	params := map[string]any{
		"include_email": true,
		"filters": map[string]any{
			"active":     false,
			"department": nil,
		},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	assert.NoError(t, err)

	// Test simple boolean evaluation
	result := executor.isExpressionTruthy("include_email")
	assert.True(t, result)

	result = executor.isExpressionTruthy("!include_email")
	assert.False(t, result)

	// Test nested property evaluation
	result = executor.isExpressionTruthy("filters.active")
	assert.False(t, result)

	result = executor.isExpressionTruthy("!filters.active")
	assert.True(t, result)

	// Test complex expression
	result = executor.isExpressionTruthy("filters.active || filters.department")
	assert.False(t, result) // Both are falsy

	result = executor.isExpressionTruthy("!(filters.active || filters.department)")
	assert.True(t, result) // Negation of false is true
}
