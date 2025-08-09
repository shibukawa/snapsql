package intermediate

import (
	"fmt"

	. "github.com/shibukawa/snapsql"
	snapsql "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// SystemFieldProcessor handles system field validation and processing
type SystemFieldProcessor struct{}

func (s *SystemFieldProcessor) Name() string {
	return "SystemFieldProcessor"
}

func (s *SystemFieldProcessor) Process(ctx *ProcessingContext) error {
	// Extract system fields information from config
	if ctx.Config != nil {
		ctx.SystemFields = extractSystemFieldsInfo(ctx.Config, ctx.Statement)

		// Perform system field validation and get implicit parameters
		systemFieldErr := &GenerateError{}
		ctx.ImplicitParams = CheckSystemFields(ctx.Statement, ctx.Config, ctx.Parameters, systemFieldErr)

		// Check if there were any system field validation errors
		if systemFieldErr.HasErrors() {
			return fmt.Errorf("system field validation failed: %w", systemFieldErr)
		}
	}

	return nil
}

type StatementTypeProvider interface {
	Type() parser.NodeType
}

// CheckSystemFields validates system fields configuration and returns implicit parameters
// Errors are accumulated in the provided GenerateError
func CheckSystemFields(stmt StatementTypeProvider, config *Config, parameters []Parameter, gerr *GenerateError) []ImplicitParameter {
	if config == nil {
		return nil
	}

	// Create a map of existing parameters for quick lookup
	paramMap := make(map[string]bool)
	for _, param := range parameters {
		paramMap[param.Name] = true
	}

	// For INSERT statements, extract existing column names for explicit field validation
	var existingColumns map[string]bool

	if stmt.Type() == parser.SELECT_STATEMENT {
		if insertStmt, ok := stmt.(*parser.InsertIntoStatement); ok {
			existingColumns = extractInsertColumnNames(insertStmt)
		}
	}

	var implicitParams []ImplicitParameter

	// Check each system field configured for the current operation
	for _, field := range config.System.Fields {
		var operation *SystemFieldOperation

		switch stmt.Type() {
		case parser.INSERT_INTO_STATEMENT:
			if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
				operation = &field.OnInsert
			}
		case parser.UPDATE_STATEMENT:
			if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
				operation = &field.OnUpdate
			}
		default:
			// SELECT, DELETE, etc. don't need system field validation
			continue
		}

		if operation == nil {
			continue
		}

		// Perform validation logic with column existence check for INSERT
		implicitParam := checkSystemFieldWithColumns(field, operation, stmt.Type(), paramMap, existingColumns, gerr)
		if implicitParam != nil {
			implicitParams = append(implicitParams, *implicitParam)
		}
	}

	return implicitParams
}

// checkSystemFieldWithColumns performs validation for a single system field with column existence check
func checkSystemFieldWithColumns(field SystemField, operation *SystemFieldOperation, nodeType parser.NodeType, paramMap map[string]bool, existingColumns map[string]bool, gerr *GenerateError) *ImplicitParameter {
	// Handle parameter configuration
	switch operation.Parameter {
	case ParameterExplicit:
		// Check if explicit parameter is provided
		if !paramMap[field.Name] {
			gerr.AddError(fmt.Errorf("%w: %s statement parameter '%s'", snapsql.ErrParameterNotProvided, nodeType.String(), field.Name))
			return nil
		}

		// For INSERT statements, also check if the field exists in column list
		if nodeType == parser.INSERT_INTO_STATEMENT && existingColumns != nil {
			if !existingColumns[field.Name] {
				gerr.AddError(fmt.Errorf("%w: %s statement field '%s'", snapsql.ErrSystemFieldNotIncluded, nodeType.String(), field.Name))
				return nil
			}
		}

		// Explicit parameter provided and column exists, no implicit parameter needed
		return nil

	case ParameterImplicit:
		// Add to implicit parameters list
		implicitParam := &ImplicitParameter{
			Name: field.Name,
			Type: field.Type,
		}

		// Add default value if specified
		if operation.Default != nil {
			implicitParam.Default = operation.Default
		}

		return implicitParam

	case ParameterError:
		// Check if parameter is provided (should cause error)
		if paramMap[field.Name] {
			gerr.AddError(fmt.Errorf("%w: %s statement parameter '%s'", snapsql.ErrParameterConfiguredError, nodeType.String(), field.Name))
			return nil
		}
		// No parameter provided, no implicit parameter needed
		return nil

	default:
		// For fields without explicit parameter configuration,
		// if there's a default value and the parameter is not provided,
		// it should be added as implicit parameter
		if operation.Default != nil && !paramMap[field.Name] {
			return &ImplicitParameter{
				Name:    field.Name,
				Type:    field.Type,
				Default: operation.Default,
			}
		}
	}

	return nil
}

