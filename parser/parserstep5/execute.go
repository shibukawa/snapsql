package parserstep5

import (
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute is the entry point for parserstep5.
// It runs parserstep3 and parserstep4 first, then applies dummy detection and implicit if conditions.
// Returns the processed statement and any errors from previous steps.
func Execute(statement parsercommon.StatementNode, functionDef *parsercommon.FunctionDefinition) error {
	// Create GenerateError to collect all generation errors
	gerr := &GenerateError{}

	// Apply parserstep5 processing
	// Apply array expansion for VALUES clauses
	expandArraysInValues(statement, functionDef, gerr)
	// Apply dummy detection
	detectDummyRanges(statement)
	// Apply implicit if conditions for LIMIT and OFFSET clauses
	applyImplicitIfConditions(statement)

	perr := &parsercommon.ParseError{}
	validateAndLinkDirectives(statement, perr)

	// Check for generation errors first
	if gerr.HasErrors() {
		return gerr
	}

	// Then check for parse errors
	if len(perr.Errors) > 0 {
		return perr
	}

	return nil
}
