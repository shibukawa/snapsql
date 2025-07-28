package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser"
)

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestDialectAutoDetection(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []Instruction
	}{
		{
			name: "CAST syntax detection",
			sql: `/*#
function_name: test
parameters:
  user_id: int
*/
SELECT id, CAST(age AS INTEGER) as age_int FROM users WHERE id = /*= user_id */123`,
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, "},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "CAST(age AS INTEGER)",
					Dialects:    []string{"mysql", "sqlite"},
				},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "(age)::INTEGER",
					Dialects:    []string{"postgresql"},
				},
				{Op: OpEmitStatic, Value: " as age_int FROM users WHERE id ="},
				{Op: OpEmitEval, ExprIndex: intPtr(0)},
			},
		},
		{
			name: "NOW() function detection",
			sql: `/*#
function_name: test
*/
SELECT id, NOW() as current_time FROM users`,
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, "},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "NOW()",
					Dialects:    []string{"mysql"},
				},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "CURRENT_TIMESTAMP",
					Dialects:    []string{"postgresql", "sqlite"},
				},
				{Op: OpEmitStatic, Value: " as current_time FROM users"},
			},
		},
		{
			name: "TRUE literal detection",
			sql: `/*#
function_name: test
*/
SELECT id, TRUE as is_active FROM users`,
			expected: []Instruction{
				{Op: OpEmitStatic, Value: "SELECT id, "},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "TRUE",
					Dialects:    []string{"postgresql"},
				},
				{
					Op:          OpEmitIfDialect,
					SqlFragment: "1",
					Dialects:    []string{"mysql", "sqlite"},
				},
				{Op: OpEmitStatic, Value: " as is_active FROM users"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL
			reader := strings.NewReader(tt.sql)
			stmt, funcDef, err := parser.ParseSQLFile(reader, nil, "", "")
			assert.NoError(t, err)

			// Generate intermediate format (create new reader since the previous one was consumed)
			reader2 := strings.NewReader(tt.sql)
			result, err := GenerateFromSQL(reader2, nil, "", "", nil, nil)
			assert.NoError(t, err)

			// Check that dialect instructions are generated
			hasDialectInstructions := false
			for _, instruction := range result.Instructions {
				if instruction.Op == OpEmitIfDialect {
					hasDialectInstructions = true
					t.Logf("Found dialect instruction: %s for dialects %v",
						instruction.SqlFragment, instruction.Dialects)
				}
			}

			if !hasDialectInstructions {
				t.Log("No dialect instructions found. All instructions:")
				for i, instruction := range result.Instructions {
					t.Logf("  %d: %s - %s", i, instruction.Op, instruction.Value)
				}
			}

			// For now, just verify that the parsing works
			// TODO: Add more specific assertions once the integration is complete
			_ = stmt
			_ = funcDef
		})
	}
}

func TestDetectDialectPatterns(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int // Number of conversions expected
	}{
		{
			name:     "CAST syntax",
			sql:      "/*# function_name: test */\nSELECT CAST(age AS INTEGER) FROM users",
			expected: 1,
		},
		{
			name:     "NOW function",
			sql:      "/*# function_name: test */\nSELECT NOW() FROM users",
			expected: 1,
		},
		{
			name:     "TRUE literal",
			sql:      "/*# function_name: test */\nSELECT TRUE FROM users",
			expected: 1,
		},
		{
			name:     "Multiple patterns",
			sql:      "/*# function_name: test */\nSELECT CAST(age AS INTEGER), NOW(), TRUE FROM users",
			expected: 3,
		},
		{
			name:     "PostgreSQL cast syntax",
			sql:      "/*# function_name: test */\nSELECT price::DECIMAL(10,2) FROM users",
			expected: 1,
		},
		{
			name:     "PostgreSQL cast with parentheses",
			sql:      "/*# function_name: test */\nSELECT (price + tax)::DECIMAL(10,2) FROM users",
			expected: 1,
		},
		{
			name:     "Complex PostgreSQL cast",
			sql:      "/*# function_name: test */\nSELECT (CASE WHEN active THEN price ELSE 0 END)::DECIMAL(10,2) FROM users",
			expected: 1,
		},
		{
			name:     "CURRENT_TIMESTAMP",
			sql:      "/*# function_name: test */\nSELECT CURRENT_TIMESTAMP FROM users",
			expected: 1,
		},
		{
			name:     "Mixed CAST and PostgreSQL cast",
			sql:      "/*# function_name: test */\nSELECT CAST(age AS INTEGER), price::DECIMAL(10,2) FROM users",
			expected: 2,
		},
		{
			name:     "CONCAT function",
			sql:      "/*# function_name: test */\nSELECT CONCAT(first_name, ' ', last_name) FROM users",
			expected: 1,
		},
		{
			name:     "Nested dialect in CONCAT",
			sql:      "/*# function_name: test */\nSELECT CONCAT('User: ', CAST(id AS TEXT)) FROM users",
			expected: 1, // Only CONCAT is detected, CAST inside is part of the arguments
		},
		{
			name:     "RAND function",
			sql:      "/*# function_name: test */\nSELECT RAND() FROM users",
			expected: 1,
		},
		{
			name:     "RANDOM function",
			sql:      "/*# function_name: test */\nSELECT RANDOM() FROM users",
			expected: 1,
		},
		{
			name:     "No patterns",
			sql:      "/*# function_name: test */\nSELECT id, name FROM users",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL to get tokens
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "")
			assert.NoError(t, err)

			// Extract tokens
			tokens := extractTokensFromStatement(stmt)

			// Detect patterns
			conversions := detectDialectPatterns(tokens)

			assert.Equal(t, tt.expected, len(conversions),
				"Expected %d conversions, got %d", tt.expected, len(conversions))

			// Log detected conversions for debugging
			for i, conversion := range conversions {
				originalText := buildSQLFromTokens(conversion.OriginalTokens)
				t.Logf("Conversion %d: %s -> %d variants",
					i, originalText, len(conversion.Variants))
				for j, variant := range conversion.Variants {
					t.Logf("  Variant %d: %s for %v",
						j, variant.SqlFragment, variant.Dialects)
				}
			}
		})
	}
}
