package parserstep6

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute is the entry point for parserstep6.
// It takes a parsed statement, parameter namespace, and constant namespace,
// and validates template variables and directives.
func Execute(statement cmn.StatementNode, paramNamespace *cmn.Namespace, constNamespace *cmn.Namespace) *cmn.ParseError {
	// Validate template variables and directives using both namespaces
	perr := &cmn.ParseError{}

	validateVariables(statement, paramNamespace, constNamespace, perr)

	if len(perr.Errors) > 0 {
		return perr
	}

	return nil
}
