package parserstep3

import (
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// NodeType to TokenType mapping for clause keyword extraction
var nodeTypeToTokenTypes = map[cmn.NodeType][]tok.TokenType{
	cmn.SELECT_CLAUSE:      {tok.SELECT},
	cmn.FROM_CLAUSE:        {tok.FROM},
	cmn.WHERE_CLAUSE:       {tok.WHERE},
	cmn.GROUP_BY_CLAUSE:    {tok.GROUP},
	cmn.HAVING_CLAUSE:      {tok.HAVING},
	cmn.ORDER_BY_CLAUSE:    {tok.ORDER},
	cmn.LIMIT_CLAUSE:       {tok.LIMIT},
	cmn.OFFSET_CLAUSE:      {tok.OFFSET},
	cmn.RETURNING_CLAUSE:   {tok.RETURNING},
	cmn.INSERT_INTO_CLAUSE: {tok.INTO}, // Prefer INTO for error message
	cmn.VALUES_CLAUSE:      {tok.VALUES},
	cmn.UPDATE_CLAUSE:      {tok.UPDATE},
	cmn.SET_CLAUSE:         {tok.SET},
	cmn.DELETE_FROM_CLAUSE: {tok.FROM},
	cmn.WITH_CLAUSE:        {tok.WITH},
	cmn.ON_CONFLICT_CLAUSE: {tok.CONFLICT, tok.ON},
	cmn.FOR_CLAUSE:         {tok.FOR},
}

// Extracts the actual clause keyword as written by the user from RawTokens
func clauseKeywordFromTokens(node cmn.ClauseNode) string {
	types, ok := nodeTypeToTokenTypes[node.Type()]
	if !ok {
		// fallback: use NodeType string
		return node.Type().String()
	}
	tokens := node.RawTokens()
	for _, typ := range types {
		// Search from the end for multi-keyword (e.g. INTO)
		for i := len(tokens) - 1; i >= 0; i-- {
			if tokens[i].Type == typ {
				return tokens[i].Value
			}
		}
	}
	// fallback: use NodeType string
	return node.Type().String()
}

// ErrClauseOrderViolation is returned when clause order is invalid
var ErrClauseOrderViolation = fmt.Errorf("clause order violation")

// ValidateClauseOrder validates the order of clauses for a given statement type.
// Returns nil if order is valid, otherwise returns an error.
func ValidateClauseOrder(stmtType cmn.NodeType, clauses []cmn.ClauseNode) error {
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
		return nil // No order check for other types
	}
	allowed, ok := clauseOrder[key]
	if !ok {
		return nil
	}
	prevOrder := -1
	for i, clause := range clauses {
		order, ok := allowed[clause.Type()]
		if !ok {
			continue // skip unknown clauses (should be filtered before)
		}
		if order < prevOrder {
			minJ := -1
			minOrder := 9999
			for j := 0; j < i; j++ {
				prevOrderJ, ok2 := allowed[clauses[j].Type()]
				if ok2 && prevOrderJ > order && prevOrderJ < minOrder {
					minOrder = prevOrderJ
					minJ = j
				}
			}
			if minJ != -1 {
				return fmt.Errorf("%w: Please move %s clause before %s clause", ErrClauseOrderViolation, clauseKeywordFromTokens(clause), clauseKeywordFromTokens(clauses[minJ]))
			}
			if i+1 < len(clauses) {
				return fmt.Errorf("%w: Please move %s clause before %s clause", ErrClauseOrderViolation, clauseKeywordFromTokens(clause), clauseKeywordFromTokens(clauses[i+1]))
			}
			return fmt.Errorf("%w: Please move %s clause before %s clause", ErrClauseOrderViolation, clauseKeywordFromTokens(clause), clauseKeywordFromTokens(clause))
		}
		prevOrder = order
	}
	return nil
}
