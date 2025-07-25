package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestDetectLimitOffsetClause(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected *LimitOffsetClauseInfo
	}{
		{
			name: "no limit or offset clause",
			sql:  "SELECT * FROM users",
			expected: &LimitOffsetClauseInfo{
				HasLimit:  false,
				HasOffset: false,
			},
		},
		{
			name: "simple limit clause",
			sql:  "SELECT * FROM users LIMIT 10",
			expected: &LimitOffsetClauseInfo{
				HasLimit:          true,
				HasLimitCondition: false,
				HasOffset:         false,
			},
		},
		{
			name: "simple offset clause",
			sql:  "SELECT * FROM users OFFSET 20",
			expected: &LimitOffsetClauseInfo{
				HasLimit:           false,
				HasOffset:          true,
				HasOffsetCondition: false,
			},
		},
		{
			name: "limit and offset clause",
			sql:  "SELECT * FROM users LIMIT 10 OFFSET 20",
			expected: &LimitOffsetClauseInfo{
				HasLimit:           true,
				HasLimitCondition:  false,
				HasOffset:          true,
				HasOffsetCondition: false,
			},
		},
		{
			name: "limit with variable",
			sql:  "SELECT * FROM users LIMIT /*= page_size */10",
			expected: &LimitOffsetClauseInfo{
				HasLimit:          true,
				HasLimitCondition: false,
				HasOffset:         false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			result := detectLimitOffsetClause(tokens)
			assert.Equal(t, tt.expected.HasLimit, result.HasLimit)
			assert.Equal(t, tt.expected.HasLimitCondition, result.HasLimitCondition)
			assert.Equal(t, tt.expected.HasOffset, result.HasOffset)
			assert.Equal(t, tt.expected.HasOffsetCondition, result.HasOffsetCondition)

			if tt.expected.HasLimit {
				assert.True(t, result.LimitTokenIndex >= 0, "LimitTokenIndex should be set")
			}
			if tt.expected.HasOffset {
				assert.True(t, result.OffsetTokenIndex >= 0, "OffsetTokenIndex should be set")
			}
		})
	}
}

func TestGenerateInstructionsWithLimitOffset(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []string // Expected instruction operations
	}{
		{
			name: "no limit or offset clause",
			sql:  "SELECT * FROM users",
			expected: []string{
				"EMIT_STATIC",     // SELECT * FROM users
				"IF_SYSTEM_LIMIT", // Add system limit if available
				"EMIT_STATIC",     // LIMIT
				"EMIT_SYSTEM_LIMIT",
				"END",
				"IF_SYSTEM_OFFSET", // Add system offset if available
				"EMIT_STATIC",      // OFFSET
				"EMIT_SYSTEM_OFFSET",
				"END",
			},
		},
		{
			name: "simple limit clause",
			sql:  "SELECT * FROM users LIMIT 10",
			expected: []string{
				"EMIT_STATIC", // SELECT * FROM users
				"EMIT_STATIC", // LIMIT
				"IF_SYSTEM_LIMIT",
				"EMIT_SYSTEM_LIMIT",
				"ELSE",
				"EMIT_STATIC", // 10
				"END",
				"IF_SYSTEM_OFFSET", // Add system offset if available
				"EMIT_STATIC",      // OFFSET
				"EMIT_SYSTEM_OFFSET",
				"END",
			},
		},
		{
			name: "limit and offset clause",
			sql:  "SELECT * FROM users LIMIT 10 OFFSET 20",
			expected: []string{
				"EMIT_STATIC", // SELECT * FROM users
				"EMIT_STATIC", // LIMIT
				"IF_SYSTEM_LIMIT",
				"EMIT_SYSTEM_LIMIT",
				"ELSE",
				"EMIT_STATIC", // 10
				"END",
				"EMIT_STATIC", // OFFSET
				"IF_SYSTEM_OFFSET",
				"EMIT_SYSTEM_OFFSET",
				"ELSE",
				"EMIT_STATIC", // 20
				"END",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			instructions := GenerateInstructions(tokens, []string{})

			// Extract operation names for comparison
			ops := make([]string, len(instructions))
			for i, instr := range instructions {
				ops[i] = instr.Op
			}

			// Check that we have the expected operations (order may vary)
			for _, expectedOp := range tt.expected {
				found := false
				for _, actualOp := range ops {
					if actualOp == expectedOp {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected operation %s not found in %v", expectedOp, ops)
			}
		})
	}
}
