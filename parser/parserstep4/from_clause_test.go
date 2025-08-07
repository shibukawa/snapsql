package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
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
	}{
		{
			name:      "single table",
			sql:       "SELECT * FROM users",
			wantError: false,
			wantTable: []string{"users"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
		},
		{
			name:      "single table with alias",
			sql:       "SELECT * FROM users AS u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
		},
		{
			name:      "single table with alias(no AS)",
			sql:       "SELECT * FROM users u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{"users"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
		},
		{
			name:      "subquery with alias",
			sql:       "SELECT * FROM (SELECT id FROM users) AS u",
			wantError: false,
			wantTable: []string{"u"},
			wantOrig:  []string{""},
			wantJoin:  []cmn.JoinType{cmn.JoinNone},
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
		},
		{
			name:      "left join with alias",
			sql:       "SELECT * FROM users u LEFT JOIN orders AS o ON u.id = o.user_id",
			wantError: false,
			wantTable: []string{"u", "o"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft},
		},
		{
			name:      "right join",
			sql:       "SELECT * FROM users RIGHT JOIN orders ON users.id = orders.user_id",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinRight},
		},
		{
			name:      "full join",
			sql:       "SELECT * FROM users FULL JOIN orders ON users.id = orders.user_id",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinFull},
		},
		{
			name:      "cross join",
			sql:       "SELECT * FROM users CROSS JOIN orders",
			wantError: false,
			wantTable: []string{"users", "orders"},
			wantOrig:  []string{"users", "orders"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinCross},
		},
		{
			name:      "two left joins",
			sql:       "SELECT * FROM users AS u LEFT JOIN orders AS o ON u.id = o.user_id LEFT JOIN payments AS p ON o.id = p.order_id",
			wantError: false,
			wantTable: []string{"u", "o", "p"},
			wantOrig:  []string{"users", "orders", "payments"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft, cmn.JoinLeft},
		},
		{
			name:      "three left joins",
			sql:       "SELECT * FROM users AS u LEFT JOIN orders AS o ON u.id = o.user_id LEFT JOIN payments AS p ON o.id = p.order_id LEFT JOIN refunds AS r ON p.id = r.payment_id",
			wantError: false,
			wantTable: []string{"u", "o", "p", "r"},
			wantOrig:  []string{"users", "orders", "payments", "refunds"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinLeft, cmn.JoinLeft, cmn.JoinLeft},
		},
		{
			name:      "Subquery after JOIN",
			sql:       "SELECT * FROM users JOIN (SELECT * FROM orders) o ON users.id = o.user_id",
			wantError: false,
			wantTable: []string{"users", "o"},
			wantOrig:  []string{"users", ""},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinInner},
		},
		{
			name:      "Subquery after JOIN without alias",
			sql:       "SELECT * FROM users JOIN (SELECT * FROM orders) ON users.id = orders.user_id",
			wantError: true,
		},
		{
			name:      "Multiple subqueries after JOIN",
			sql:       "SELECT * FROM (SELECT * FROM users) AS u JOIN (SELECT * FROM orders) AS o ON u.id = o.user_id",
			wantError: false,
			wantTable: []string{"u", "o"},
			wantOrig:  []string{"", ""},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinInner},
		},
		{
			name:      "Table then subquery after JOIN",
			sql:       "SELECT * FROM users JOIN (SELECT * FROM orders) AS o ON users.id = o.user_id JOIN payments AS p ON o.id = p.order_id",
			wantError: false,
			wantTable: []string{"users", "o", "p"},
			wantOrig:  []string{"users", "", "payments"},
			wantJoin:  []cmn.JoinType{cmn.JoinNone, cmn.JoinInner, cmn.JoinInner},
		},
		{
			name:      "Table then subquery after JOIN without alias",
			sql:       "SELECT * FROM users JOIN (SELECT * FROM orders) ON users.id = orders.user_id JOIN payments p ON orders.id = p.order_id",
			wantError: true,
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
			finalizeFromClause(fromClause, perr)
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
					for i := range got {
						gotTable[i] = got[i].Name
						gotOrig[i] = got[i].TableName
						gotJoin[i] = got[i].JoinType
					}
					assert.Equal(t, tc.wantTable, gotTable, "table name")
					assert.Equal(t, tc.wantOrig, gotOrig, "original table")
					assert.Equal(t, tc.wantJoin, gotJoin, "join type")
				}
			}
		})
	}
}

