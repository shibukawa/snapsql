package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func TestCheckClauseDuplicates(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		err := ValidateClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.WHERE_CLAUSE))
		assert.NoError(t, err)
	})
	t.Run("with duplicates", func(t *testing.T) {
		err := ValidateClauseDuplicates(clauseNodes(cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.SELECT_CLAUSE))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "duplicate clause")
		}
	})
}
