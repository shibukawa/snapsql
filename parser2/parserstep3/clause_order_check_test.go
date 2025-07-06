package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func clauseNodes(types ...cmn.NodeType) []cmn.ClauseNode {
	var nodes []cmn.ClauseNode
	for _, t := range types {
		switch t {
		case cmn.SELECT_CLAUSE:
			nodes = append(nodes, cmn.NewSelectClause(nil, nil))
		case cmn.FROM_CLAUSE:
			nodes = append(nodes, cmn.NewFromClause(nil, nil))
		case cmn.WHERE_CLAUSE:
			nodes = append(nodes, cmn.NewWhereClause(nil, nil))
		case cmn.GROUP_BY_CLAUSE:
			nodes = append(nodes, cmn.NewGroupByClause(nil, nil))
		case cmn.HAVING_CLAUSE:
			nodes = append(nodes, cmn.NewHavingClause(nil, nil))
		case cmn.ORDER_BY_CLAUSE:
			nodes = append(nodes, cmn.NewOrderByClause(nil, nil))
		case cmn.LIMIT_CLAUSE:
			nodes = append(nodes, cmn.NewLimitClause(nil, nil))
		case cmn.OFFSET_CLAUSE:
			nodes = append(nodes, cmn.NewOffsetClause(nil, nil))
		case cmn.RETURNING_CLAUSE:
			nodes = append(nodes, cmn.NewReturningClause(nil, nil))
		case cmn.INSERT_INTO_CLAUSE:
			nodes = append(nodes, cmn.NewInsertIntoClause(nil, nil))
		case cmn.VALUES_CLAUSE:
			nodes = append(nodes, cmn.NewValuesClause(nil, nil))
		case cmn.UPDATE_CLAUSE:
			nodes = append(nodes, cmn.NewUpdateClause(nil, nil))
		case cmn.SET_CLAUSE:
			nodes = append(nodes, cmn.NewSetClause(nil, nil))
		case cmn.DELETE_FROM_CLAUSE:
			nodes = append(nodes, cmn.NewDeleteFromClause(nil, nil))
		default:
			panic("unsupported NodeType in test")
		}
	}
	return nodes
}

func TestValidateClauseOrder_Select(t *testing.T) {
	// Valid order: SELECT -> FROM -> WHERE -> GROUP BY -> HAVING -> ORDER BY -> LIMIT
	valid := clauseNodes(
		cmn.SELECT_CLAUSE,
		cmn.FROM_CLAUSE,
		cmn.WHERE_CLAUSE,
		cmn.GROUP_BY_CLAUSE,
		cmn.HAVING_CLAUSE,
		cmn.ORDER_BY_CLAUSE,
		cmn.LIMIT_CLAUSE,
	)
	err := ValidateClauseOrder(cmn.SELECT_STATEMENT, valid)
	assert.NoError(t, err)

	// Invalid order: WHERE before FROM, and FROM before LIMIT
	invalid := clauseNodes(
		cmn.SELECT_CLAUSE,
		cmn.LIMIT_CLAUSE,
		cmn.FROM_CLAUSE,
		cmn.WHERE_CLAUSE,
	)
	err = ValidateClauseOrder(cmn.SELECT_STATEMENT, invalid)
	assert.Error(t, err)
	// Only the first misplaced clause after the current one should be reported
	// In this case, FROM_CLAUSE should be moved before LIMIT_CLAUSE
	assert.Contains(t, err.Error(), "Please move FROM_CLAUSE clause before LIMIT_CLAUSE clause")
	// The error should mention both clause names
	assert.Contains(t, err.Error(), "FROM_CLAUSE")
	assert.Contains(t, err.Error(), "LIMIT_CLAUSE")
	// Only one error should be reported
	// (simulate by calling once and checking error is not a multi-error)
}

func TestValidateClauseOrder_InsertValues(t *testing.T) {
	// Valid order: INSERT INTO -> VALUES
	valid := clauseNodes(
		cmn.INSERT_INTO_CLAUSE,
		cmn.VALUES_CLAUSE,
	)
	err := ValidateClauseOrder(cmn.INSERT_INTO_STATEMENT, valid)
	assert.NoError(t, err)

	// Invalid order: VALUES before INSERT INTO, and INSERT INTO before RETURNING
	invalid := clauseNodes(
		cmn.VALUES_CLAUSE,
		cmn.RETURNING_CLAUSE,
		cmn.INSERT_INTO_CLAUSE,
	)
	err = ValidateClauseOrder(cmn.INSERT_INTO_STATEMENT, invalid)
	assert.Error(t, err)
	// Only the first misplaced clause after the current one should be reported
	assert.Contains(t, err.Error(), "Please move INSERT_INTO_CLAUSE clause before VALUES_CLAUSE clause")
	assert.Contains(t, err.Error(), "INSERT_INTO_CLAUSE")
	assert.Contains(t, err.Error(), "VALUES_CLAUSE")
}

func TestValidateClauseOrder_InsertSelect(t *testing.T) {
	// Valid order: INSERT INTO -> SELECT -> FROM -> WHERE
	valid := clauseNodes(
		cmn.INSERT_INTO_CLAUSE,
		cmn.SELECT_CLAUSE,
		cmn.FROM_CLAUSE,
		cmn.WHERE_CLAUSE,
	)
	err := ValidateClauseOrder(cmn.INSERT_INTO_STATEMENT, valid)
	assert.NoError(t, err)

	// Invalid order: SELECT before INSERT INTO, and INSERT INTO before WHERE
	invalid := clauseNodes(
		cmn.SELECT_CLAUSE,
		cmn.WHERE_CLAUSE,
		cmn.INSERT_INTO_CLAUSE,
	)
	err = ValidateClauseOrder(cmn.INSERT_INTO_STATEMENT, invalid)
	assert.Error(t, err)
	// Only the first misplaced clause after the current one should be reported
	assert.Contains(t, err.Error(), "Please move INSERT_INTO_CLAUSE clause before SELECT_CLAUSE clause")
	assert.Contains(t, err.Error(), "INSERT_INTO_CLAUSE")
	assert.Contains(t, err.Error(), "SELECT_CLAUSE")
}
