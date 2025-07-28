package intermediate

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// AddSystemFieldsToInsert adds system fields to INSERT statement's column list and VALUES clause
// for each implicit parameter
func AddSystemFieldsToInsert(stmt parsercommon.StatementNode, implicitParams []ImplicitParameter) error {
	if len(implicitParams) == 0 {
		return nil
	}

	// Check if this is an INSERT statement
	if stmt.Type() != parsercommon.INSERT_INTO_STATEMENT {
		return nil // Only process INSERT statements
	}

	// Cast to InsertIntoStatement to access clauses
	insertStmt, ok := stmt.(*parsercommon.InsertIntoStatement)
	if !ok {
		return fmt.Errorf("failed to cast statement to InsertIntoStatement")
	}

	// Extract existing column names from Columns field
	existingColumns := make(map[string]bool)
	for _, column := range insertStmt.Columns {
		existingColumns[column.Name] = true
	}

	// Determine which implicit parameters need to be added
	var columnsToAdd []string
	for _, param := range implicitParams {
		if !existingColumns[param.Name] {
			columnsToAdd = append(columnsToAdd, param.Name)
		} else {
		}
	}

	if len(columnsToAdd) == 0 {
		return nil
	}

	// Add columns to Columns field
	for _, columnName := range columnsToAdd {
		insertStmt.Columns = append(insertStmt.Columns, parsercommon.FieldName{Name: columnName})
	}

	// Add EMIT_SYSTEM_VALUE tokens to VALUES clause
	if insertStmt.ValuesList != nil {
		if err := addSystemValuesToValuesClause(insertStmt.ValuesList, columnsToAdd); err != nil {
			return fmt.Errorf("failed to add system values to VALUES clause: %w", err)
		}
	}

	return nil
}

// getColumnNames extracts column names from FieldName slice for debugging
func getColumnNames(columns []parsercommon.FieldName) []string {
	var names []string
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return names
}

// addSystemValuesToValuesClause adds EMIT_SYSTEM_VALUE tokens to VALUES clause
func addSystemValuesToValuesClause(valuesClause *parsercommon.ValuesClause, columnsToAdd []string) error {
	tokens := valuesClause.RawTokens()

	// Find the position to insert new values (before the closing parenthesis)
	var insertPosition int = -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type == tokenizer.CLOSED_PARENS {
			insertPosition = i
			break
		}
	}

	if insertPosition == -1 {
		return fmt.Errorf("could not find closing parenthesis in VALUES clause")
	}

	// Create new tokens for the system values
	var newTokens []tokenizer.Token
	for _, column := range columnsToAdd {
		// Add comma and space before each new value
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.COMMA,
			Value: ",",
		})
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.WHITESPACE,
			Value: " ",
		})
		// Add EMIT_SYSTEM_VALUE token
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.BLOCK_COMMENT,
			Value: fmt.Sprintf("/*# EMIT_SYSTEM_VALUE: %s */", column),
		})
	}

	// Insert new tokens before the closing parenthesis
	for range newTokens {
	}

	return nil
}
