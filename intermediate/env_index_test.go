package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestSetEnvIndexInInstructions(t *testing.T) {
	tests := []struct {
		name               string
		envs               [][]EnvVar
		instructions       []Instruction
		expectedEnvIndices map[int]int // instruction index -> expected env_index
	}{
		{
			name: "no loops",
			envs: [][]EnvVar{},
			instructions: []Instruction{
				{Op: OpEmitStatic, Value: "test"},
			},
			expectedEnvIndices: map[int]int{},
		},
		{
			name: "single loop",
			envs: [][]EnvVar{
				{{Name: "item", Type: "any"}},
			},
			instructions: []Instruction{
				{Op: OpLoopStart, Variable: "item"},
				{Op: OpEmitStatic, Value: "test"},
				{Op: OpLoopEnd},
			},
			expectedEnvIndices: map[int]int{
				0: 1, // LOOP_START item
				2: 0, // LOOP_END (return to base environment)
			},
		},
		{
			name: "nested loops",
			envs: [][]EnvVar{
				{{Name: "category", Type: "any"}},
				{{Name: "category", Type: "any"}, {Name: "item", Type: "any"}},
			},
			instructions: []Instruction{
				{Op: OpLoopStart, Variable: "category"},
				{Op: OpEmitStatic, Value: "test1"},
				{Op: OpLoopStart, Variable: "item"},
				{Op: OpEmitStatic, Value: "test2"},
				{Op: OpLoopEnd},
				{Op: OpEmitStatic, Value: "test3"},
				{Op: OpLoopEnd},
			},
			expectedEnvIndices: map[int]int{
				0: 1, // LOOP_START category
				2: 2, // LOOP_START item
				4: 1, // LOOP_END item (return to category environment)
				6: 0, // LOOP_END category (return to base environment)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of instructions to avoid modifying the test data
			instructions := make([]Instruction, len(tt.instructions))
			copy(instructions, tt.instructions)

			setEnvIndexInInstructions(tt.envs, instructions)

			// Verify that env_index is set correctly
			for i, instruction := range instructions {
				if expectedIndex, exists := tt.expectedEnvIndices[i]; exists {
					assert.True(t, instruction.EnvIndex != nil, "EnvIndex should be set for instruction %d (%s)", i, instruction.Op)
					assert.Equal(t, expectedIndex, *instruction.EnvIndex, "Instruction %d (%s) should have env_index %d", i, instruction.Op, expectedIndex)
				} else {
					// Instructions without expected env_index should not have it set
					if instruction.Op == OpLoopStart || instruction.Op == OpLoopEnd {
						// These should always have env_index set if they have a variable
						if instruction.Variable != "" {
							assert.True(t, instruction.EnvIndex != nil, "EnvIndex should be set for instruction %d (%s) with variable", i, instruction.Op)
						}
					}
				}
			}
		})
	}
}

func TestFindVariableEnvironmentIndex(t *testing.T) {
	tests := []struct {
		name     string
		envs     [][]EnvVar
		variable string
		expected int
	}{
		{
			name:     "empty envs",
			envs:     [][]EnvVar{},
			variable: "test",
			expected: 0,
		},
		{
			name: "variable in first level",
			envs: [][]EnvVar{
				{{Name: "item", Type: "any"}},
			},
			variable: "item",
			expected: 1,
		},
		{
			name: "variable introduced in second level",
			envs: [][]EnvVar{
				{{Name: "category", Type: "any"}},
				{{Name: "category", Type: "any"}, {Name: "item", Type: "any"}},
			},
			variable: "item",
			expected: 2,
		},
		{
			name: "variable from first level",
			envs: [][]EnvVar{
				{{Name: "category", Type: "any"}},
				{{Name: "category", Type: "any"}, {Name: "item", Type: "any"}},
			},
			variable: "category",
			expected: 1,
		},
		{
			name: "variable not found",
			envs: [][]EnvVar{
				{{Name: "category", Type: "any"}},
				{{Name: "category", Type: "any"}, {Name: "item", Type: "any"}},
			},
			variable: "notfound",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findVariableEnvironmentIndex(tt.envs, tt.variable)
			assert.Equal(t, tt.expected, result)
		})
	}
}
