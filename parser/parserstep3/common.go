package parserstep3

import cmn "github.com/shibukawa/snapsql/parser/parsercommon"

// hasSelectClause returns true if SELECT_CLAUSE is present in the clause list
func hasSelectClause(clauses []cmn.ClauseNode) bool {
	for _, c := range clauses {
		if c.Type() == cmn.SELECT_CLAUSE {
			return true
		}
	}

	return false
}

// clauseOrder defines allowed clause types and their order for each statement type
var clauseOrder = map[string]map[cmn.NodeType]int{
	"SELECT": {
		cmn.WITH_CLAUSE:      1,
		cmn.SELECT_CLAUSE:    2,
		cmn.FROM_CLAUSE:      3,
		cmn.WHERE_CLAUSE:     4,
		cmn.GROUP_BY_CLAUSE:  5,
		cmn.HAVING_CLAUSE:    6,
		cmn.ORDER_BY_CLAUSE:  7,
		cmn.LIMIT_CLAUSE:     8,
		cmn.OFFSET_CLAUSE:    9,
		cmn.RETURNING_CLAUSE: 10,
		cmn.FOR_CLAUSE:       11,
	},
	"INSERT_SELECT": {
		cmn.WITH_CLAUSE:        1,
		cmn.INSERT_INTO_CLAUSE: 2,
		cmn.SELECT_CLAUSE:      3,
		cmn.FROM_CLAUSE:        4,
		cmn.WHERE_CLAUSE:       5,
		cmn.GROUP_BY_CLAUSE:    6,
		cmn.HAVING_CLAUSE:      7,
		cmn.ORDER_BY_CLAUSE:    8,
		cmn.LIMIT_CLAUSE:       9,
		cmn.OFFSET_CLAUSE:      10,
		cmn.RETURNING_CLAUSE:   11,
		cmn.ON_CONFLICT_CLAUSE: 12,
	},
	"INSERT_VALUES": {
		cmn.WITH_CLAUSE:        1,
		cmn.INSERT_INTO_CLAUSE: 2,
		cmn.VALUES_CLAUSE:      3,
		cmn.RETURNING_CLAUSE:   4,
		cmn.ON_CONFLICT_CLAUSE: 5,
	},
	"UPDATE": {
		cmn.WITH_CLAUSE:      1,
		cmn.UPDATE_CLAUSE:    2,
		cmn.SET_CLAUSE:       3,
		cmn.WHERE_CLAUSE:     4,
		cmn.RETURNING_CLAUSE: 5,
	},
	"DELETE": {
		cmn.WITH_CLAUSE:        1,
		cmn.DELETE_FROM_CLAUSE: 2,
		cmn.WHERE_CLAUSE:       3,
		cmn.RETURNING_CLAUSE:   4,
	},
}
