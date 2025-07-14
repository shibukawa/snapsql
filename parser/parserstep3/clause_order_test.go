package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// parseClausesFromSQL is now in test_helper.go

func TestValidateClauseOrder(t *testing.T) {
	tests := []struct {
		name      string
		statement cmn.NodeType
		sql       string
		wantErr   bool
		wantMsg   string
	}{
		{
			name:      "SELECT valid order",
			statement: cmn.SELECT_STATEMENT,
			sql:       "SELECT id FROM t WHERE id > 0 GROUP BY id HAVING count(id) > 1 ORDER BY id LIMIT 10",
			wantErr:   false,
		},
		{
			name:      "SELECT invalid order (LIMIT before FROM)",
			statement: cmn.SELECT_STATEMENT,
			sql:       "SELECT id LIMIT 10 from t WHERE id > 0",
			wantErr:   true,
			wantMsg:   "clause order violation: Please move 'from' at 1:20 clause before 'LIMIT' clause at 1:11",
		},
		{
			name:      "INSERT INTO VALUES valid order",
			statement: cmn.INSERT_INTO_STATEMENT,
			sql:       "INSERT INTO t (id) VALUES (1)",
			wantErr:   false,
		},
		{
			name:      "INSERT INTO SELECT valid order",
			statement: cmn.INSERT_INTO_STATEMENT,
			sql:       "INSERT INTO t (id) SELECT id FROM t2 WHERE id > 0",
			wantErr:   false,
		},
		{
			name:      "INSERT INTO SELECT invalid order (RETURNING before SELECT)",
			statement: cmn.INSERT_INTO_STATEMENT,
			sql:       "INSERT INTO t (id) RETURNING id select id WHERE id > 0 ",
			wantErr:   true,
			wantMsg:   "clause order violation: Please move 'select' at 1:33 clause before 'RETURNING' clause at 1:20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses := parseClausesFromSQL(t, tt.sql)
			var perr cmn.ParseError
			ValidateClauseOrder(tt.statement, clauses, &perr)
			if tt.wantErr {
				assert.Equal(t, 1, len(perr.Errors), "expected 1 error, got %d: %v", len(perr.Errors), perr.Errors)
				assert.Equal(t, tt.wantMsg, perr.Errors[0].Error())
			} else {
				assert.Equal(t, 0, len(perr.Errors), "expected no errors, got %d: %v", len(perr.Errors), perr.Errors)
			}
		})
	}
}
