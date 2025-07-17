package parserstep5

import (
	"testing"
)

func TestParseFullPipelineTokens(t *testing.T) {
	sql := `SELECT /*= user.name */'default_name' FROM users`

	stmt := parseFullPipeline(t, sql)

	t.Logf("After parsing pipeline for: %s", sql)

	// Check all clauses and their tokens
	for i, clause := range stmt.Clauses() {
		t.Logf("Clause[%d]: %s", i, clause.Type())

		tokens := clause.ContentTokens()
		t.Logf("  ContentTokens count: %d", len(tokens))
		for j, token := range tokens {
			if token.Directive != nil {
				t.Logf("  Token[%d]: %s = %q (Directive: %s)",
					j, token.Type, token.Value, token.Directive.Type)
			} else {
				t.Logf("  Token[%d]: %s = %q", j, token.Type, token.Value)
			}
		}

		// Also check raw tokens if available
		rawTokens := clause.RawTokens()
		t.Logf("  RawTokens count: %d", len(rawTokens))
		for j, token := range rawTokens {
			if token.Directive != nil {
				t.Logf("  RawToken[%d]: %s = %q (Directive: %s)",
					j, token.Type, token.Value, token.Directive.Type)
			} else {
				t.Logf("  RawToken[%d]: %s = %q", j, token.Type, token.Value)
			}
		}
	}

	// Check if any directive tokens exist
	foundDirective := false
	for _, clause := range stmt.Clauses() {
		for _, token := range clause.ContentTokens() {
			if token.Directive != nil {
				foundDirective = true
				break
			}
		}
	}

	t.Logf("Found directive in parsed statement: %v", foundDirective)
}