// extractInsertColumnNames extracts column names from INSERT statement
func extractInsertColumnNames(stmt *parser.InsertIntoStatement) map[string]bool {
	columns := make(map[string]bool)

	// First try the Columns field
	for _, column := range stmt.Columns {
		columns[column.Name] = true
	}

	// If Columns field is empty, extract from clauses
	if len(columns) == 0 {
		for _, clause := range stmt.Clauses() {
			if clause.Type() == parser.INSERT_INTO_CLAUSE {
				// Extract column names from INSERT INTO clause tokens
				tokens := clause.RawTokens()
				inParentheses := false

				for _, token := range tokens {
					if token.Type == tok.OPENED_PARENS {
						inParentheses = true
						continue
					}

					if token.Type == tok.CLOSED_PARENS {
						inParentheses = false
						continue
					}

					if inParentheses && token.Type == tok.IDENTIFIER {
						// All IDENTIFIER tokens in column list are column names
						columns[token.Value] = true
					}
				}
			}
		}
	}

	return columns
}

// AddSystemFieldsToInsert adds system fields to INSERT statement's column list and VALUES clause
// for each implicit parameter
func AddSystemFieldsToInsert(stmt parser.StatementNode, implicitParams []ImplicitParameter) error {
	if len(implicitParams) == 0 {
		return nil
	}

	// Check if this is an INSERT statement
	if stmt.Type() != parser.INSERT_INTO_STATEMENT {
		return nil // Only process INSERT statements
	}

	// Cast to InsertIntoStatement to access clauses
	insertStmt, ok := stmt.(*parser.InsertIntoStatement)
	if !ok {
		return fmt.Errorf("%w: InsertIntoStatement", snapsql.ErrInvalidStatementCast)
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

	if len(columnsToAdd) == 0 {
		return nil
	}

	// Add columns to Columns field
	for _, columnName := range columnsToAdd {
		insertStmt.Columns = append(insertStmt.Columns, parser.FieldName{Name: columnName})
	}

	// Update InsertIntoClause tokens to include new columns
	if insertStmt.Into != nil {
		err := addSystemColumnsToInsertIntoClause(insertStmt.Into, columnsToAdd)
		if err != nil {
			return fmt.Errorf("failed to add system columns to InsertIntoClause: %w", err)
		}
	}

	// Add EMIT_SYSTEM_VALUE tokens to VALUES clause
	if insertStmt.ValuesList != nil {
		err := addSystemValuesToValuesClause(insertStmt.ValuesList, columnsToAdd)
		if err != nil {
			return fmt.Errorf("failed to add system values to VALUES clause: %w", err)
		}
	}

	return nil
}

// addSystemValuesToValuesClause adds EMIT_SYSTEM_VALUE tokens to VALUES clause
func addSystemValuesToValuesClause(valuesClause *parser.ValuesClause, columnsToAdd []string) error {
	tokens := valuesClause.RawTokens()

	// Find the position to insert new values (before the closing parenthesis)
	var insertPosition = -1

	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type == tok.CLOSED_PARENS {
			insertPosition = i
			break
		}
	}

	if insertPosition == -1 {
		return fmt.Errorf("%w: VALUES clause", snapsql.ErrClosingParenthesisNotFound)
	}

	// Create new tokens for the system values
	var newTokens []tok.Token
	for _, column := range columnsToAdd {
		// Add comma and space before each new value
		newTokens = append(newTokens, tok.Token{
			Type:  tok.COMMA,
			Value: ",",
		})
		newTokens = append(newTokens, tok.Token{
			Type:  tok.WHITESPACE,
			Value: " ",
		})
		// Add EMIT_SYSTEM_VALUE token
		newTokens = append(newTokens, tok.Token{
			Type:  tok.BLOCK_COMMENT,
			Value: fmt.Sprintf("/*# EMIT_SYSTEM_VALUE: %s */", column),
			Directive: &tok.Directive{
				Type:        "system_value",
				SystemField: column,
			},
		})
	}

	// Insert new tokens before the closing parenthesis
	for range newTokens {
	}

	return nil
}

