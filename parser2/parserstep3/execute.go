package parserstep3

import (
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Execute runs all clause checks (duplicate, required, order) for a statement node.
// Returns error if any check fails.
func Execute(stmt cmn.StatementNode) error {
	clauses := stmt.Clauses()
	if err := ValidateClauseDuplicates(clauses); err != nil {
		return err
	}
	if err := ValidateClauseRequired(stmt.Type(), clauses); err != nil {
		return err
	}
	if err := ValidateClauseOrder(stmt.Type(), clauses); err != nil {
		return err
	}
	return nil
}
