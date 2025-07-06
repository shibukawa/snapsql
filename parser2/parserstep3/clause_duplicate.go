package parserstep3

import (
	"errors"
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Sentinel error for duplicate clause detection
var ErrDuplicateClause = errors.New("duplicate clause detected")

// Check for duplicate clauses in the clause list
func ValidateClauseDuplicates(clauses []cmn.ClauseNode) error {
	seen := make(map[cmn.NodeType]int)
	for _, clause := range clauses {
		seen[clause.Type()]++
		if seen[clause.Type()] > 1 {
			// Wrap with sentinel error
			return fmt.Errorf("%w: %s", ErrDuplicateClause, clauseKeywordFromTokens(clause))
		}
	}
	return nil
}
