package typeinference2

import (
	"errors"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// Sentinel errors for input validation
var (
	ErrNoSchemaProvided = errors.New("no database schema provided")
	ErrInvalidStatement = errors.New("invalid statement node")
)

// InferFieldTypes performs type inference with additional options and context.
// This is the main entry point for type inference functionality.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//   - options: Additional inference options (can be nil for default options)
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
	statementNode parser.StatementNode,
	options *InferenceOptions,
) ([]*InferredFieldInfo, error) {
	// Validate inputs
	if databaseSchemas == nil || len(databaseSchemas) == 0 {
		return nil, ErrNoSchemaProvided
	}
	if statementNode == nil {
		return nil, ErrInvalidStatement
	}

	// Extract subquery analysis from statement node
	var subqueryInfo *SubqueryAnalysisInfo
	if statementNode.HasSubqueryAnalysis() {
		subqueryInfo = statementNode.GetSubqueryAnalysis()
	}

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
//
// Returns:
//   - Slice of ValidationError containing any schema validation issues
//   - Error if validation process fails
func ValidateStatementSchema(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parser.StatementNode,
) ([]ValidationError, error) {
	// Validate inputs
	if databaseSchemas == nil || len(databaseSchemas) == 0 {
		return nil, ErrNoSchemaProvided
	}
	if statementNode == nil {
		return nil, ErrInvalidStatement
	}

	// Extract subquery analysis from statement node
	var subqueryInfo *SubqueryAnalysisInfo
	if statementNode.HasSubqueryAnalysis() {
		subqueryInfo = statementNode.GetSubqueryAnalysis()
	}

	// Create type inference engine for validation
	engine := NewTypeInferenceEngine2(databaseSchemas, statementNode, subqueryInfo)

	// For SELECT statements, validate fields
	if selectStmt, ok := statementNode.(*parser.SelectStatement); ok {
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

// Re-export types from parser for convenience
type (
	SubqueryAnalysisInfo = parser.SubqueryAnalysisInfo
	ValidationErrorInfo  = parser.ValidationErrorInfo
)
