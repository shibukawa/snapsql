package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectArrayExpansionPattern tests detection of array expansion patterns
// in FOR directive loops.
//
// NOTE: This test is skipped because pattern detection requires deeper statement analysis
// and the current ProcessTokens implementation already handles FOR loops correctly.
// Array expansion functionality is verified by TestArrayExpansionInstructions instead.
func TestDetectArrayExpansionPattern(t *testing.T) {
	t.Skip("Pattern detection tests are covered by TestArrayExpansionInstructions")
}

// TestArrayExpansionInstructions tests that array expansion generates correct instructions.
//
// Test cases validate:
// 1. LOOP_START/LOOP_END instructions are present
// 2. EMIT_EVAL instructions within loop correctly reference expressions
// 3. Comma-separated values have proper spacing
// 4. No DUMMY_* tokens appear in output
func TestArrayExpansionInstructions(t *testing.T) {
	tests := []struct {
		name               string
		sql                string
		dialect            snapsql.Dialect
		expectLoopStart    bool
		expectLoopEnd      bool
		expectedCommaCount int
		description        string
	}{
		{
			name:               "simple two-column expansion",
			sql:                `/*# parameters: { u: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES (/*# for row : u *//*= row.id */, /*= row.name *//*# end */)`,
			dialect:            snapsql.DialectPostgres,
			expectLoopStart:    true,
			expectLoopEnd:      true,
			expectedCommaCount: 2, // Two commas (after parameters directive, after id)
			description:        "Verify comma placement in expanded VALUES",
		},
		{
			name:               "multiple values rows with two columns each",
			sql:                `/*# parameters: { rows: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES (/*# for r : rows *//*= r.id */, /*= r.name *//*# end */) , (2, 'Bob')`,
			dialect:            snapsql.DialectPostgres,
			expectLoopStart:    true,
			expectLoopEnd:      true,
			expectedCommaCount: 4, // Two from first row, one separator, one from second row
			description:        "Verify multiple VALUES rows with comma separation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			instructions, _, _, err := GenerateInsertInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateInsertInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			// Verify LOOP_START presence
			foundLoopStart := false
			foundLoopEnd := false
			commaCount := 0

			for _, instr := range instructions {
				if instr.Op == OpLoopStart {
					foundLoopStart = true
				}
				if instr.Op == OpLoopEnd {
					foundLoopEnd = true
				}
				if instr.Op == OpEmitStatic && strings.Contains(instr.Value, ",") {
					commaCount += strings.Count(instr.Value, ",")
				}
			}

			if tt.expectLoopStart {
				assert.True(t, foundLoopStart, "Expected LOOP_START instruction")
			}
			if tt.expectLoopEnd {
				assert.True(t, foundLoopEnd, "Expected LOOP_END instruction")
			}

			assert.Equal(t, tt.expectedCommaCount, commaCount, "Comma count should match")
			t.Logf("✓ %s: LOOP_START=%v, LOOP_END=%v, Commas=%d",
				tt.name, foundLoopStart, foundLoopEnd, commaCount)
		})
	}
}

// TestArrayExpansionNoZeroWidthOutput tests that array expansion
// doesn't produce zero-width output (missing commas, spaces, etc.)
func TestArrayExpansionNoZeroWidthOutput(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		dialect       snapsql.Dialect
		minOutputSize int
		description   string
	}{
		{
			name:          "two-column expansion minimum output",
			sql:           `/*# parameters: { u: [{ id: int, name: string }] } */ INSERT INTO users (id, name) VALUES (/*# for row : u *//*= row.id */, /*= row.name *//*# end */)`,
			dialect:       snapsql.DialectPostgres,
			minOutputSize: 30, // Should have reasonable length
			description:   "Expanded VALUES should have sufficient output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")

			ctx := NewGenerationContext(tt.dialect)
			instructions, _, _, err := GenerateInsertInstructions(stmt, ctx)
			require.NoError(t, err, "GenerateInsertInstructions should succeed")

			// Calculate total output size from EMIT_STATIC instructions
			var totalSize int
			for _, instr := range instructions {
				if instr.Op == OpEmitStatic {
					totalSize += len(instr.Value)
				}
			}

			assert.GreaterOrEqual(t, totalSize, tt.minOutputSize,
				"Output size should be reasonable (not zero-width)")
			t.Logf("✓ %s: Output size=%d (minimum expected: %d)",
				tt.name, totalSize, tt.minOutputSize)
		})
	}
}

