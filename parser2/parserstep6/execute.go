package parserstep6

import (
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
)

// Execute is the entry point for parserstep6.
// It takes a parsed statement and namespace, applies all previous steps (parserstep3-5),
// and validates template variables and directives.
func Execute(statement cmn.StatementNode, namespace *cmn.Namespace) *cmn.ParseError {
	// Validate template variables and directives
	perr := &cmn.ParseError{}
	ValidateVariables(statement, namespace, perr)

	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}
