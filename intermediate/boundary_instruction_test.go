package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestBoundaryInstructions(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectedOps []string
	}{
		{
			name: "UPDATE with conditional comma removal - current behavior",
			sql: `/*#
function_name: updateUser
parameters:
  name: string
  email: string
*/
UPDATE users SET 
    name = /*= name */'John'
    /*# if email != "" */
    , email = /*= email */'john@example.com'
    /*# end */
WHERE id = 1`,
			expectedOps: []string{
				"EMIT_STATIC",          // "UPDATE users SET name ="
				"EMIT_EVAL",            // name
				"IF",                   // if email != ""
				"EMIT_UNLESS_BOUNDARY", // ", email ="
				"EMIT_STATIC",          // "email ="
				"EMIT_EVAL",            // email
				"END",                  // end if
				"BOUNDARY",             // boundary before WHERE
				"EMIT_STATIC",          // "WHERE id = 1"
			},
		},
		{
			name: "SELECT with conditional field comma removal - current behavior",
			sql: `/*#
function_name: getUser
parameters:
  include_email: bool
*/
SELECT 
    id,
    name
    /*# if include_email */
    , email
    /*# end */
FROM users`,
			expectedOps: []string{
				"EMIT_STATIC",          // "SELECT id, name"
				"IF",                   // if include_email
				"EMIT_UNLESS_BOUNDARY", // ", email"
				"EMIT_STATIC",          // "email"
				"END",                  // end if
				"BOUNDARY",             // boundary before FROM
				"EMIT_STATIC",          // "FROM users"
				"IF_SYSTEM_LIMIT",      // システム生成のLIMIT処理
				"EMIT_STATIC",
				"EMIT_SYSTEM_LIMIT",
				"END",
				"IF_SYSTEM_OFFSET", // システム生成のOFFSET処理
				"EMIT_STATIC",
				"EMIT_SYSTEM_OFFSET",
				"END",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			format, err := GenerateFromSQL(reader, nil, "test.sql", "", nil, nil)
			assert.NoError(t, err)

			// Extract operation names from instructions
			actualOps := make([]string, len(format.Instructions))
			for i, instruction := range format.Instructions {
				actualOps[i] = instruction.Op
			}

			// Compare operation sequences
			assert.Equal(t, tt.expectedOps, actualOps, "Operation sequence should match expected")
		})
	}
}

func TestBoundaryInstructionGeneration(t *testing.T) {
	// Test specific boundary instruction generation scenarios
	tests := []struct {
		name      string
		sql       string
		checkFunc func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "EMIT_UNLESS_BOUNDARY should contain delimiter",
			sql: `/*#
function_name: test
parameters:
  include_field: bool
*/
SELECT id
/*# if include_field */
, name
/*# end */
FROM users`,
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Find EMIT_UNLESS_BOUNDARY instruction
				var boundaryInstr *Instruction
				for i := range instructions {
					if instructions[i].Op == "EMIT_UNLESS_BOUNDARY" {
						boundaryInstr = &instructions[i]
						break
					}
				}
				assert.True(t, boundaryInstr != nil, "Should have EMIT_UNLESS_BOUNDARY instruction")
				assert.Equal(t, ",", boundaryInstr.Value, "Should contain comma delimiter")
			},
		},
		{
			name: "BOUNDARY should be placed at clause boundaries",
			sql: `/*#
function_name: test
parameters:
  include_email: bool
*/
SELECT id, name
/*# if include_email */
, email
/*# end */
FROM users`,
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Count BOUNDARY instructions
				boundaryCount := 0
				for _, instr := range instructions {
					if instr.Op == "BOUNDARY" {
						boundaryCount++
					}
				}
				assert.True(t, boundaryCount > 0, "Should have at least one BOUNDARY instruction")
			},
		},
		{
			name: "Multiple delimiters should each have EMIT_UNLESS_BOUNDARY",
			sql: `/*#
function_name: test
parameters:
  name: string
  email: string
  phone: string
*/
UPDATE users SET
    id = 1
    /*# if name != "" */
    , name = /*= name */'John'
    /*# end */
    /*# if email != "" */
    , email = /*= email */'john@example.com'
    /*# end */
    /*# if phone != "" */
    , phone = /*= phone */'123-456-7890'
    /*# end */
WHERE id = 1`,
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Count EMIT_UNLESS_BOUNDARY instructions
				boundaryCount := 0
				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" {
						boundaryCount++
					}
				}
				assert.Equal(t, 3, boundaryCount, "Should have 3 EMIT_UNLESS_BOUNDARY instructions for 3 conditional commas")
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
