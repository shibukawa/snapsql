package intermediate

import (
	"testing"
)

func TestNormalizeTimeFunctions(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		input    []Instruction
		expected []Instruction
	}{
		{
			name:    "MySQL: CURRENT_TIMESTAMP to NOW()",
			dialect: "mysql",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_TIMESTAMP FROM users WHERE created_at > CURRENT_TIMESTAMP"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW() FROM users WHERE created_at > NOW()"},
			},
		},
		{
			name:    "PostgreSQL: NOW() to CURRENT_TIMESTAMP",
			dialect: "postgres",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW() FROM users WHERE created_at > NOW()"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_TIMESTAMP FROM users WHERE created_at > CURRENT_TIMESTAMP"},
			},
		},
		{
			name:    "SQLite: NOW() to CURRENT_TIMESTAMP",
			dialect: "sqlite",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW() FROM users WHERE created_at > NOW()"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_TIMESTAMP FROM users WHERE created_at > CURRENT_TIMESTAMP"},
			},
		},
		{
			name:    "Mixed functions in MySQL",
			dialect: "mysql",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW(), CURRENT_TIMESTAMP, CAST(NOW() AS CHAR) FROM users"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW(), NOW(), CAST(NOW() AS CHAR) FROM users"},
			},
		},
		{
			name:    "Mixed functions in PostgreSQL",
			dialect: "postgresql",
			input: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT NOW(), CURRENT_TIMESTAMP, CAST(NOW() AS TEXT) FROM users"},
			},
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CAST(CURRENT_TIMESTAMP AS TEXT) FROM users"},
			},
		},
		{
			name:    "Non-static instructions unchanged",
			dialect: "mysql",
			input: []Instruction{
				{Op: OpEmitEval, Value: "SELECT NOW() FROM users", ExprIndex: intPtr(0)},
			},
			expected: []Instruction{
				{Op: OpEmitEval, Value: "SELECT NOW() FROM users", ExprIndex: intPtr(0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instructions := make([]Instruction, len(tt.input))
			copy(instructions, tt.input)

			normalizeTimeFunctions(instructions, tt.dialect)

			if len(instructions) != len(tt.expected) {
				t.Fatalf("Expected %d instructions, got %d", len(tt.expected), len(instructions))
			}

			for i, expected := range tt.expected {
				if instructions[i].Op != expected.Op {
					t.Errorf("Instruction %d: expected Op %q, got %q", i, expected.Op, instructions[i].Op)
				}

				if instructions[i].Value != expected.Value {
					t.Errorf("Instruction %d: expected Value %q, got %q", i, expected.Value, instructions[i].Value)
				}

				if instructions[i].ExprIndex != expected.ExprIndex {
					// Compare pointer values, not addresses
					if (instructions[i].ExprIndex == nil) != (expected.ExprIndex == nil) {
						t.Errorf("Instruction %d: ExprIndex nil mismatch", i)
					} else if instructions[i].ExprIndex != nil && expected.ExprIndex != nil && *instructions[i].ExprIndex != *expected.ExprIndex {
						t.Errorf("Instruction %d: expected ExprIndex %d, got %d", i, *expected.ExprIndex, *instructions[i].ExprIndex)
					}
				}
			}
		})
	}
}
