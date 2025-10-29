package parserstep6

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute is the entry point for parserstep6.
// It takes a parsed statement, parameter namespace, and constant namespace,
// and validates template variables and directives.
// Execute runs parserstep6 validations in strict mode.
func Execute(statement cmn.StatementNode, paramNamespace *cmn.Namespace, constNamespace *cmn.Namespace) (map[string]any, *cmn.ParseError) {
	return ExecuteWithOptions(statement, paramNamespace, constNamespace, false)
}

// ExecuteWithOptions runs parserstep6 with optional relaxed behavior for inspect mode.
// When inspectMode is true, variable/directive validations are skipped.
func ExecuteWithOptions(statement cmn.StatementNode, paramNamespace *cmn.Namespace, constNamespace *cmn.Namespace, inspectMode bool) (map[string]any, *cmn.ParseError) {
	// Validate template variables and directives using both namespaces
	perr := &cmn.ParseError{}
	typeInfo := make(map[string]any)

	if !inspectMode {
		validateVariables(statement, paramNamespace, constNamespace, perr, typeInfo)
	}

	if len(perr.Errors) > 0 {
		return typeInfo, perr
	}

	return typeInfo, nil
}
