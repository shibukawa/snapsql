package parserstep6

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Execute is the entry point for parserstep6.
// It takes a parsed statement and namespace, replaces DUMMY_LITERAL tokens,
// and validates template variables and directives.
func Execute(statement cmn.StatementNode, namespace *cmn.Namespace) *cmn.ParseError {
	// Note: DUMMY_LITERAL replacement requires FunctionDefinition
	// For now, we skip replacement until ExecuteWithFunctionDef is called

	// Validate template variables and directives
	perr := &cmn.ParseError{}
	validateVariables(statement, namespace, perr)

	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}

// ExecuteWithFunctionDef is an extended entry point that includes FunctionDefinition
// for DUMMY_LITERAL token replacement
func ExecuteWithFunctionDef(statement cmn.StatementNode, namespace *cmn.Namespace, functionDef cmn.FunctionDefinition) *cmn.ParseError {
	// Step 1: Replace DUMMY_LITERAL tokens with actual literals
	perr := &cmn.ParseError{}
	replaceDummyLiterals(statement, namespace, functionDef, perr)

	// Step 2: Validate template variables and directives
	validateVariables(statement, namespace, perr)

	if len(perr.Errors) > 0 {
		return perr
	}
	return nil
}
