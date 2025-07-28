package parserstep4

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute runs clause-level validation for parserstep4
// Returns *ParseError (nil if no error)
func Execute(stmt cmn.StatementNode) error {
	perr := &cmn.ParseError{}

	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		finalizeSelectClause(s.Select, perr)

		finalizeFromClause(s.From, perr)
		if s.With != nil {
			emptyCheck(s.With, perr)
		}
		if s.Where != nil {
			emptyCheck(s.Where, perr)
		}
		if s.GroupBy != nil {
			finalizeGroupByClause(s.GroupBy, perr)
		}
		if s.Having != nil {
			finalizeHavingClause(s.Having, s.GroupBy, perr)
		}
		if s.OrderBy != nil {
			finalizeOrderByClause(s.OrderBy, perr)
		}
		if s.Limit != nil {
			finalizeLimitOffsetClause(s.Limit, s.Offset, perr)
		} else if s.Offset != nil {
			// OFFSET without LIMIT: validate OFFSET clause individually
			finalizeOffsetClause(s.Offset, perr)
		}
		if s.For != nil {
			emptyCheck(s.For, perr)
		}
	case *cmn.InsertIntoStatement:
		finalizeInsertIntoClause(s.Into, s.Select, perr)
		
		// Copy columns from InsertIntoClause to InsertIntoStatement
		if s.Into != nil {
			s.Columns = make([]cmn.FieldName, len(s.Into.Columns))
			for i, columnName := range s.Into.Columns {
				s.Columns[i] = cmn.FieldName{Name: columnName}
			}
		}
		
		if s.With != nil {
			emptyCheck(s.With, perr)
		}
		if s.Select != nil {
			finalizeSelectClause(s.Select, perr)
			finalizeFromClause(s.From, perr)
			if s.Where != nil {
				emptyCheck(s.Where, perr)
			}
			if s.GroupBy != nil {
				finalizeGroupByClause(s.GroupBy, perr)
			}
			if s.Having != nil {
				finalizeHavingClause(s.Having, s.GroupBy, perr)
			}
			if s.OrderBy != nil {
				finalizeOrderByClause(s.OrderBy, perr)
			}
			if s.Limit != nil {
				finalizeLimitOffsetClause(s.Limit, s.Offset, perr)
			} else if s.Offset != nil {
				// OFFSET without LIMIT: validate OFFSET clause individually
				finalizeOffsetClause(s.Offset, perr)
			}
		}
		if s.Returning != nil {
			emptyCheck(s.Returning, perr)
			finalizeReturningClause(s.Returning, perr)
		}
	case *cmn.UpdateStatement:
		finalizeUpdateClause(s.Update, perr)
		finalizeSetClause(s.Set, perr)
		if s.Where != nil {
			emptyCheck(s.Where, perr)
		}
		if s.Returning != nil {
			emptyCheck(s.Returning, perr)
			finalizeReturningClause(s.Returning, perr)
		}
	case *cmn.DeleteFromStatement:
		finalizeDeleteFromClause(s.From, perr)
		if s.Where != nil {
			emptyCheck(s.Where, perr)
		}
		if s.Returning != nil {
			emptyCheck(s.Returning, perr)
			finalizeReturningClause(s.Returning, perr)
		}
	}
	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}
