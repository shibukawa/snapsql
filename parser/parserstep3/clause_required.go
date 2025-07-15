package parserstep3

import (
	"errors"
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Check for required (mandatory) clauses for a given statement type
// Sentinel error for required clause missing
var ErrRequiredClauseMissing = errors.New("required clause missing")

var required = map[string][]cmn.NodeType{
	"SELECT":        {cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE},
	"INSERT_VALUES": {cmn.INSERT_INTO_CLAUSE, cmn.VALUES_CLAUSE},
	"INSERT_SELECT": {cmn.INSERT_INTO_CLAUSE, cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE},
	"UPDATE":        {cmn.UPDATE_CLAUSE, cmn.SET_CLAUSE},
	"DELETE":        {cmn.DELETE_FROM_CLAUSE},
}

// ValidateClauseRequired checks for required (mandatory) clauses for a given statement type.
// It appends errors to the provided ParseError pointer, does not return error.
func ValidateClauseRequired(stmtType cmn.NodeType, clauses []cmn.ClauseNode, perr *cmn.ParseError) {
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
		return // No check for other types
	}
	if _, ok := clauseOrder[key]; !ok {
		return
	}
	// Define required clauses for each statement type

	req, ok := required[key]
	if !ok {
		return
	}
	found := make(map[cmn.NodeType]bool)
	for _, clause := range clauses {
		found[clause.Type()] = true
	}
	for _, r := range req {
		if !found[r] {
			if perr != nil {
				perr.Add(fmt.Errorf("%w: %s clause is required", ErrRequiredClauseMissing, r.String()))
			}
		}
	}
}
