package parserstep3

import (
	"errors"
	"fmt"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel error for duplicate clause detection
var ErrDuplicateClause = errors.New("duplicate clause detected")

// Check for duplicate clauses in the clause list
func CheckClauseDuplicates(clauses []cmn.ClauseNode) error {
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

// Check for required (mandatory) clauses for a given statement type
// Sentinel error for required clause missing
var ErrRequiredClauseMissing = errors.New("required clause missing")

func CheckClauseRequired(stmtType cmn.NodeType, clauses []cmn.ClauseNode) error {
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
		return nil // No check for other types
	}
	if _, ok := clauseOrder[key]; !ok {
		return nil
	}
	// Define required clauses for each statement type
	required := map[string][]cmn.NodeType{
		"SELECT":        {cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE},
		"INSERT_VALUES": {cmn.INSERT_INTO_CLAUSE, cmn.VALUES_CLAUSE},
		"INSERT_SELECT": {cmn.INSERT_INTO_CLAUSE, cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE},
		"UPDATE":        {cmn.UPDATE_CLAUSE, cmn.SET_CLAUSE},
		"DELETE":        {cmn.DELETE_FROM_CLAUSE},
	}
	req, ok := required[key]
	if !ok {
		return nil
	}
	found := make(map[cmn.NodeType]bool)
	for _, clause := range clauses {
		found[clause.Type()] = true
	}
	for _, r := range req {
		if !found[r] {
			// Use a dummy ClauseNode to get the keyword for error message
			dummy := &dummyClauseNode{nodeType: r}
			return fmt.Errorf("%w: %s", ErrRequiredClauseMissing, clauseKeywordFromTokens(dummy))
		}
	}
	return nil
}

// dummyClauseNode is used only for error message keyword extraction
// Implements cmn.ClauseNode
// All methods except RawTokens/Type are dummies
// RawTokens returns a fake token for error message

type dummyClauseNode struct {
	nodeType cmn.NodeType
}

func (d *dummyClauseNode) Type() cmn.NodeType { return d.nodeType }
func (d *dummyClauseNode) RawTokens() []tok.Token {
	types, ok := nodeTypeToTokenTypes[d.nodeType]
	if !ok || len(types) == 0 {
		return nil
	}
	return []tok.Token{{Type: types[len(types)-1], Value: strings.ToLower(types[len(types)-1].String())}}
}
func (d *dummyClauseNode) ContentTokens() []tok.Token { return nil }
func (d *dummyClauseNode) IfDirective() string        { return "" }
func (d *dummyClauseNode) Position() tok.Position     { return tok.Position{} }
func (d *dummyClauseNode) String() string             { return d.nodeType.String() }
