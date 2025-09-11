package parserstep5

import (
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute is the entry point for parserstep5.
// It runs parserstep3 and parserstep4 first, then applies dummy detection and implicit if conditions.
// Returns the processed statement and any errors from previous steps.
// Execute runs parserstep5 in strict mode (default behavior).
func Execute(statement parsercommon.StatementNode, functionDef *parsercommon.FunctionDefinition) error {
	return ExecuteWithOptions(statement, functionDef, false)
}

// ExecuteWithOptions runs parserstep5 with optional relaxed behavior for inspect mode.
// When inspectMode is true, directive validations and transformations that depend on
// FunctionDefinition or runtime evaluation are skipped to allow structural parsing only.
func ExecuteWithOptions(statement parsercommon.StatementNode, functionDef *parsercommon.FunctionDefinition, inspectMode bool) error {
	// Create ParseError to collect all generation errors
	gerr := &parsercommon.ParseError{}

	if !inspectMode {
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
		if len(gerr.Errors) > 0 {
			return gerr
		}

		// Then check for parse errors
		if len(perr.Errors) > 0 {
			return perr
		}

		return nil
	}

	// InspectMode: skip heavy validations/transformations to avoid FD/CEL dependency
	return nil
}
