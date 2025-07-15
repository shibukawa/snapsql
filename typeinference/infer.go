package typeinference2

import (
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep7"
)

// InferFieldTypes performs type inference on SQL statement fields and returns inferred field information.
// This is the main entry point for type inference functionality.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//   - subqueryInfo: Optional subquery information from parserstep7 (can be nil)
//
// Returns:
//   - Slice of InferredFieldInfo containing type information for each field
//   - Error if type inference fails
//
// Supported statement types:
//   - SELECT statements: Returns type information for SELECT fields
//   - INSERT statements: Returns type information for RETURNING clause fields (if present)
//   - UPDATE statements: Returns type information for RETURNING clause fields (if present)
//   - DELETE statements: Returns type information for RETURNING clause fields (if present)
//
// For DML statements without RETURNING clause, returns a single field representing affected rows count.
func InferFieldTypes(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parsercommon.StatementNode,
	subqueryInfo *parserstep7.ParseResult,
) ([]*InferredFieldInfo, error) {
	// Create type inference engine
	engine := NewTypeInferenceEngine2(databaseSchemas, statementNode, subqueryInfo)

	// Perform unified type inference
	return engine.InferTypes()
}

// InferSelectFieldTypes performs type inference specifically for SELECT statement fields.
// This is a specialized entry point for SELECT-only type inference.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - selectStatement: Parsed SELECT statement AST
//   - subqueryInfo: Optional subquery information from parserstep7 (can be nil)
//
// Returns:
//   - Slice of InferredFieldInfo containing type information for each SELECT field
//   - Error if type inference fails or if statement is not a SELECT statement
func InferSelectFieldTypes(
	databaseSchemas []snapsql.DatabaseSchema,
	selectStatement *parsercommon.SelectStatement,
	subqueryInfo *parserstep7.ParseResult,
) ([]*InferredFieldInfo, error) {
	// Create type inference engine with SELECT statement
	engine := NewTypeInferenceEngine2(databaseSchemas, selectStatement, subqueryInfo)

	// Perform SELECT-specific type inference
	return engine.InferSelectTypes()
}

// InferFieldTypesSimple performs type inference without subquery information.
// This is a simplified entry point for basic type inference scenarios.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//
// Returns:
//   - Slice of InferredFieldInfo containing type information for each field
//   - Error if type inference fails
func InferFieldTypesSimple(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parsercommon.StatementNode,
) ([]*InferredFieldInfo, error) {
	return InferFieldTypes(databaseSchemas, statementNode, nil)
}

// InferFieldTypesWithOptions performs type inference with additional options and context.
// This is an advanced entry point that allows customization of the inference process.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//   - subqueryInfo: Optional subquery information from parserstep7 (can be nil)
//   - options: Additional inference options
//
// Returns:
//   - Slice of InferredFieldInfo containing type information for each field
//   - Error if type inference fails
func InferFieldTypesWithOptions(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parsercommon.StatementNode,
	subqueryInfo *parserstep7.ParseResult,
	options *InferenceOptions,
) ([]*InferredFieldInfo, error) {
	// Create type inference engine
	engine := NewTypeInferenceEngine2(databaseSchemas, statementNode, subqueryInfo)

	// Apply options if provided
	if options != nil {
		if options.Dialect != "" {
			engine.context.Dialect = options.Dialect
		}
		if options.TableAliases != nil {
			engine.context.TableAliases = options.TableAliases
		}
		if options.CurrentTables != nil {
			engine.context.CurrentTables = options.CurrentTables
		}
	}

	// Perform unified type inference
	return engine.InferTypes()
}

// InferenceOptions provides additional options for type inference
type InferenceOptions struct {
	Dialect       snapsql.Dialect   // Database dialect override
	TableAliases  map[string]string // Custom table alias mappings
	CurrentTables []string          // Custom available tables list
}

// ValidateStatementSchema validates a SQL statement against database schema.
// This function checks for schema validation errors without performing full type inference.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//   - subqueryInfo: Optional subquery information from parserstep7 (can be nil)
//
// Returns:
//   - Slice of ValidationError containing any schema validation issues
//   - Error if validation process fails
func ValidateStatementSchema(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parsercommon.StatementNode,
	subqueryInfo *parserstep7.ParseResult,
) ([]ValidationError, error) {
	// Create type inference engine for validation
	engine := NewTypeInferenceEngine2(databaseSchemas, statementNode, subqueryInfo)

	// For SELECT statements, validate fields
	if selectStmt, ok := statementNode.(*parsercommon.SelectStatement); ok {
		// Extract table aliases
		engine.extractTableAliases(selectStmt)

		// Create schema validator
		validator := NewSchemaValidator(engine.schemaResolver)
		validator.SetTableAliases(engine.context.TableAliases)
		validator.SetAvailableTables(engine.context.CurrentTables)

		// Validate SELECT fields
		validationErrors := validator.ValidateSelectFields(selectStmt.Select.Fields)

		// Add subquery validation errors if subquery resolver is available
		if engine.subqueryResolver != nil {
			subqueryErrors := engine.subqueryResolver.ValidateSubqueryReferences()
			validationErrors = append(validationErrors, subqueryErrors...)
		}

		return validationErrors, nil
	}

	// For DML statements, validation is handled by DML inference engine
	if engine.dmlEngine != nil {
		// DML validation is performed during type inference
		// For now, return empty validation errors for DML statements
		return []ValidationError{}, nil
	}

	return []ValidationError{}, nil
}
