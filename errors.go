package snapsql

import "errors"

// Common errors used throughout the SnapSQL package
var (
	// System field errors
	ErrParameterNotProvided       = errors.New("statement requires explicit parameter but it was not provided")
	ErrSystemFieldNotIncluded     = errors.New("statement requires explicit system field to be included in column list")
	ErrParameterConfiguredError   = errors.New("statement should not include parameter (configured as error)")
	ErrInvalidStatementCast       = errors.New("failed to cast statement")
	ErrMissingClause              = errors.New("statement has missing required clause")
	ErrClosingParenthesisNotFound = errors.New("could not find closing parenthesis")

	// Decimal errors
	ErrInvalidDecimalString  = errors.New("invalid decimal string")
	ErrUnsupportedConversion = errors.New("unsupported conversion")

	// Hierarchical errors
	ErrFieldNotFound   = errors.New("field not found in struct")
	ErrCannotAssignNil = errors.New("cannot assign nil to non-pointer type")
	ErrCannotConvert   = errors.New("cannot convert type")

	// Runtime errors
	ErrNoMockDataNames = errors.New("no mock data names specified")

	// Parser errors
	ErrNoParameterFound     = errors.New("no parameter code block or list found")
	ErrEmptyContent         = errors.New("empty content")
	ErrEmptyParameters      = errors.New("empty parameters object")
	ErrFailedToParse        = errors.New("failed to parse data")
	ErrEmptyExpectedResults = errors.New("empty expected results")
	ErrNoDatasetElement     = errors.New("no dataset element found")
	ErrInvalidCSVFormat     = errors.New("CSV must have at least a header row and one data row")
	ErrTestCaseMissingData  = errors.New("test case is missing required data")
	ErrDuplicateVerifyQuery = errors.New("duplicate verify query in test case")
	ErrTableNameRequired    = errors.New("table name is required for CSV fixtures in test case")

	// Function definition errors
	ErrNoFunctionDefinition     = errors.New("no function definition found in SQL comment")
	ErrUnsupportedParameterType = errors.New("unsupported parameter type")
	ErrInvalidParameterFormat   = errors.New("invalid parameter format")

	// Node and graph errors
	ErrNodeNotFound           = errors.New("node not found")
	ErrInvalidFieldSourceType = errors.New("invalid field source type")
	ErrInvalidStatement       = errors.New("invalid statement for subquery parsing")
	ErrUnexpectedReturnType   = errors.New("unexpected return type")
	ErrSubqueryExtraction     = errors.New("failed to extract subqueries")

	// Array and value errors
	ErrValueIsNil            = errors.New("value is nil")
	ErrNotSliceOrArray       = errors.New("value is not a slice or array")
	ErrArrayElementNotObject = errors.New("array element is not an object")

	// Tokenizer errors
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrUnterminatedString  = errors.New("unterminated string literal")
	ErrUnterminatedComment = errors.New("unterminated block comment")
	ErrInvalidNumber       = errors.New("invalid number format")
	ErrInvalidSingleColon  = errors.New("invalid single colon")

	// Query executor errors
	ErrExpressionIndexNotFound = errors.New("expression index not found")

	// Test runner errors
	ErrNoTestCasesFound                     = errors.New("no test cases found matching pattern")
	ErrFixtureOnlyModeRequiresOne           = errors.New("fixture-only mode requires exactly one test case")
	ErrUnsupportedExecutionMode             = errors.New("unsupported execution mode")
	ErrUnsupportedQueryType                 = errors.New("unsupported query type")
	ErrUnsupportedInsertStrategy            = errors.New("unsupported insert strategy")
	ErrUpsertNotSupported                   = errors.New("upsert not supported for dialect")
	ErrDeleteStrategyNotImplemented         = errors.New("delete strategy not yet implemented")
	ErrTruncateNotSupported                 = errors.New("truncate not supported for dialect")
	ErrPostgresUpsertNotImplemented         = errors.New("postgres upsert not yet implemented")
	ErrMysqlUpsertNotImplemented            = errors.New("mysql upsert not yet implemented")
	ErrSqliteUpsertNotImplemented           = errors.New("sqlite upsert not yet implemented")
	ErrResultRowCountMismatch               = errors.New("result row count mismatch")
	ErrInvalidValidationSpecFormat          = errors.New("invalid validation spec format")
	ErrUnsupportedValidationStrategy        = errors.New("unsupported validation strategy")
	ErrExpectedDataMustBeArray              = errors.New("expected data must be an array of objects for direct result validation")
	ErrExpectedNumericMustBeMap             = errors.New("expected numeric validation must be a map")
	ErrLastInsertIdNotImplemented           = errors.New("last_insert_id validation not yet implemented")
	ErrUnsupportedNumericValidationKey      = errors.New("unsupported numeric validation key")
	ErrTableStateValidationItemMustBeObject = errors.New("table state validation item must be an object")
	ErrTableStateValidationMustBeArray      = errors.New("table state validation must be an array of objects or a single object")
	ErrTableRowCountMismatch                = errors.New("table row count mismatch")
	ErrMissingRowInTable                    = errors.New("missing row in table")
	ErrExistenceValidationItemMustBeObject  = errors.New("existence validation item must be an object")
	ErrExistenceValidationMustBeArray       = errors.New("existence validation must be an array of objects or a single object")
	ErrExistenceValidationRequiresExists    = errors.New("existence validation requires 'exists' field with boolean value")
	ErrExistenceValidationRequiresCondition = errors.New("existence validation requires at least one condition field")
	ErrExistenceValidationMismatch          = errors.New("existence validation mismatch")
	ErrCountMismatch                        = errors.New("count mismatch")
	ErrMissingField                         = errors.New("missing field")
	ErrFieldValueMismatch                   = errors.New("field value mismatch")
	ErrCannotConvertToInt64                 = errors.New("cannot convert to int64")

	// Code generation errors
	ErrUnsupportedType             = errors.New("unsupported type")
	ErrUnsupportedResponseAffinity = errors.New("unsupported response affinity")
	ErrDialectMustBeSpecified      = errors.New("dialect must be specified (postgres, mysql, sqlite)")

	// Type inference errors
	ErrUnsupportedDMLStatementType      = errors.New("unsupported DML statement type")
	ErrInsertStatementMissingInto       = errors.New("INSERT statement missing INTO clause")
	ErrUpdateStatementMissingUpdate     = errors.New("UPDATE statement missing UPDATE clause")
	ErrDeleteStatementMissingFrom       = errors.New("DELETE statement missing FROM clause")
	ErrColumnDoesNotExist               = errors.New("column does not exist in table")
	ErrSchemaDoesNotExist               = errors.New("schema does not exist")
	ErrTableDoesNotExist                = errors.New("table does not exist in schema")
	ErrDependencyNodeNotFound           = errors.New("dependency node not found")
	ErrNoStatementFoundForSubquery      = errors.New("no statement found for subquery node")
	ErrUnsupportedSubqueryStatementType = errors.New("unsupported subquery statement type")
	ErrStatementIsNotSelect             = errors.New("statement is not a SELECT statement")
	ErrUnsupportedStatementType         = errors.New("unsupported statement type")
	ErrSchemaValidationFailed           = errors.New("schema validation failed")
	ErrColumnNotFoundInSubquery         = errors.New("column not found in subquery")
	ErrTableNotFoundInSchema            = errors.New("table not found in schema")
	ErrColumnAmbiguousInSubqueries      = errors.New("column is ambiguous in subqueries")
	ErrColumnNotFoundInAnyTable         = errors.New("column not found in any available table")
	ErrColumnAmbiguous                  = errors.New("column is ambiguous, found in tables")
	ErrInvalidTableReference            = errors.New("invalid table reference")
	ErrEmptyTokenList                   = errors.New("empty token list")
	ErrNotCaseExpression                = errors.New("not a CASE expression")
	ErrConfigFileNotFound               = errors.New("configuration file not found")
	ErrNoResponseFields                 = errors.New("no response fields found")
)
