package parserstep6

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Test that a boolean literal immediately after a directive comment is treated as a dummy literal
// and replaced with DUMMY_START <value> DUMMY_END, not left as a raw boolean (e.g., false).
func TestBooleanAfterDirectiveIsDummyLiteral(t *testing.T) {
	sql := `/*#
function_name: test_func
parameters:
  is_archived: bool
*/
SELECT 1 FROM tasks WHERE is_archived = /*= is_archived */false`

	// Full pipeline up to step6
	tokens, err := tokenizer.Tokenize(sql)
	assert.NoError(t, err)

	fd, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, "", "")
	assert.NoError(t, err)

	step1, err := parserstep1.Execute(tokens)
	assert.NoError(t, err)

	stmt, err := parserstep2.Execute(step1)
	assert.NoError(t, err)

	assert.NoError(t, parserstep3.Execute(stmt))
	assert.NoError(t, parserstep4.Execute(stmt))
	assert.NoError(t, parserstep5.Execute(stmt, nil))

	paramNs, err := cmn.NewNamespaceFromDefinition(fd)
	assert.NoError(t, err)
	constNs, err := cmn.NewNamespaceFromConstants(map[string]any{})
	assert.NoError(t, err)

	if err := Execute(stmt, paramNs, constNs); err != nil {
		t.Fatalf("Execute step6 failed: %v", err)
	}

	// Inspect WHERE clause tokens to ensure the boolean literal after directive was not left as raw
	clauses := stmt.Clauses()
	foundWhere := false

	for _, c := range clauses {
		toks := c.RawTokens()
		if len(toks) > 0 && toks[0].Type == tokenizer.WHERE {
			foundWhere = true
			// We expect to see DUMMY_START ... DUMMY_END sequence after the directive position
			dummyStart := 0
			dummyEnd := 0
			boolLiterals := 0

			for _, tk := range toks {
				if tk.Type == tokenizer.DUMMY_START {
					dummyStart++
				}

				if tk.Type == tokenizer.DUMMY_END {
					dummyEnd++
				}

				if tk.Type == tokenizer.BOOLEAN {
					boolLiterals++
				}
			}
			// DUMMY_START and DUMMY_END must be present at least once
			assert.True(t, dummyStart >= 1 && dummyEnd >= 1, "expected DUMMY_START/DUMMY_END to wrap boolean value")
			// And the raw trailing boolean literal should not remain duplicated (should be wrapped)
			// There will be one BOOLEAN inside the dummy wrapper; ensure not more than 1 remains
			assert.True(t, boolLiterals <= 1, "unexpected extra raw BOOLEAN literal tokens present")
		}
	}

	assert.True(t, foundWhere, "WHERE clause not found")
}
