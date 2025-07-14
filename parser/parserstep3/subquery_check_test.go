package parserstep3

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestCheckSubqueryUsage(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "subquery_in_select_field_is_ok",
			sql:     "SELECT id, (SELECT 1) as x FROM t",
			wantErr: false,
		},
		{
			name:    "subquery_in_where_is_ok",
			sql:     "SELECT id FROM t WHERE id IN (SELECT 1)",
			wantErr: false,
		},
		{
			name:    "subquery_in_groupby_is_error",
			sql:     "SELECT id FROM t GROUP BY (SELECT 1)",
			wantErr: true,
		},
		{
			name:    "subquery_in_orderby_is_error",
			sql:     "SELECT id FROM t ORDER BY (SELECT 1)",
			wantErr: true,
		},
		{
			name:    "subquery_in_having_is_ok",
			sql:     "SELECT id FROM t HAVING id IN (SELECT 1)",
			wantErr: false,
		},
		{
			name:    "subquery_in_limit_is_error",
			sql:     "SELECT id FROM t LIMIT (SELECT 1)",
			wantErr: true,
		},
		{
			name:    "subquery_in_offset_is_error",
			sql:     "SELECT id FROM t OFFSET (SELECT 1)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses := parseClausesFromSQL(t, tt.sql)
			perr := &cmn.ParseError{}
			CheckSubqueryUsage(clauses, perr)
			if tt.wantErr {
				assert.Equal(t, 1, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
}
