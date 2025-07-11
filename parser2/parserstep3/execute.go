package parserstep3

import (
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Execute runs all clause checks and assigns clause fields to the statement struct.
func Execute(stmt cmn.StatementNode) error {
	perr := &cmn.ParseError{}
	clauses := ValidateClausePresence(stmt.Type(), stmt.Clauses(), perr)
	ValidateClauseDuplicates(clauses, perr)
	ValidateClauseRequired(stmt.Type(), clauses, perr)
	ValidateClauseOrder(stmt.Type(), clauses, perr)
	// フィールド割り当て
	assignStatementFields(stmt, clauses)
	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}

// assignStatementFields assigns clause nodes to the corresponding fields of the statement struct.
func assignStatementFields(stmt cmn.StatementNode, clauses []cmn.ClauseNode) {
	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		assignSelectStatementFields(s, clauses)
	case *cmn.InsertIntoStatement:
		assignInsertStatementFields(s, clauses)
	case *cmn.UpdateStatement:
		assignUpdateStatementFields(s, clauses)
	case *cmn.DeleteFromStatement:
		assignDeleteStatementFields(s, clauses)
	}
}

func assignSelectStatementFields(stmt *cmn.SelectStatement, clauses []cmn.ClauseNode) {
	for _, c := range clauses {
		switch c.Type() {
		case cmn.SELECT_CLAUSE:
			if v, ok := c.(*cmn.SelectClause); ok {
				stmt.Select = v
			}
		case cmn.FROM_CLAUSE:
			if v, ok := c.(*cmn.FromClause); ok {
				stmt.From = v
			}
		case cmn.WHERE_CLAUSE:
			if v, ok := c.(*cmn.WhereClause); ok {
				stmt.Where = v
			}
		case cmn.GROUP_BY_CLAUSE:
			if v, ok := c.(*cmn.GroupByClause); ok {
				stmt.GroupBy = v
			}
		case cmn.HAVING_CLAUSE:
			if v, ok := c.(*cmn.HavingClause); ok {
				stmt.Having = v
			}
		case cmn.ORDER_BY_CLAUSE:
			if v, ok := c.(*cmn.OrderByClause); ok {
				stmt.OrderBy = v
			}
		case cmn.LIMIT_CLAUSE:
			if v, ok := c.(*cmn.LimitClause); ok {
				stmt.Limit = v
			}
		case cmn.OFFSET_CLAUSE:
			if v, ok := c.(*cmn.OffsetClause); ok {
				stmt.Offset = v
			}
		case cmn.FOR_CLAUSE:
			if v, ok := c.(*cmn.ForClause); ok {
				stmt.For = v
			}
		}
	}
}

func assignInsertStatementFields(stmt *cmn.InsertIntoStatement, clauses []cmn.ClauseNode) {
	for _, c := range clauses {
		switch c.Type() {
		case cmn.INSERT_INTO_CLAUSE:
			// TableNameはINSERT_INTO_CLAUSEから取得
			if v, ok := c.(*cmn.InsertIntoClause); ok {
				stmt.InsertInto = v
			}
		case cmn.VALUES_CLAUSE:
			if v, ok := c.(*cmn.ValuesClause); ok {
				stmt.ValuesList = v
			}
		case cmn.SELECT_CLAUSE:
			if v, ok := c.(*cmn.SelectClause); ok {
				stmt.Select = v
			}
		case cmn.FROM_CLAUSE:
			if v, ok := c.(*cmn.FromClause); ok {
				stmt.From = v
			}
		case cmn.WHERE_CLAUSE:
			if v, ok := c.(*cmn.WhereClause); ok {
				stmt.Where = v
			}
		case cmn.GROUP_BY_CLAUSE:
			if v, ok := c.(*cmn.GroupByClause); ok {
				stmt.GroupBy = v
			}
		case cmn.HAVING_CLAUSE:
			if v, ok := c.(*cmn.HavingClause); ok {
				stmt.Having = v
			}
		case cmn.ORDER_BY_CLAUSE:
			if v, ok := c.(*cmn.OrderByClause); ok {
				stmt.OrderBy = v
			}
		case cmn.LIMIT_CLAUSE:
			if v, ok := c.(*cmn.LimitClause); ok {
				stmt.Limit = v
			}
		case cmn.OFFSET_CLAUSE:
			if v, ok := c.(*cmn.OffsetClause); ok {
				stmt.Offset = v
			}
		case cmn.ON_CONFLICT_CLAUSE:
			if v, ok := c.(*cmn.OnConflictClause); ok {
				stmt.OnConflict = v
			}
		case cmn.RETURNING_CLAUSE:
			if v, ok := c.(*cmn.ReturningClause); ok {
				stmt.Returning = v
			}
		}
	}
}

func assignUpdateStatementFields(stmt *cmn.UpdateStatement, clauses []cmn.ClauseNode) {
	for _, c := range clauses {
		switch c.Type() {
		case cmn.UPDATE_CLAUSE:
			if v, ok := c.(*cmn.UpdateClause); ok {
				stmt.Table = v.TableName
			}
		case cmn.SET_CLAUSE:
			if v, ok := c.(*cmn.SetClause); ok {
				stmt.Sets = append(stmt.Sets, *v)
			}
		case cmn.WHERE_CLAUSE:
			if v, ok := c.(*cmn.WhereClause); ok {
				stmt.Where = v
			}
		case cmn.RETURNING_CLAUSE:
			if v, ok := c.(*cmn.ReturningClause); ok {
				stmt.Returning = v
			}
		}
	}
}

func assignDeleteStatementFields(stmt *cmn.DeleteFromStatement, clauses []cmn.ClauseNode) {
	for _, c := range clauses {
		switch c.Type() {
		case cmn.DELETE_FROM_CLAUSE:
			if v, ok := c.(*cmn.DeleteFromClause); ok {
				stmt.Table = v.TableName
			}
		case cmn.WHERE_CLAUSE:
			if v, ok := c.(*cmn.WhereClause); ok {
				stmt.Where = v
			}
		case cmn.RETURNING_CLAUSE:
			if v, ok := c.(*cmn.ReturningClause); ok {
				stmt.Returning = v
			}
		}
	}
}
