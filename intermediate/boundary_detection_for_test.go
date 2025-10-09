package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestIsInConditionalBlock_ForLoop tests the FOR loop detection in isInConditionalBlock
func TestIsInConditionalBlock_ForLoop(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		index    int
		expected bool
	}{
		{
			name: "Inside FOR loop",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
			},
			index:    1,
			expected: true,
		},
		{
			name: "Outside FOR loop - after end",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
			},
			index:    2,
			expected: false,
		},
		{
			name: "Nested FOR and IF",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "if"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
			},
			index:    2,
			expected: true,
		},
		{
			name: "Inside IF block only",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "if"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
			},
			index:    1,
			expected: true,
		},
		{
			name: "Multiple nested FOR loops",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
			},
			index:    2,
			expected: true,
		},
		{
			name: "After first FOR loop ends, inside second",
			tokens: []tokenizer.Token{
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "for"}},
				{Type: tokenizer.IDENTIFIER, Value: "test"},
				{Type: tokenizer.BLOCK_COMMENT, Directive: &tokenizer.Directive{Type: "end"}},
			},
			index:    3,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInConditionalBlock(tt.tokens, tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBoundaryDetectionInForLoop tests boundary detection within FOR loops
func TestBoundaryDetectionInForLoop(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		description string
		checkFunc   func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "FOR loop with trailing comma in INSERT VALUES",
			sql: `/*#
function_name: deliverNotification
parameters:
  notification_id: int
  user_ids: [string]
*/
INSERT INTO inbox (
    notification_id,
    user_id
) VALUES
/*# for user_id : user_ids */
    (
        /*= notification_id */1,
        /*= user_id */'EMP001'
    ),
/*# end */
RETURNING notification_id, user_id`,
			description: "Should detect trailing comma in FOR loop",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// Should find EMIT_UNLESS_BOUNDARY for the comma after the closing paren
				found := false

				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && strings.Contains(instr.Value, ",") {
						found = true
						break
					}
				}

				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for trailing comma in FOR loop")
			},
		},
		{
			name: "FOR loop with leading comma",
			sql: `/*#
function_name: selectFields
parameters:
  field_names: [string]
*/
SELECT id
/*# for field_name : field_names */
, /*= field_name */'name'
/*# end */
FROM users`,
			description: "Should detect leading comma in FOR loop",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// Should find EMIT_UNLESS_BOUNDARY for the leading comma
				found := false

				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && instr.Value == "," {
						found = true
						break
					}
				}

				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for leading comma in FOR loop")
			},
		},
		{
			name: "FOR loop with AND in WHERE clause",
			sql: `/*#
function_name: filterUsers
parameters:
  conditions: [string]
*/
SELECT id FROM users WHERE 1=1
/*# for condition : conditions */
AND /*= condition */'name = "test"'
/*# end */`,
			description: "Should detect AND keyword in FOR loop",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				t.Helper()
				// Should find EMIT_UNLESS_BOUNDARY for the AND
				found := false

				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && strings.Contains(instr.Value, "AND") {
						found = true
						break
					}
				}

				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for AND in FOR loop")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			format, err := GenerateFromSQL(reader, nil, "test.sql", "", nil, nil)
			assert.NoError(t, err)

			tt.checkFunc(t, format.Instructions)
		})
	}
}
