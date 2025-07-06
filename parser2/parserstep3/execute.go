package parserstep3

import (
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

func Execute(stmt cmn.StatementNode) error {
	perr := &cmn.ParseError{}
	clauses := ValidateClausePresence(stmt.Type(), stmt.Clauses(), perr)
	ValidateClauseDuplicates(clauses, perr)
	ValidateClauseRequired(stmt.Type(), clauses, perr)
	ValidateClauseOrder(stmt.Type(), clauses, perr)
	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}
