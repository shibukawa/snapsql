package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func TestCheckClauseDuplicates(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		err := CheckClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.WHERE_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("with duplicates", func(t *testing.T) {
		err := CheckClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.SELECT_CLAUSE))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "duplicate clause")
		}
	})
}

func TestCheckClauseRequired(t *testing.T) {
	t.Run("SELECT statement with all required clauses", func(t *testing.T) {
		err := CheckClauseRequired(cmn.SELECT_STATEMENT, clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("SELECT statement missing FROM", func(t *testing.T) {
		err := CheckClauseRequired(cmn.SELECT_STATEMENT, clauseNodes(cmn.SELECT_CLAUSE))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "required clause")
		}
	})
	t.Run("INSERT VALUES with all required clauses", func(t *testing.T) {
		err := CheckClauseRequired(cmn.INSERT_INTO_STATEMENT, clauseNodes(cmn.INSERT_INTO_CLAUSE, cmn.VALUES_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("INSERT VALUES missing VALUES", func(t *testing.T) {
		err := CheckClauseRequired(cmn.INSERT_INTO_STATEMENT, clauseNodes(cmn.INSERT_INTO_CLAUSE))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "required clause")
		}
	})
	t.Run("UPDATE with all required clauses", func(t *testing.T) {
		err := CheckClauseRequired(cmn.UPDATE_STATEMENT, clauseNodes(cmn.UPDATE_CLAUSE, cmn.SET_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("UPDATE missing SET", func(t *testing.T) {
		err := CheckClauseRequired(cmn.UPDATE_STATEMENT, clauseNodes(cmn.UPDATE_CLAUSE))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "required clause")
		}
	})
	t.Run("DELETE with all required clauses", func(t *testing.T) {
		err := CheckClauseRequired(cmn.DELETE_FROM_STATEMENT, clauseNodes(cmn.DELETE_FROM_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("DELETE missing DELETE_FROM", func(t *testing.T) {
		err := CheckClauseRequired(cmn.DELETE_FROM_STATEMENT, clauseNodes())
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "required clause")
		}
	})
}
