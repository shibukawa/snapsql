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

// Test that a NULL literal immediately after a directive comment is treated as a dummy literal
// and wrapped with DUMMY_START/DUMMY_END so it does not leak into generated SQL.
func TestNullAfterDirectiveIsDummyLiteral(t *testing.T) {
	sql := `/*#
function_name: create_notification
parameters:
  icon_url: string
*/
INSERT INTO notifications (icon_url)
VALUES (/*= icon_url */NULL)`

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

	if _, err := Execute(stmt, paramNs, constNs); err != nil {
		t.Fatalf("Execute step6 failed: %v", err)
	}

	clauses := stmt.Clauses()

	var (
		dummyStartCount int
		dummyEndCount   int
		nullOutside     bool
	)

	insideDummy := false

	for _, clause := range clauses {
		for _, tk := range clause.RawTokens() {
			switch tk.Type {
			case tokenizer.DUMMY_START:
				dummyStartCount++
				insideDummy = true
			case tokenizer.DUMMY_END:
				dummyEndCount++
				insideDummy = false
			case tokenizer.NULL:
				if !insideDummy {
					nullOutside = true
				}
			}
		}
	}

	assert.True(t, dummyStartCount >= 1 && dummyEndCount >= 1, "expected NULL literal to be wrapped with dummy tokens")
	assert.False(t, nullOutside, "NULL literal should not remain outside dummy wrapper")
}