// addSystemColumnsToInsertIntoClause adds system columns to InsertIntoClause tokens
func addSystemColumnsToInsertIntoClause(insertIntoClause *parser.InsertIntoClause, columnsToAdd []string) error {
	tokens := insertIntoClause.RawTokens()

	// Find the position to insert new columns (before the closing parenthesis)
	var insertPosition = -1

	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type == tok.CLOSED_PARENS {
			insertPosition = i
			break
		}
	}

	if insertPosition == -1 {
		return fmt.Errorf("%w: InsertIntoClause", snapsql.ErrClosingParenthesisNotFound)
	}

	// Create new tokens for the system columns
	var newTokens []tok.Token
	for _, column := range columnsToAdd {
		// Add comma and space before each new column
		newTokens = append(newTokens, tok.Token{
			Type:  tok.COMMA,
			Value: ",",
		})
		newTokens = append(newTokens, tok.Token{
			Type:  tok.WHITESPACE,
			Value: " ",
		})
		// Add the column name
		newTokens = append(newTokens, tok.Token{
			Type:  tok.IDENTIFIER,
			Value: column,
		})
	}

	// Insert new tokens before the closing parenthesis
	updatedTokens := make([]tok.Token, 0, len(tokens)+len(newTokens))
	updatedTokens = append(updatedTokens, tokens[:insertPosition]...)
	updatedTokens = append(updatedTokens, newTokens...)
	updatedTokens = append(updatedTokens, tokens[insertPosition:]...)

	// TODO: Update the InsertIntoClause tokens in pipeline approach
	// insertIntoClause.SetRawTokens(updatedTokens)
	_ = updatedTokens // Will be used when TODO above is implemented

	// TODO: Update the InsertIntoClause tokens in pipeline approach
	// insertIntoClause.SetRawTokens(updatedTokens)

	// Also update the Columns field in InsertIntoClause
	insertIntoClause.Columns = append(insertIntoClause.Columns, columnsToAdd...)

	return nil
}

// AddSystemFieldsToUpdate adds EMIT_SYSTEM_VALUE calls to UPDATE statement's SET clause
// for each implicit parameter
func AddSystemFieldsToUpdate(stmt parser.StatementNode, implicitParams []ImplicitParameter) error {
	if len(implicitParams) == 0 {
		return nil
	}

	// Check if this is an UPDATE statement
	if stmt.Type() != parser.UPDATE_STATEMENT {
		return nil // Only process UPDATE statements
	}

	// Cast to UpdateStatement to access SET clause
	updateStmt, ok := stmt.(*parser.UpdateStatement)
	if !ok {
		return fmt.Errorf("%w: UpdateStatement", snapsql.ErrInvalidStatementCast)
	}

	// Find the SET clause
	if updateStmt.Set == nil {
		return fmt.Errorf("%w: SET clause", snapsql.ErrMissingClause)
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
func addSystemValueToSetClause(setClause *parser.SetClause, columnName string) error {
	// Create tokens for EMIT_SYSTEM_VALUE(column_name)
	valueTokens := []tok.Token{
		{Type: tok.IDENTIFIER, Value: "EMIT_SYSTEM_VALUE"},
		{Type: tok.OPENED_PARENS, Value: "("},
		{Type: tok.IDENTIFIER, Value: columnName},
		{Type: tok.CLOSED_PARENS, Value: ")"},
	}

	// Create a new SetAssign for the system field
	systemAssign := parser.SetAssign{
		FieldName: columnName,
		Value:     valueTokens,
	}

	// Add to existing assignments
	setClause.Assigns = append(setClause.Assigns, systemAssign)

	return nil
}

// GetSystemFieldAssignments returns the system field assignments as SetAssign slice
// This can be used to append to existing SET clause assignments
func GetSystemFieldAssignments(implicitParams []ImplicitParameter) []parser.SetAssign {
	var assignments []parser.SetAssign

	for _, param := range implicitParams {
		// Create tokens for EMIT_SYSTEM_VALUE(column_name)
		valueTokens := []tok.Token{
			{Type: tok.IDENTIFIER, Value: "EMIT_SYSTEM_VALUE"},
			{Type: tok.OPENED_PARENS, Value: "("},
			{Type: tok.IDENTIFIER, Value: param.Name},
			{Type: tok.CLOSED_PARENS, Value: ")"},
		}

		assignment := parser.SetAssign{
			FieldName: param.Name,
			Value:     valueTokens,
		}

		assignments = append(assignments, assignment)
	}

	return assignments
}