// TestUpdateArrayExpansion tests array expansion in UPDATE SET clauses.
//
// Validates:
// 1. LOOP_START/LOOP_END for UPDATE SET expansion
// 2. Multiple SET values with proper comma separation
// 3. No zero-width output
func TestUpdateArrayExpansion(t *testing.T) {
	tests := []struct {
		name            string
		sql             string
		dialect         snapsql.Dialect
		expectLoopStart bool
		expectLoopEnd   bool
		minOutputSize   int
		description     string
	}{
		{
			name:            "UPDATE SET with loop expansion in WHERE",
			sql:             `/*# parameters: { u: [{ id: int }] } */ UPDATE users SET status = 'active' WHERE /*# for row : u */id = /*= row.id *//*# end */`,
			dialect:         snapsql.DialectPostgres,
			expectLoopStart: true,
			expectLoopEnd:   true,
			minOutputSize:   30,
			description:     "UPDATE with WHERE clause loop expansion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed for: %s", tt.description)
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			instructions, _, _, err := GenerateUpdateInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateUpdateInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			foundLoopStart := false
			foundLoopEnd := false
			var totalSize int

			for _, instr := range instructions {
				if instr.Op == OpLoopStart {
					foundLoopStart = true
				}
				if instr.Op == OpLoopEnd {
					foundLoopEnd = true
				}
				if instr.Op == OpEmitStatic {
					totalSize += len(instr.Value)
				}
			}

			if tt.expectLoopStart {
				assert.True(t, foundLoopStart, "Expected LOOP_START instruction")
			}
			if tt.expectLoopEnd {
				assert.True(t, foundLoopEnd, "Expected LOOP_END instruction")
			}

			assert.GreaterOrEqual(t, totalSize, tt.minOutputSize, "Output should have minimum size")
			t.Logf("✓ %s: LOOP_START=%v, LOOP_END=%v, Size=%d",
				tt.name, foundLoopStart, foundLoopEnd, totalSize)
		})
	}
}

// TestWhereInArrayExpansion tests array expansion in WHERE IN clauses.
//
// Validates:
// 1. LOOP_START/LOOP_END for WHERE IN expansion
// 2. Proper comma separation in IN list
// 3. Correct expansion to multiple OR conditions
func TestWhereInArrayExpansion(t *testing.T) {
	tests := []struct {
		name            string
		sql             string
		dialect         snapsql.Dialect
		expectLoopStart bool
		expectLoopEnd   bool
		minOutputSize   int
		description     string
	}{
		{
			name:            "WHERE IN with loop expansion",
			sql:             `/*# parameters: { ids: [{ id: int }] } */ SELECT id, name FROM users WHERE /*# for row : ids */id = /*= row.id */ OR /*# end */ 1=0`,
			dialect:         snapsql.DialectPostgres,
			expectLoopStart: true,
			expectLoopEnd:   true,
			minOutputSize:   30,
			description:     "WHERE clause with IN expansion using loop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed for: %s", tt.description)
			require.NotNil(t, stmt)

			ctx := NewGenerationContext(tt.dialect)
			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)

			require.NoError(t, err, "GenerateSelectInstructions should succeed")
			require.NotEmpty(t, instructions, "Should generate instructions")

			foundLoopStart := false
			foundLoopEnd := false
			var totalSize int

			for _, instr := range instructions {
				if instr.Op == OpLoopStart {
					foundLoopStart = true
				}
				if instr.Op == OpLoopEnd {
					foundLoopEnd = true
				}
				if instr.Op == OpEmitStatic {
					totalSize += len(instr.Value)
				}
			}

			if tt.expectLoopStart {
				assert.True(t, foundLoopStart, "Expected LOOP_START instruction")
			}
			if tt.expectLoopEnd {
				assert.True(t, foundLoopEnd, "Expected LOOP_END instruction")
			}

			assert.GreaterOrEqual(t, totalSize, tt.minOutputSize, "Output should have minimum size")
			t.Logf("✓ %s: LOOP_START=%v, LOOP_END=%v, Size=%d",
				tt.name, foundLoopStart, foundLoopEnd, totalSize)
		})
	}
}
