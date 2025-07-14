package parserstep4

import (
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Execute runs clause-level validation for parserstep4
// Returns *ParseError (nil if no error)
func Execute(stmt cmn.StatementNode) *cmn.ParseError {
	perr := &cmn.ParseError{}

	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		FinalizeSelectClause(s.Select, perr)
		FinalizeFromClause(s.From, perr)
		if s.With != nil {
			EmptyCheck(s.With, perr)
		}
		if s.Where != nil {
			EmptyCheck(s.Where, perr)
		}
		if s.GroupBy != nil {
			FinalizeGroupByClause(s.GroupBy, perr)
		}
		if s.Having != nil {
			FinalizeHavingClause(s.Having, s.GroupBy, perr)
		}
		if s.OrderBy != nil {
			FinalizeOrderByClause(s.OrderBy, perr)
		}
		if s.Limit != nil || s.Offset != nil {
			FinalizeLimitOffsetClause(s.Limit, s.Offset, perr)
		}
		if s.For != nil {
			EmptyCheck(s.For, perr)
		}
	case *cmn.InsertIntoStatement:
		FinalizeInsertIntoClause(s.Into, s.Select, perr)
		if s.With != nil {
			EmptyCheck(s.With, perr)
		}
		if s.Select != nil {
			FinalizeSelectClause(s.Select, perr)
			FinalizeFromClause(s.From, perr)
			if s.Where != nil {
				EmptyCheck(s.Where, perr)
			}
			if s.GroupBy != nil {
				FinalizeGroupByClause(s.GroupBy, perr)
			}
			if s.Having != nil {
				FinalizeHavingClause(s.Having, s.GroupBy, perr)
			}
			if s.OrderBy != nil {
				FinalizeOrderByClause(s.OrderBy, perr)
			}
			if s.Limit != nil || s.Offset != nil {
				FinalizeLimitOffsetClause(s.Limit, s.Offset, perr)
			}
		}
		if s.Returning != nil {
			EmptyCheck(s.Returning, perr)
		}
	case *cmn.UpdateStatement:
		FinalizeUpdateClause(s.Update, perr)
		FinalizeSetClause(s.Set, perr)
		if s.Where != nil {
			EmptyCheck(s.Where, perr)
		}
		if s.Returning != nil {
			EmptyCheck(s.Returning, perr)
		}
	case *cmn.DeleteFromStatement:
		FinalizeDeleteFromClause(s.From, perr)
		if s.Where != nil {
			EmptyCheck(s.Where, perr)
		}
		if s.Returning != nil {
			EmptyCheck(s.Returning, perr)
		}
	}
	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}
