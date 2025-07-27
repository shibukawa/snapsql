package intermediate

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// AddSystemFieldsToUpdate adds EMIT_SYSTEM_VALUE calls to UPDATE statement's SET clause
// for each implicit parameter
func AddSystemFieldsToUpdate(stmt parsercommon.StatementNode, implicitParams []ImplicitParameter) error {
	if len(implicitParams) == 0 {
		return nil
	}

	// Check if this is an UPDATE statement
	if stmt.Type() != parsercommon.UPDATE_STATEMENT {
		return nil // Only process UPDATE statements
	}

	// Cast to UpdateStatement to access SET clause
	updateStmt, ok := stmt.(*parsercommon.UpdateStatement)
	if !ok {
		return fmt.Errorf("failed to cast statement to UpdateStatement")
	}

	// Find the SET clause
	if updateStmt.Set == nil {
		return fmt.Errorf("UPDATE statement has no SET clause")
	}

	// Add EMIT_SYSTEM_VALUE calls for each implicit parameter
	for _, param := range implicitParams {
		err := addSystemValueToSetClause(updateStmt.Set, param.Name)
		if err != nil {
			return fmt.Errorf("failed to add system value for %s: %w", param.Name, err)
		}
	}

	return nil
}

// addSystemValueToSetClause adds a system value assignment to the SET clause
func addSystemValueToSetClause(setClause *parsercommon.SetClause, columnName string) error {
	// Create tokens for EMIT_SYSTEM_VALUE(column_name)
	valueTokens := []tokenizer.Token{
		{Type: tokenizer.IDENTIFIER, Value: "EMIT_SYSTEM_VALUE"},
		{Type: tokenizer.OPENED_PARENS, Value: "("},
		{Type: tokenizer.IDENTIFIER, Value: columnName},
		{Type: tokenizer.CLOSED_PARENS, Value: ")"},
	}

	// Create a new SetAssign for the system field
	systemAssign := parsercommon.SetAssign{
		FieldName: columnName,
		Value:     valueTokens,
	}

	// Add to existing assignments
	setClause.Assigns = append(setClause.Assigns, systemAssign)

	return nil
}

// GetSystemFieldsSQL generates the SQL fragment for system fields
// This is a helper function to generate the actual SQL text for manual SQL construction
func GetSystemFieldsSQL(implicitParams []ImplicitParameter) string {
	if len(implicitParams) == 0 {
		return ""
	}

	var systemCalls []string
	for _, param := range implicitParams {
		systemCalls = append(systemCalls, fmt.Sprintf("EMIT_SYSTEM_VALUE(%s)", param.Name))
	}

	return ", " + strings.Join(systemCalls, ", ")
}

// GetSystemFieldAssignments returns the system field assignments as SetAssign slice
// This can be used to append to existing SET clause assignments
func GetSystemFieldAssignments(implicitParams []ImplicitParameter) []parsercommon.SetAssign {
	var assignments []parsercommon.SetAssign

	for _, param := range implicitParams {
		// Create tokens for EMIT_SYSTEM_VALUE(column_name)
		valueTokens := []tokenizer.Token{
			{Type: tokenizer.IDENTIFIER, Value: "EMIT_SYSTEM_VALUE"},
			{Type: tokenizer.OPENED_PARENS, Value: "("},
			{Type: tokenizer.IDENTIFIER, Value: param.Name},
			{Type: tokenizer.CLOSED_PARENS, Value: ")"},
		}

		assignment := parsercommon.SetAssign{
			FieldName: param.Name,
			Value:     valueTokens,
		}

		assignments = append(assignments, assignment)
	}

	return assignments
}
