package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func TestCheckClauseRequired(t *testing.T) {
	t.Run("SELECT statement with all required clauses", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.SELECT_STATEMENT, clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE), &perr)
		assert.Equal(t, 0, len(perr.Errors))
	})
	t.Run("SELECT statement missing FROM", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.SELECT_STATEMENT, clauseNodes(cmn.SELECT_CLAUSE), &perr)
		assert.NotEqual(t, 0, len(perr.Errors))
		assert.Contains(t, perr.Error(), "required clause")
	})
	t.Run("INSERT VALUES with all required clauses", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.INSERT_INTO_STATEMENT, clauseNodes(cmn.INSERT_INTO_CLAUSE, cmn.VALUES_CLAUSE), &perr)
		assert.Equal(t, 0, len(perr.Errors))
	})
	t.Run("INSERT VALUES missing VALUES", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.INSERT_INTO_STATEMENT, clauseNodes(cmn.INSERT_INTO_CLAUSE), &perr)
		assert.NotEqual(t, 0, len(perr.Errors))
		assert.Contains(t, perr.Error(), "required clause")
	})
	t.Run("UPDATE with all required clauses", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.UPDATE_STATEMENT, clauseNodes(cmn.UPDATE_CLAUSE, cmn.SET_CLAUSE), &perr)
		assert.Equal(t, 0, len(perr.Errors))
	})
	t.Run("UPDATE missing SET", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.UPDATE_STATEMENT, clauseNodes(cmn.UPDATE_CLAUSE), &perr)
		assert.NotEqual(t, 0, len(perr.Errors))
		assert.Contains(t, perr.Error(), "required clause")
	})
	t.Run("DELETE with all required clauses", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.DELETE_FROM_STATEMENT, clauseNodes(cmn.DELETE_FROM_CLAUSE), &perr)
		assert.Equal(t, 0, len(perr.Errors))
	})
	t.Run("DELETE missing DELETE_FROM", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseRequired(cmn.DELETE_FROM_STATEMENT, clauseNodes(), &perr)
		assert.NotEqual(t, 0, len(perr.Errors))
		assert.Contains(t, perr.Error(), "required clause")
	})
}
