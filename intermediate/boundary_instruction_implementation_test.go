package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

// TestBoundaryInstructionsImplementation tests the implementation of EMIT_UNLESS_BOUNDARY and BOUNDARY instructions
func TestBoundaryInstructionsImplementation(t *testing.T) {

	tests := []struct {
		name        string
		sql         string
		expectedOps []string
	}{
		{
			name: "Simple UPDATE with conditional comma - should use EMIT_UNLESS_BOUNDARY",
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
			name: "Simple SELECT with conditional field - should use EMIT_UNLESS_BOUNDARY",
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
				"EMIT_STATIC",          // " LIMIT "
				"END",                  // end LIMIT block
				"EMIT_STATIC",          // " OFFSET "
				"END",                  // end OFFSET block
			},
		},
		{
			name: "UPDATE with trailing comma style (JSON-like) - should use EMIT_UNLESS_BOUNDARY",
			sql: `/*#
function_name: updateUserTrailing
parameters:
  name: string
  email: string
  phone: string
*/
UPDATE users SET 
    name = /*= name */'John',
    /*# if email != "" */
    email = /*= email */'john@example.com',
    /*# end */
    /*# if phone != "" */
    phone = /*= phone */'123-456-7890'
    /*# end */
WHERE id = 1`,
			expectedOps: []string{
				"EMIT_STATIC",          // "UPDATE users SET name ="
				"EMIT_EVAL",            // name
				"EMIT_STATIC",          // ","
				"IF",                   // if email != ""
				"EMIT_STATIC",          // "email ="
				"EMIT_EVAL",            // email
				"EMIT_UNLESS_BOUNDARY", // ","
				"END",                  // end if
				"IF",                   // if phone != ""
				"EMIT_STATIC",          // "phone ="
				"EMIT_EVAL",            // phone
				"END",                  // end if
				"BOUNDARY",             // boundary before WHERE
				"EMIT_STATIC",          // "WHERE id = 1"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			format, err := GenerateFromSQL(reader, nil, "test.sql", "", nil, nil)
			assert.NoError(t, err)

			// Extract operation names from instructions (excluding system-generated ones for simplicity)
			actualOps := make([]string, 0)
			for _, instruction := range format.Instructions {
				// Skip system-generated instructions for this test
				if !strings.Contains(instruction.Op, "SYSTEM") &&
					!strings.Contains(instruction.Op, "IF_SYSTEM") {
					actualOps = append(actualOps, instruction.Op)
				}
			}

			// Compare operation sequences
			assert.Equal(t, tt.expectedOps, actualOps, "Operation sequence should match expected")
		})
	}
}

// TestBoundaryInstructionDetection tests the detection logic for when to use EMIT_UNLESS_BOUNDARY
func TestBoundaryInstructionDetection(t *testing.T) {

	tests := []struct {
		name        string
		sql         string
		description string
		checkFunc   func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "Detect conditional comma in UPDATE SET clause",
			sql: `/*#
function_name: test
parameters:
  name: string
  email: string
*/
UPDATE users SET name = /*= name */'John'
/*# if email != "" */
, email = /*= email */'john@example.com'
/*# end */
WHERE id = 1`,
			description: "Should detect comma at start of conditional block in SET clause",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Should find EMIT_UNLESS_BOUNDARY for the comma
				found := false
				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && strings.Contains(instr.Value, ",") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for comma")
			},
		},
		{
			name: "Detect conditional AND in WHERE clause",
			sql: `/*#
function_name: test
parameters:
  name: string
*/
SELECT id FROM users WHERE 1=1
/*# if name != "" */
AND name = /*= name */'John'
/*# end */`,
			description: "Should detect AND at start of conditional block in WHERE clause",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Should find EMIT_UNLESS_BOUNDARY for the AND
				found := false
				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && strings.Contains(instr.Value, "AND") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for AND")
			},
		},
		{
			name: "Detect conditional OR in WHERE clause",
			sql: `/*#
function_name: test
parameters:
  name: string
*/
SELECT id FROM users WHERE (1=0
/*# if name != "" */
OR name = /*= name */'John'
/*# end */
)`,
			description: "Should detect OR at start of conditional block in WHERE clause",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Should find EMIT_UNLESS_BOUNDARY for the OR
				found := false
				for _, instr := range instructions {
					if instr.Op == "EMIT_UNLESS_BOUNDARY" && strings.Contains(instr.Value, "OR") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have EMIT_UNLESS_BOUNDARY instruction for OR")
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

// TestBoundaryPlacement tests the placement of BOUNDARY instructions
func TestBoundaryPlacement(t *testing.T) {

	tests := []struct {
		name        string
		sql         string
		description string
		checkFunc   func(t *testing.T, instructions []Instruction)
	}{
		{
			name: "BOUNDARY before WHERE clause",
			sql: `/*#
function_name: test
parameters:
  include_email: bool
*/
UPDATE users SET name = 'John'
/*# if include_email */
, email = 'john@example.com'
/*# end */
WHERE id = 1`,
			description: "Should place BOUNDARY before WHERE clause",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Find WHERE clause and check if BOUNDARY is placed before it
				for i, instr := range instructions {
					if instr.Op == "EMIT_STATIC" && strings.Contains(instr.Value, "WHERE") {
						// Check if there's a BOUNDARY instruction before this
						foundBoundary := false
						for j := i - 1; j >= 0; j-- {
							if instructions[j].Op == "BOUNDARY" {
								foundBoundary = true
								break
							}
							if instructions[j].Op == "IF" {
								break // Stop searching when we hit the IF block
							}
						}
						assert.True(t, foundBoundary, "Should have BOUNDARY before WHERE clause")
						return
					}
				}
				t.Error("WHERE clause not found in instructions")
			},
		},
		{
			name: "BOUNDARY before FROM clause",
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
			description: "Should place BOUNDARY before FROM clause",
			checkFunc: func(t *testing.T, instructions []Instruction) {
				// Find FROM clause and check if BOUNDARY is placed before it
				for i, instr := range instructions {
					if instr.Op == "EMIT_STATIC" && strings.Contains(instr.Value, "FROM") {
						// Check if there's a BOUNDARY instruction before this
						foundBoundary := false
						for j := i - 1; j >= 0; j-- {
							if instructions[j].Op == "BOUNDARY" {
								foundBoundary = true
								break
							}
							if instructions[j].Op == "IF" {
								break // Stop searching when we hit the IF block
							}
						}
						assert.True(t, foundBoundary, "Should have BOUNDARY before FROM clause")
						return
					}
				}
				t.Error("FROM clause not found in instructions")
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
