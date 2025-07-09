package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/parser2/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeFromClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
		wantTable []string
		wantOrig  []string
		wantJoin  []cmn.JoinType
		wantSubq  []bool
	}{
		{
			name:      "single table",
			sql:       "SELECT * FROM users",
			wantError: false,
			wantTable: []string{"users"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
			wantSubq:  []bool{false},
		},
		{
			name:      "single table with alias",
			sql:       "SELECT * FROM users AS u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
			wantSubq:  []bool{false},
		},
		{
			name:      "single table with alias(no AS)",
			sql:       "SELECT * FROM users u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
			wantSubq:  []bool{false},
		},
		{
			name:      "subquery with alias",
			sql:       "SELECT * FROM (SELECT id FROM users) u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{""},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
			wantSubq:  []bool{true},
		},
		{
			name:      "subquery without alias",
			sql:       "SELECT * FROM (SELECT id FROM users)",
			wantError: true,
		},
		{
			name:      "implicit cross join table",
			sql:       "SELECT * FROM a, b",
			wantError: true,
		},
		{
			name:      "inner join",
			sql:       "SELECT * FROM users INNER JOIN orders ON users.id = orders.user_id",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinInner},
			wantSubq:  []bool{false, false},
		},
		{
			name:      "left join with alias",
			sql:       "SELECT * FROM users u LEFT JOIN orders AS o ON u.id = o.user_id",
			wantError: false,
			wantTable: []string{"u", "o"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft},
			wantSubq:  []bool{false, false},
		},
		{
			name:      "right join",
			sql:       "SELECT * FROM users RIGHT JOIN orders ON users.id = orders.user_id",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinRight},
			wantSubq:  []bool{false, false},
		},
		{
			name:      "full join",
			sql:       "SELECT * FROM users FULL JOIN orders ON users.id = orders.user_id",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinFull},
			wantSubq:  []bool{false, false},
		},
		{
			name:      "cross join",
			sql:       "SELECT * FROM users CROSS JOIN orders",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinCross},
			wantSubq:  []bool{false, false},
		},
		{
			name:      "two left joins",
			sql:       "SELECT * FROM users AS u LEFT JOIN orders AS o ON u.id = o.user_id LEFT JOIN payments AS p ON o.id = p.order_id",
			wantError: false,
			wantTable: []string{"u", "o", "p"},
			wantOrig:  []string{"users", "orders", "payments"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft, cmn.JoinLeft},
			wantSubq:  []bool{false, false, false},
		},
		{
			name:      "three left joins",
			sql:       "SELECT * FROM users AS u LEFT JOIN orders AS o ON u.id = o.user_id LEFT JOIN payments AS p ON o.id = p.order_id LEFT JOIN refunds AS r ON p.id = r.payment_id",
			wantError: false,
			wantTable: []string{"u", "o", "p", "r"},
			wantOrig:  []string{"users", "orders", "payments", "refunds"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft, cmn.JoinLeft, cmn.JoinLeft},
			wantSubq:  []bool{false, false, false, false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, _ := tok.Tokenize(tc.sql)
			stmt, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(stmt)
			assert.NoError(t, err)
			selectStmt, ok := stmt.(*cmn.SelectStatement)
			assert.True(t, ok)
			fromClause := selectStmt.From
			perr := &cmn.ParseError{}
			FinalizeFromClause(fromClause, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors), "should have parse error")
			} else {
				assert.Equal(t, 0, len(perr.Errors), "should not have parse error")
				if tc.wantTable != nil {
					got := fromClause.Tables
					assert.Equal(t, len(tc.wantTable), len(got), "table count")
					gotTable := make([]string, len(got))
					gotOrig := make([]string, len(got))
					gotJoin := make([]cmn.JoinType, len(got))
					gotSubq := make([]bool, len(got))
					for i := range got {
						gotTable[i] = got[i].Table.Name
						gotOrig[i] = got[i].OriginalTable
						gotJoin[i] = got[i].JoinType
						gotSubq[i] = got[i].IsSubquery
					}
					assert.Equal(t, tc.wantTable, gotTable, "table name")
					assert.Equal(t, tc.wantOrig, gotOrig, "original table")
					assert.Equal(t, tc.wantJoin, gotJoin, "join type")
					assert.Equal(t, tc.wantSubq, gotSubq, "is subquery")
				}
			}
		})
	}
}
