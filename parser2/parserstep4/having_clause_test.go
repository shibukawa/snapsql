package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/parser2/parserstep3"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeHavingClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{
			name:      "HAVING without GROUP BY is forbidden",
			sql:       "SELECT name FROM users HAVING COUNT(*) > 1",
			wantError: true,
		},
		{
			name:      "HAVING with GROUP BY is allowed",
			sql:       "SELECT name, COUNT(*) FROM users GROUP BY name HAVING COUNT(*) > 1",
			wantError: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)

			selectStmt, ok := ast.(*cmn.SelectStatement)
			assert.True(t, ok)
			havingClause := selectStmt.Having
			groupByClause := selectStmt.GroupBy
			perr := &cmn.ParseError{}
			FinalizeHavingClause(havingClause, groupByClause, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors), "should return error")
			} else {
				assert.Equal(t, 0, len(perr.Errors), "should not return error")
			}
		})
	}
}