func TestFinalizeFromClause_InvalidJoinCombinations(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{"JOIN at start", "SELECT * FROM JOIN users u"},
		{"Double JOIN", "SELECT * FROM users JOIN JOIN orders"},         //nolint:dupword // intentional duplicate for testing
		{"Double INNER", "SELECT * FROM users INNER INNER JOIN orders"}, //nolint:dupword // intentional duplicate for testing
		{"Double LEFT", "SELECT * FROM users LEFT LEFT JOIN orders"},    //nolint:dupword // intentional duplicate for testing
		{"Double OUTER", "SELECT * FROM users OUTER OUTER JOIN orders"}, //nolint:dupword // intentional duplicate for testing
		{"CROSS with INNER", "SELECT * FROM users CROSS INNER JOIN orders"},
		{"CROSS with LEFT", "SELECT * FROM users CROSS LEFT JOIN orders"},
		{"Double NATURAL", "SELECT * FROM users NATURAL NATURAL JOIN orders"}, //nolint:dupword // intentional duplicate for testing
		{"NATURAL CROSS", "SELECT * FROM users NATURAL CROSS JOIN orders"},
		{"OUTER only", "SELECT * FROM users OUTER JOIN orders"},
		{"INNER only", "SELECT * FROM users INNER orders"},
		{"OUTER before LEFT", "SELECT * FROM users OUTER LEFT JOIN orders"},
		{"RIGHT before NATURAL", "SELECT * FROM users RIGHT NATURAL JOIN orders"},
		{"USING with CROSS JOIN", "SELECT * FROM users CROSS JOIN orders USING (id)"},
		{"USING with NATURAL JOIN", "SELECT * FROM users NATURAL JOIN orders USING (id)"},
		{"USING with NATURAL LEFT JOIN", "SELECT * FROM users NATURAL LEFT JOIN orders USING (id)"},
		{"USING with NATURAL RIGHT JOIN", "SELECT * FROM users NATURAL RIGHT JOIN orders USING (id)"},
		{"USING with NATURAL FULL JOIN", "SELECT * FROM users NATURAL FULL JOIN orders USING (id)"},
		{"No ON/USING with INNER JOIN", "SELECT * FROM users INNER JOIN orders"},
		{"No ON/USING with LEFT JOIN", "SELECT * FROM users LEFT JOIN orders"},
		{"No ON/USING with RIGHT JOIN", "SELECT * FROM users RIGHT JOIN orders"},
		{"No ON/USING with FULL JOIN", "SELECT * FROM users FULL JOIN orders"},
		{"No table name", "SELECT * FROM AS u"},
		{"invalid for table name before alias", "SELECT * FROM 1 AS u"},
		{"invalid for table name before alias (No AS)", "SELECT * FROM 1 u"},
		{"invalid for table name", "SELECT * FROM 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, _ := tok.Tokenize(tc.sql)
			stmt, err := parserstep2.Execute(tokens)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			err = parserstep3.Execute(stmt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			selectStmt, ok := stmt.(*cmn.SelectStatement)
			if !ok {
				t.Fatalf("cast should be success")
			}
			fromClause := selectStmt.From
			perr := &cmn.ParseError{}
			finalizeFromClause(fromClause, perr)
			assert.NotEqual(t, 0, len(perr.Errors), "should have parse error for invalid join combination")
		})
	}
}
