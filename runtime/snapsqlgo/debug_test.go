package snapsqlgo

import (
	"fmt"
	"testing"
)

func TestDebugNestedConditionsExecution(t *testing.T) {
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
		{Op: "LABEL", Name: "end_where"},
	}

	params := map[string]any{
		"filters": map[string]any{},
	}

	executor, err := NewInstructionExecutor(instructions, params)
	if err != nil {
		t.Fatal(err)
	}

	// Debug step by step execution
	fmt.Printf("Starting execution with params: %+v\n", params)

	// Check the first condition
	result := executor.isExpressionTruthy("!(filters.active || filters.department)")
	fmt.Printf("First condition !(filters.active || filters.department): %v\n", result)

	sql, args, err := executor.Execute()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Final SQL: %q\n", sql)
	fmt.Printf("Final args: %v\n", args)
}
