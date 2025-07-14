package parserstep2_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestClauseContentTokensAndRawTokens(t *testing.T) {

	t.Parallel()
	testCases := []struct {
		name        string
		input       string
		clauseTyp   parsercommon.NodeType
		wantRawLen  int
		wantBodyLen int
	}{
		{
			name:        "SELECT only",
			input:       "SELECT a, b, c",
			clauseTyp:   parsercommon.SELECT_CLAUSE,
			wantRawLen:  9, // SELECT, a, ,, b, ,, c
			wantBodyLen: 7, // a, ,, b, ,, c
		},
		{
			name:        "FROM only",
			input:       "SELECT * FROM users",
			clauseTyp:   parsercommon.FROM_CLAUSE,
			wantRawLen:  3, // FROM, users
			wantBodyLen: 1, // users
		},
		{
			name:        "WHERE only",
			input:       "SELECT * FROM t WHERE id = 1",
			clauseTyp:   parsercommon.WHERE_CLAUSE,
			wantRawLen:  7, // WHERE, id, =, 1
			wantBodyLen: 5, // id, =, 1
		},
		{
			name:        "INSERT INTO only",
			input:       "INSERT INTO users (id)",
			clauseTyp:   parsercommon.INSERT_INTO_CLAUSE,
			wantRawLen:  9, // INSERT, INTO, users, (, id, ), VALUES
			wantBodyLen: 5, // users, (, id, )
		},
		{
			name:        "UPDATE only",
			input:       "UPDATE users",
			clauseTyp:   parsercommon.UPDATE_CLAUSE,
			wantRawLen:  3, // UPDATE, users, SET
			wantBodyLen: 1, // users, SET
		},
		{
			name:        "DELETE FROM only",
			input:       "DELETE FROM users",
			clauseTyp:   parsercommon.DELETE_FROM_CLAUSE,
			wantRawLen:  5, // DELETE, FROM, users
			wantBodyLen: 1, // FROM, users
		},
		{
			name:        "SELECT with subquery in FROM",
			input:       "SELECT x FROM (SELECT y FROM t2) AS sub",
			clauseTyp:   parsercommon.FROM_CLAUSE,
			wantRawLen:  15, // FROM, (, SELECT, y, FROM, t2, ), AS, sub
			wantBodyLen: 13, // (, SELECT, y, FROM, t2, ), AS, sub
		},
		{
			name:        "SELECT with subquery in WHERE at last",
			input:       "SELECT x FROM t WHERE id IN (SELECT y FROM t2)",
			clauseTyp:   parsercommon.FROM_CLAUSE,
			wantRawLen:  15, // FROM, (, SELECT, y, FROM, t2, ), AS, sub
			wantBodyLen: 13, // (, SELECT, y, FROM, t2, ), AS, sub
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := tokenizer.Tokenize(tc.input)
			assert.NoError(t, err)
			stmt, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			clause := stmt.Clauses()[len(stmt.Clauses())-1] // Get the last clause
			assert.Equal(t, tc.wantRawLen, len(clause.RawTokens()), "RawTokens() length unexpected for %s: got=%v", tc.name, clause.RawTokens())
			assert.Equal(t, tc.wantBodyLen, len(clause.ContentTokens()), "ContentTokens() length unexpected for %s: got=%v", tc.name, clause.ContentTokens())
		})
	}
}
