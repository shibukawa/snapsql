package gogen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestEmitUnlessBoundaryWithParenthesis(t *testing.T) {
	// Test case: EMIT_UNLESS_BOUNDARY followed by EMIT_STATIC starting with '('
	// Expected: The EMIT_UNLESS_BOUNDARY should be output (not skipped)
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestQuery",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "string"},
			{Name: "include_archived", Type: "bool"},
		},
		CELExpressions: []intermediate.CELExpression{
			{ID: "expr_001", Expression: "user_id", EnvironmentIndex: 0},
			{ID: "expr_002", Expression: "include_archived", EnvironmentIndex: 0},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT * FROM users WHERE user_id ="},
			{Op: "EMIT_EVAL", ExprIndex: intPtr(0)},
			{Op: "IF", ExprIndex: intPtr(1)},
			{Op: "EMIT_UNLESS_BOUNDARY", Value: "AND"},
			{Op: "EMIT_STATIC", Value: "(archived_at IS NULL OR archived_at > NOW())"},
			{Op: "END"},
		},
	}

	sqlData, err := processSQLBuilderWithDialect(format, "postgres", "TestQuery")
	if err != nil {
		t.Fatalf("Failed to process SQL builder: %v", err)
	}

	if sqlData.IsStatic {
		t.Fatal("Expected dynamic SQL, got static")
	}

	// Check that the generated code includes the boundary check
	code := strings.Join(sqlData.BuilderCode, "\n")

	// The code should include: if boundaryNeeded { builder.WriteString(" AND ") }
	if !strings.Contains(code, "if boundaryNeeded") {
		t.Error("Expected code to contain 'if boundaryNeeded' check")
	}

	if !strings.Contains(code, `builder.WriteString(" AND ")`) {
		t.Error("Expected code to contain boundary delimiter output")
	}

	t.Logf("Generated code:\n%s", code)
}

func TestEmitUnlessBoundaryWithClosingParenthesis(t *testing.T) {
	// Test case: EMIT_UNLESS_BOUNDARY followed by EMIT_STATIC starting with ')'
	// Expected: The EMIT_UNLESS_BOUNDARY should be skipped
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestQuery",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "string"},
			{Name: "optional_field", Type: "bool"},
		},
		CELExpressions: []intermediate.CELExpression{
			{ID: "expr_001", Expression: "user_id", EnvironmentIndex: 0},
			{ID: "expr_002", Expression: "optional_field", EnvironmentIndex: 0},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT * FROM users WHERE (user_id ="},
			{Op: "EMIT_EVAL", ExprIndex: intPtr(0)},
			{Op: "IF", ExprIndex: intPtr(1)},
			{Op: "EMIT_UNLESS_BOUNDARY", Value: "AND"},
			{Op: "EMIT_STATIC", Value: ")"},
			{Op: "END"},
		},
	}

	sqlData, err := processSQLBuilderWithDialect(format, "postgres", "TestQuery")
	if err != nil {
		t.Fatalf("Failed to process SQL builder: %v", err)
	}

	if sqlData.IsStatic {
		t.Fatal("Expected dynamic SQL, got static")
	}

	// Check that the generated code does NOT include the boundary check
	// (because it should be skipped when next token starts with ')')
	code := strings.Join(sqlData.BuilderCode, "\n")

	// Count occurrences of "if boundaryNeeded"
	// There should be at least one from the initial setup, but the EMIT_UNLESS_BOUNDARY
	// before ')' should be skipped
	t.Logf("Generated code:\n%s", code)

	// The key check: after the IF condition, there should NOT be a boundary check
	// before the ')' token
	lines := strings.Split(code, "\n")
	foundIfCondition := false
	foundBoundaryCheckAfterIf := false

	for i, line := range lines {
		if strings.Contains(line, "// IF condition") {
			foundIfCondition = true
		}

		if foundIfCondition && strings.Contains(line, "if boundaryNeeded") {
			// Check if this is before the ')' token
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], `")"`) {
					foundBoundaryCheckAfterIf = true
					break
				}
			}
		}
	}

	if foundBoundaryCheckAfterIf {
		t.Error("Expected EMIT_UNLESS_BOUNDARY to be skipped when next token is ')', but found boundary check")
	}
}
