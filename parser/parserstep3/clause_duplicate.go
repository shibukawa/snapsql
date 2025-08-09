package parserstep3

import (
	"errors"
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Sentinel error for duplicate clause detection
var ErrDuplicateClause = errors.New("duplicate clause detected")

// ValidateClauseDuplicates checks for duplicate clauses in the clause list.
// It appends errors to the provided ParseError pointer, does not return error.
func ValidateClauseDuplicates(clauses []cmn.ClauseNode, perr *cmn.ParseError) {
	seen := make(map[cmn.NodeType]int)
	for _, clause := range clauses {
		seen[clause.Type()]++
		if seen[clause.Type()] > 1 {
			if perr != nil {
				perr.Add(fmt.Errorf("%w: %s", ErrDuplicateClause, clauseKeywordFromTokens(clause)))
			}

			return
		}
	}
}
