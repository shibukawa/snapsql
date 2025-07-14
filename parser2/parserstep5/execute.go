package parserstep5

import (
	"github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Execute is the entry point for parserstep5.
// It runs parserstep3 and parserstep4 first, then applies dummy detection and implicit if conditions.
// Returns the processed statement and any errors from previous steps.
func Execute(statement parsercommon.StatementNode) (parsercommon.StatementNode, error) {
	// Apply parserstep5 processing
	// Apply dummy detection
	DetectDummyRanges(statement)

	// Apply implicit if conditions for LIMIT and OFFSET clauses
	ApplyImplicitIfConditions(statement)

	return statement, nil
}
