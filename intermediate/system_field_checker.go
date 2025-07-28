package intermediate

import (
	"fmt"
	"slices"
	"strings"

	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
)

// StatementTypeProvider is a minimal interface for getting statement type
type StatementTypeProvider interface {
	Type() parsercommon.NodeType
}

// CheckSystemFields validates system fields configuration and returns implicit parameters
// Errors are accumulated in the provided GenerateError
func CheckSystemFields(stmt StatementTypeProvider, config *Config, parameters []Parameter, gerr *parserstep5.GenerateError) []ImplicitParameter {
	if config == nil {
		return nil
	}

	// Create a map of existing parameters for quick lookup
	paramMap := make(map[string]bool)
	for _, param := range parameters {
		paramMap[param.Name] = true
	}

	// Determine statement type
	stmtType := getStatementType(stmt)

	// For INSERT statements, extract existing column names for explicit field validation
	var existingColumns map[string]bool
	if stmtType == "INSERT" {
		if insertStmt, ok := stmt.(*parsercommon.InsertIntoStatement); ok {
			existingColumns = extractInsertColumnNames(insertStmt)
		}
	}

	var implicitParams []ImplicitParameter

	// Check each system field configured for the current operation
	for _, field := range config.System.Fields {
		var operation *SystemFieldOperation
		var operationName string

		switch stmtType {
		case "INSERT":
			if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
				operation = &field.OnInsert
				operationName = "INSERT"
			}
		case "UPDATE":
			if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
				operation = &field.OnUpdate
				operationName = "UPDATE"
			}
		default:
			// SELECT, DELETE, etc. don't need system field validation
			continue
		}

		if operation == nil {
			continue
		}

		// Perform validation logic with column existence check for INSERT
		implicitParam := checkSystemFieldWithColumns(field, operation, operationName, paramMap, existingColumns, gerr)
		if implicitParam != nil {
			implicitParams = append(implicitParams, *implicitParam)
		}
	}

	return implicitParams
}

// checkSystemFieldWithColumns performs validation for a single system field with column existence check
func checkSystemFieldWithColumns(field SystemField, operation *SystemFieldOperation, operationName string, paramMap map[string]bool, existingColumns map[string]bool, gerr *parserstep5.GenerateError) *ImplicitParameter {
	// Handle parameter configuration
	switch operation.Parameter {
	case ParameterExplicit:
		// Check if explicit parameter is provided
		if !paramMap[field.Name] {
			gerr.AddError(fmt.Errorf("%s statement requires explicit parameter '%s' but it was not provided", operationName, field.Name))
			return nil
		}

		// For INSERT statements, also check if the field exists in column list
		if operationName == "INSERT" && existingColumns != nil {
			if !existingColumns[field.Name] {
				gerr.AddError(fmt.Errorf("%s statement requires explicit system field '%s' to be included in column list", operationName, field.Name))
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
			gerr.AddError(fmt.Errorf("%s statement should not include parameter '%s' (configured as error)", operationName, field.Name))
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
func extractInsertColumnNames(stmt *parsercommon.InsertIntoStatement) map[string]bool {
	columns := make(map[string]bool)

	// First try the Columns field
	for _, column := range stmt.Columns {
		columns[column.Name] = true
	}

	// If Columns field is empty, extract from clauses
	if len(columns) == 0 {
		for _, clause := range stmt.Clauses() {
			if clause.Type() == parsercommon.INSERT_INTO_CLAUSE {
				// Extract column names from INSERT INTO clause tokens
				tokens := clause.RawTokens()
				inParentheses := false
				for _, token := range tokens {
					if token.Type == tokenizer.OPENED_PARENS {
						inParentheses = true
						continue
					}
					if token.Type == tokenizer.CLOSED_PARENS {
						inParentheses = false
						continue
					}
					if inParentheses && token.Type == tokenizer.IDENTIFIER {
						// Skip keywords
						if !isKeywordOrTableName(token.Value) {
							columns[token.Value] = true
						}
					}
				}
			}
		}
	}

	return columns
}

// isKeywordOrTableName checks if a token value is a keyword or table name (not a column)
func isKeywordOrTableName(value string) bool {
	keywords := []string{"INSERT", "INTO", "VALUES", "SELECT", "FROM", "WHERE", "ORDER", "BY", "LIMIT", "OFFSET", "users"}
	upperValue := strings.ToUpper(value)
	return slices.Contains(keywords, upperValue)
}

// getStatementType determines the type of SQL statement
func getStatementType(stmt StatementTypeProvider) string {
	// Use the same approach as response_affinity.go
	switch stmt.Type() {
	case parsercommon.SELECT_STATEMENT:
		return "SELECT"
	case parsercommon.INSERT_INTO_STATEMENT:
		return "INSERT"
	case parsercommon.UPDATE_STATEMENT:
		return "UPDATE"
	case parsercommon.DELETE_FROM_STATEMENT:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}
