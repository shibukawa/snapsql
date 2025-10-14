package typeinference

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

// InferFieldTypes performs type inference and schema validation with additional options and context.
// This function collects all type inference and validation errors independently,
// returning them as a joined error using errors.Join.
//
// Parameters:
//   - databaseSchemas: Database schema information from pull functionality
//   - statementNode: Parsed SQL AST (StatementNode)
//   - options: Additional inference options (can be nil for default options)
//
// Returns:
//   - Slice of InferredFieldInfo containing type information for each field
//   - Joined error containing all type inference and validation errors (use AsTypeInferenceErrors/AsValidationErrors to separate)
//
// Error Handling:
//   - Type inference errors: Use AsTypeInferenceErrors(err) to extract
//   - Validation errors: Use AsValidationErrors(err) to extract
//   - Multiple independent errors are collected and returned as a joined error
//
// Schema Validation:
//   - Table existence validation
//   - Column existence validation
//   - Type compatibility checks
//   - Use AsValidationErrors(err) to extract validation errors only
//
// Supported statement types:
//   - SELECT statements: Returns type information for SELECT fields
//   - INSERT statements: Returns type information for RETURNING clause fields (if present)
//   - UPDATE statements: Returns type information for RETURNING clause fields (if present)
//   - DELETE statements: Returns type information for RETURNING clause fields (if present)
//
// For DML statements without RETURNING clause, returns a single field representing affected rows count.
//
// For validation-only use cases, ignore the first return value and extract validation errors:
//
//	_, err := InferFieldTypes(schemas, statement, nil)
//	if err != nil {
//	    validationErrors := AsValidationErrors(err)
//	    // Handle validation errors only
//	}
func InferFieldTypes(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parser.StatementNode,
	options *InferenceOptions,
) ([]*InferredFieldInfo, error) {
	results, _, err := InferFieldTypesWithWarnings(databaseSchemas, statementNode, options)
	return results, err
}

// InferFieldTypesWithWarnings behaves like InferFieldTypes but also returns collected warnings.
func InferFieldTypesWithWarnings(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode parser.StatementNode,
	options *InferenceOptions,
) ([]*InferredFieldInfo, []string, error) {
	// Validate inputs
	if len(databaseSchemas) == 0 {
		return nil, nil, ErrNoSchemaProvided
	}

	if statementNode == nil {
		return nil, nil, ErrInvalidStatement
	}

	// Create type inference engine
	engine := NewTypeInferenceEngine2(databaseSchemas, statementNode)

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

	var allErrors []error

	// Perform schema validation first (independent of type inference)
	validationErrors := engine.performSchemaValidation()
	allErrors = append(allErrors, validationErrors...)

	// Perform type inference (independent of validation)
	results, inferenceErr := engine.performTypeInference()
	if inferenceErr != nil {
		allErrors = append(allErrors, inferenceErr)
	}

	warnings := engine.Warnings()

	// Return results with combined errors
	if len(allErrors) > 0 {
		return results, warnings, errors.Join(allErrors...)
	}

	return results, warnings, nil
}

// InferenceOptions provides additional options for type inference
type InferenceOptions struct {
	Dialect       snapsql.Dialect   // Database dialect override
	TableAliases  map[string]string // Custom table alias mappings
	CurrentTables []string          // Custom available tables list
}

// Re-export types from parser for convenience
type (
	SubqueryAnalysisResult = parser.SubqueryAnalysisResult
	ValidationErrorInfo    = parser.ValidationError
)
