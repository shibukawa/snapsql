package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

func TestCheckClauseRequired(t *testing.T) {
	tests := []struct {
		name      string
		statement cmn.NodeType
		sql       string
		wantErr   bool
		msg       string
	}{
		{
			name:      "SELECT statement with all required clauses",
			statement: cmn.SELECT_STATEMENT,
			sql:       "SELECT id FROM t",
			wantErr:   false,
			msg:       "SELECT with FROM should not error",
		},
		{
			name:      "SELECT statement missing FROM",
			statement: cmn.SELECT_STATEMENT,
			sql:       "SELECT id",
			wantErr:   true,
			msg:       "SELECT missing FROM should error",
		},
		{
			name:      "INSERT VALUES with all required clauses",
			statement: cmn.INSERT_INTO_STATEMENT,
			sql:       "INSERT INTO t (id) VALUES (1)",
			wantErr:   false,
			msg:       "INSERT INTO with VALUES should not error",
		},
		{
			name:      "INSERT VALUES missing VALUES",
			statement: cmn.INSERT_INTO_STATEMENT,
			sql:       "INSERT INTO t (id)",
			wantErr:   true,
			msg:       "INSERT INTO missing VALUES should error",
		},
		{
			name:      "UPDATE with all required clauses",
			statement: cmn.UPDATE_STATEMENT,
			sql:       "UPDATE t SET id = 1",
			wantErr:   false,
			msg:       "UPDATE with SET should not error",
		},
		{
			name:      "UPDATE missing SET",
			statement: cmn.UPDATE_STATEMENT,
			sql:       "UPDATE t",
			wantErr:   true,
			msg:       "UPDATE missing SET should error",
		},
		{
			name:      "DELETE with all required clauses",
			statement: cmn.DELETE_FROM_STATEMENT,
			sql:       "DELETE FROM t",
			wantErr:   false,
			msg:       "DELETE FROM should not error",
		},
		{
			name:      "DELETE missing DELETE_FROM",
			statement: cmn.DELETE_FROM_STATEMENT,
			sql:       "",
			wantErr:   true,
			msg:       "DELETE missing DELETE_FROM should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var clauses []cmn.ClauseNode
			if tt.sql != "" {
				clauses = parseClausesFromSQL(t, tt.sql)
			} else {
				clauses = []cmn.ClauseNode{}
			}

			var perr cmn.ParseError

			ValidateClauseRequired(tt.statement, clauses, &perr)

			if tt.wantErr {
				if len(perr.Errors) == 0 {
					t.Errorf("%s: want error but got none", tt.msg)
				}

				assert.Contains(t, perr.Error(), "required clause")
			} else {
				if len(perr.Errors) != 0 {
					t.Errorf("%s: got error(s): %v", tt.msg, perr.Errors)
				}
			}
		})
	}
}
