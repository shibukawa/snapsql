package parserstep3

import (
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

var (
	ErrInvalidInsertClause       = fmt.Errorf("invalid clause for INSERT statement")
	ErrInvalidUpdateClause       = fmt.Errorf("invalid clause for UPDATE statement")
	ErrInvalidDeleteClause       = fmt.Errorf("invalid clause for DELETE statement")
	ErrInvalidClauseForStatement = fmt.Errorf("invalid clause for statement")
)

// ValidateClausePresence filters valid clauses for the given statement type and context.
// It appends errors to the provided ParseError pointer, but always returns the filtered clauses.
func ValidateClausePresence(stmtType cmn.NodeType, clauses []cmn.ClauseNode, perr *cmn.ParseError) []cmn.ClauseNode {
	var valid []cmn.ClauseNode
	var key string
	switch stmtType {
	case cmn.SELECT_STATEMENT:
		key = "SELECT"
	case cmn.INSERT_INTO_STATEMENT:
		if hasSelectClause(clauses) {
			key = "INSERT_SELECT"
		} else {
			key = "INSERT_VALUES"
		}
	case cmn.UPDATE_STATEMENT:
		key = "UPDATE"
	case cmn.DELETE_FROM_STATEMENT:
		key = "DELETE"
	default:
		key = ""
	}
	allowed, ok := clauseOrder[key]
	for _, clause := range clauses {
		if ok {
			if _, ok2 := allowed[clause.Type()]; ok2 {
				valid = append(valid, clause)
				continue
			}
		}
		if perr != nil {
			perr.Add(fmt.Errorf("%w: %s statement cannot have clause: %s", ErrInvalidClauseForStatement, key, clause.Type()))
		}
	}
	return valid
}
