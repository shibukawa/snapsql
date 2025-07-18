package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestExecutionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ExecutionError
		expected string
	}{
		{
			name: "Error with file and position",
			err: &ExecutionError{
				Message:    "variable not found",
				Pos:        []int{5, 12, 45},
				SourceFile: "queries/users.snap.sql",
			},
			expected: "queries/users.snap.sql:5:12: variable not found",
		},
		{
			name: "Error with position only",
			err: &ExecutionError{
				Message: "invalid expression",
				Pos:     []int{3, 8, 20},
			},
			expected: "line 3, column 8: invalid expression",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.err.Error())
		})
	}
}

func TestExecutionError_DetailedError(t *testing.T) {
	err := &ExecutionError{
		Message:    "variable 'include_email' not found",
		Pos:        []int{1, 18, 17},
		SourceFile: "test.sql",
		SourceLine: "SELECT id, name /*# if include_email */, email",
	}

	detailed := err.DetailedError()

	// Should contain the basic error message
	assert.True(t, strings.Contains(detailed, "test.sql:1:18: variable 'include_email' not found"))

	// Should contain the source line
	assert.True(t, strings.Contains(detailed, "SELECT id, name /*# if include_email */, email"))

	// Should contain the pointer
	assert.True(t, strings.Contains(detailed, "^"))
}

func TestNewExecutionError(t *testing.T) {
	sourceContent := `SELECT id, name /*# if include_email */, email
FROM users 
WHERE id = /*= user_id */123`

	instruction := &Instruction{
		Op:       "JUMP_IF_EXP",
		Pos:      []int{1, 18, 17},
		ExpIndex: 0,
	}

	err := NewExecutionError("variable not found", 1, instruction, "test.sql", sourceContent)

	assert.Equal(t, "variable not found", err.Message)
	assert.Equal(t, 1, err.Instruction)
	assert.Equal(t, []int{1, 18, 17}, err.Pos)
	assert.Equal(t, "test.sql", err.SourceFile)
	assert.Equal(t, "SELECT id, name /*# if include_email */, email", err.SourceLine)
}

func TestErrorReporter_ReportError(t *testing.T) {
	sourceContent := `SELECT id, name /*# if include_email */, email
FROM users 
WHERE id = /*= user_id */123`

	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Pos: []int{1, 1, 0}, Value: "SELECT id, name"},
		{Op: "JUMP_IF_EXP", Pos: []int{1, 18, 17}, ExpIndex: 0, Target: 4},
		{Op: "EMIT_LITERAL", Pos: []int{1, 42, 41}, Value: ", email"},
	}

	reporter := NewErrorReporter("test.sql", sourceContent, instructions)
	err := reporter.ReportError("condition evaluation failed", 1)

	assert.Equal(t, "condition evaluation failed", err.Message)
	assert.Equal(t, 1, err.Instruction)
	assert.Equal(t, []int{1, 18, 17}, err.Pos)
	assert.Equal(t, "test.sql", err.SourceFile)
	assert.Equal(t, "SELECT id, name /*# if include_email */, email", err.SourceLine)
}

func TestValidateInstructionPositions(t *testing.T) {
	sourceContent := `SELECT id, name
FROM users`

	tests := []struct {
		name         string
		instructions []Instruction
		expectErrors int
	}{
		{
			name: "Valid positions",
			instructions: []Instruction{
				{Op: "EMIT_LITERAL", Pos: []int{1, 1, 0}, Value: "SELECT id, name"},
				{Op: "EMIT_LITERAL", Pos: []int{2, 1, 16}, Value: "FROM users"},
			},
			expectErrors: 0,
		},
		{
			name: "Invalid position format",
			instructions: []Instruction{
				{Op: "EMIT_LITERAL", Pos: []int{1, 1}, Value: "SELECT id, name"}, // Missing offset
			},
			expectErrors: 1,
		},
		{
			name: "Invalid line number",
			instructions: []Instruction{
				{Op: "EMIT_LITERAL", Pos: []int{5, 1, 0}, Value: "SELECT id, name"},
			},
			expectErrors: 1,
		},
		{
			name: "Invalid column number",
			instructions: []Instruction{
				{Op: "EMIT_LITERAL", Pos: []int{1, 50, 0}, Value: "SELECT id, name"},
			},
			expectErrors: 1,
		},
		{
			name: "Invalid offset",
			instructions: []Instruction{
				{Op: "EMIT_LITERAL", Pos: []int{1, 1, 100}, Value: "SELECT id, name"},
			},
			expectErrors: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errors := ValidateInstructionPositions(test.instructions, sourceContent)
			assert.Equal(t, test.expectErrors, len(errors))
		})
	}
}

func TestErrorReporter_InvalidInstructionIndex(t *testing.T) {
	instructions := []Instruction{
		{Op: "EMIT_LITERAL", Pos: []int{1, 1, 0}, Value: "SELECT"},
	}

	reporter := NewErrorReporter("test.sql", "SELECT", instructions)

	// Test with invalid instruction index
	err := reporter.ReportError("error message", 10)

	assert.Equal(t, "error message", err.Message)
	assert.Equal(t, 10, err.Instruction)
	assert.Equal(t, []int{0, 0, 0}, err.Pos) // Default position for invalid index
}

func TestExecutionError_DetailedErrorWithoutSourceLine(t *testing.T) {
	err := &ExecutionError{
		Message:    "some error",
		Pos:        []int{1, 5, 4},
		SourceFile: "test.sql",
		// No SourceLine
	}

	detailed := err.DetailedError()

	// Should contain the basic error message
	assert.True(t, strings.Contains(detailed, "test.sql:1:5: some error"))

	// Should not contain source context since SourceLine is empty
	assert.False(t, strings.Contains(detailed, "^"))
}
