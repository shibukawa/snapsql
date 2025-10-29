package codegenerator

import (
	"bytes"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/require"
)

// TestWhereInLiteralValues tests WHERE IN clause with literal values
// This test examines how the parser represents (1, 2, 3) in RawTokens
func TestWhereInLiteralValues(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		description string
	}{
		{
			name:        "WHERE_IN_literal_values",
			sql:         `SELECT id FROM users WHERE id IN (1, 2, 3)`,
			description: "Simple WHERE IN with literal values",
		},
		{
			name:        "WHERE_IN_with_directive",
			sql:         `/*# parameters: { values: [int] } */ SELECT id FROM users WHERE id IN /*= values */(1, 2, 3)`,
			description: "WHERE IN with directive - directive should control the list",
		},
		{
			name:        "WHERE_IN_directive_before_paren",
			sql:         `/*# parameters: { values: [int] } */ SELECT id FROM users WHERE id IN /*= values */( 1, 2, 3 )`,
			description: "WHERE IN with directive and spaces inside parens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse with snapsql parser
			reader := bytes.NewReader([]byte(tt.sql))
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			// Cast to SelectStatement
			selectStmt, ok := stmt.(*parser.SelectStatement)
			require.True(t, ok, "expected SelectStatement")

			// Generate instructions with CEL environment for "values" variable
			ctx := NewGenerationContext(snapsql.DialectPostgres)

			// For directive cases, create a minimal FunctionDefinition with parameter info
			// and set it in the context
			if tt.name != "WHERE_IN_literal_values" {
				funcDef := &parser.FunctionDefinition{
					ParameterOrder: []string{"values"},
					OriginalParameters: map[string]interface{}{
						"values": "int[]",
					},
				}
				ctx.SetFunctionDefinition(funcDef)
			}

			instructions, _, _, err := GenerateSelectInstructions(selectStmt, ctx)
			require.NoError(t, err, "GenerateSelectInstructions should succeed")
			require.NotEmpty(t, instructions, "expected instructions")

			// Print token analysis for debugging
			t.Logf("=== Token Analysis for %s ===", tt.name)

			if selectStmt.Where != nil {
				whereTokens := selectStmt.Where.RawTokens()
				for idx, token := range whereTokens {
					t.Logf("Token[%d]: Type=%s, Value=%q, HasDirective=%v, Pos=%s",
						idx, token.Type, token.Value, token.Directive != nil, token.Position.String())

					if token.Directive != nil {
						t.Logf("  Directive: Type=%s, Condition=%q", token.Directive.Type, token.Directive.Condition)
					}
				}
			}

			// Print instruction analysis
			t.Logf("=== Instructions ===")

			for idx, instr := range instructions {
				t.Logf("Instr[%d]: Op=%s, Value=%q, ExprIndex=%v", idx, instr.Op, instr.Value, instr.ExprIndex)
			}

			// Detailed LOOP instruction check
			t.Logf("=== Checking for LOOP instructions ===")

			hasLoopStart := false
			hasLoopEnd := false

			for _, instr := range instructions {
				if instr.Op == "LOOP_START" {
					hasLoopStart = true

					t.Logf("Found LOOP_START")
				}

				if instr.Op == "LOOP_END" {
					hasLoopEnd = true

					t.Logf("Found LOOP_END")
				}
			}

			if !hasLoopStart && tt.name != "WHERE_IN_literal_values" {
				t.Logf("WARNING: No LOOP_START found for directive case")
			}

			if !hasLoopEnd && tt.name != "WHERE_IN_literal_values" {
				t.Logf("WARNING: No LOOP_END found for directive case")
			}

			// Debug: Check if we have a "values" variable in context
			if len(ctx.CELEnvironments) > 0 {
				env := ctx.CELEnvironments[0]

				t.Logf("=== CEL Environment ===")
				t.Logf("CEL Variables: %+v", env.AdditionalVariables)
			}
		})
	}
}

// TestWhereInDummyTokens examines the structure of (1, 2, 3) in parser output
func TestWhereInDummyTokens(t *testing.T) {
	sql := `/*# parameters: { values: [int] } */ SELECT id FROM users WHERE id IN /*= values */(1, 2, 3)`

	reader := bytes.NewReader([]byte(sql))
	stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
	require.NoError(t, err, "ParseSQLFile should succeed")
	require.NotNil(t, stmt)

	selectStmt, ok := stmt.(*parser.SelectStatement)
	require.True(t, ok, "expected SelectStatement")

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}

	whereTokens := selectStmt.Where.RawTokens()

	t.Logf("=== WHERE Clause RawTokens Analysis ===")
	t.Logf("Total tokens in WHERE: %d", len(whereTokens))

	for idx, token := range whereTokens {
		typeStr := token.Type.String()
		t.Logf("[%2d] Type=%-20s Value=%-15q Pos=%s", idx, typeStr, token.Value, token.Position.String())

		if token.Directive != nil {
			t.Logf("     -> Directive: Type=%s, Condition=%q", token.Directive.Type, token.Directive.Condition)
		}
	}

	// Look for parenthesis and comma patterns
	t.Logf("\n=== Searching for parenthesis and comma patterns ===")

	for i := range whereTokens {
		token := whereTokens[i]
		if token.Value == "(" || token.Value == ")" || token.Value == "," {
			t.Logf("Found %q at index %d", token.Value, i)
		}
	}
}
