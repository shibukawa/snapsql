package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func TestCheckClauseDuplicates(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
		msg     string
	}{
		{
			name:    "no duplicates",
			sql:     "SELECT id FROM t WHERE id > 0",
			wantErr: false,
			msg:     "No duplicate clause should not error",
		},
		{
			name:    "with duplicates",
			sql:     "SELECT id FROM t SELECT id",
			wantErr: true,
			msg:     "Duplicate SELECT clause should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses := parseClausesFromSQL(t, tt.sql)
			var perr cmn.ParseError
			ValidateClauseDuplicates(clauses, &perr)
			if tt.wantErr {
				if len(perr.Errors) == 0 {
					t.Errorf("%s: want error but got none", tt.msg)
				}
				assert.Contains(t, perr.Error(), "duplicate clause")
			} else {
				if len(perr.Errors) != 0 {
					t.Errorf("%s: got error(s): %v", tt.msg, perr.Errors)
				}
			}
		})
	}
}
