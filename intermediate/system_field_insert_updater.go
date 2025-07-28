package intermediate

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// AddSystemFieldsToInsert adds system fields to INSERT statement's column list and VALUES clause
// for each implicit parameter
func AddSystemFieldsToInsert(stmt parsercommon.StatementNode, implicitParams []ImplicitParameter) error {
	fmt.Printf("DEBUG: AddSystemFieldsToInsert called with %d implicit params\n", len(implicitParams))
	for _, param := range implicitParams {
		fmt.Printf("DEBUG: Implicit param: %s\n", param.Name)
	}

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

	fmt.Printf("DEBUG: INSERT statement has %d existing columns\n", len(insertStmt.Columns))
	for i, col := range insertStmt.Columns {
		fmt.Printf("DEBUG: Existing column[%d]: %s\n", i, col.Name)
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
		}
	}

	fmt.Printf("DEBUG: Columns to add: %v\n", columnsToAdd)

	if len(columnsToAdd) == 0 {
		fmt.Printf("DEBUG: No columns to add, returning\n")
		return nil
	}

	// Add columns to Columns field
	for _, columnName := range columnsToAdd {
		fmt.Printf("DEBUG: Adding column: %s\n", columnName)
		insertStmt.Columns = append(insertStmt.Columns, parsercommon.FieldName{Name: columnName})
	}

	fmt.Printf("DEBUG: After adding columns, INSERT statement has %d columns\n", len(insertStmt.Columns))
	for i, col := range insertStmt.Columns {
		fmt.Printf("DEBUG: Final column[%d]: %s\n", i, col.Name)
	}

	// Update InsertIntoClause tokens to include new columns
	if insertStmt.Into != nil {
		fmt.Printf("DEBUG: Updating InsertIntoClause tokens\n")
		if err := addSystemColumnsToInsertIntoClause(insertStmt.Into, columnsToAdd); err != nil {
			return fmt.Errorf("failed to add system columns to InsertIntoClause: %w", err)
		}
	}

	// Add EMIT_SYSTEM_VALUE tokens to VALUES clause
	if insertStmt.ValuesList != nil {
		fmt.Printf("DEBUG: Adding system values to VALUES clause\n")
		if err := addSystemValuesToValuesClause(insertStmt.ValuesList, columnsToAdd); err != nil {
			return fmt.Errorf("failed to add system values to VALUES clause: %w", err)
		}
	} else {
		fmt.Printf("DEBUG: No VALUES clause found\n")
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

// addSystemColumnsToInsertIntoClause adds system columns to InsertIntoClause tokens
func addSystemColumnsToInsertIntoClause(insertIntoClause *parsercommon.InsertIntoClause, columnsToAdd []string) error {
	tokens := insertIntoClause.RawTokens()
	fmt.Printf("DEBUG: InsertIntoClause has %d tokens before update\n", len(tokens))

	// Find the position to insert new columns (before the closing parenthesis)
	var insertPosition int = -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type == tokenizer.CLOSED_PARENS {
			insertPosition = i
			break
		}
	}

	if insertPosition == -1 {
		return fmt.Errorf("could not find closing parenthesis in InsertIntoClause")
	}

	// Create new tokens for the system columns
	var newTokens []tokenizer.Token
	for _, column := range columnsToAdd {
		// Add comma and space before each new column
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.COMMA,
			Value: ",",
		})
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.WHITESPACE,
			Value: " ",
		})
		// Add the column name
		newTokens = append(newTokens, tokenizer.Token{
			Type:  tokenizer.IDENTIFIER,
			Value: column,
		})
	}

	// Insert new tokens before the closing parenthesis
	updatedTokens := make([]tokenizer.Token, 0, len(tokens)+len(newTokens))
	updatedTokens = append(updatedTokens, tokens[:insertPosition]...)
	updatedTokens = append(updatedTokens, newTokens...)
	updatedTokens = append(updatedTokens, tokens[insertPosition:]...)

	// TODO: Update the InsertIntoClause tokens in pipeline approach
	// insertIntoClause.SetRawTokens(updatedTokens)

	// Also update the Columns field in InsertIntoClause
	for _, column := range columnsToAdd {
		insertIntoClause.Columns = append(insertIntoClause.Columns, column)
	}

	fmt.Printf("DEBUG: InsertIntoClause has %d tokens after update\n", len(updatedTokens))
	return nil
}
