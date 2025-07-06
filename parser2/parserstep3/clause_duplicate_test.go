package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func TestCheckClauseDuplicates(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.WHERE_CLAUSE), &perr)
		assert.Equal(t, 0, len(perr.Errors))
	})
	t.Run("with duplicates", func(t *testing.T) {
		var perr cmn.ParseError
		ValidateClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.SELECT_CLAUSE), &perr)
		assert.NotEqual(t, 0, len(perr.Errors))
		assert.Contains(t, perr.Error(), "duplicate clause")
	})
}
