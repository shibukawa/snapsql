package snapsql

import "errors"

// Common errors used throughout the SnapSQL package
var (
	// ErrParameterNotProvided is returned when a required explicit parameter is missing in a statement.
	// System field errors
	ErrParameterNotProvided = errors.New("statement requires explicit parameter but it was not provided")
	// ErrSystemFieldNotIncluded indicates a required system field was not listed in the column list.
	ErrSystemFieldNotIncluded = errors.New("statement requires explicit system field to be included in column list")
	// ErrParameterConfiguredError is returned when a parameter that must not appear was found.
	ErrParameterConfiguredError = errors.New("statement should not include parameter (configured as error)")
	// ErrInvalidStatementCast indicates a failure to cast a statement node to a more specific type.
	ErrInvalidStatementCast = errors.New("failed to cast statement")
	// ErrMissingClause indicates a required SQL clause is missing.
	ErrMissingClause = errors.New("statement has missing required clause")
	// ErrClosingParenthesisNotFound indicates an unmatched opening parenthesis.
	ErrClosingParenthesisNotFound = errors.New("could not find closing parenthesis")

	// ErrInvalidDecimalString is returned when a numeric literal cannot be parsed as decimal.
	// Decimal errors
	ErrInvalidDecimalString = errors.New("invalid decimal string")
	// ErrUnsupportedConversion indicates an attempted unsupported type conversion.
	ErrUnsupportedConversion = errors.New("unsupported conversion")

	// ErrFieldNotFound indicates a struct field was not found by name.
	// Hierarchical errors
	ErrFieldNotFound = errors.New("field not found in struct")
	// ErrCannotAssignNil is returned when assigning nil to a non-pointer field.
	ErrCannotAssignNil = errors.New("cannot assign nil to non-pointer type")
	// ErrCannotConvert indicates an unsupported structure or value conversion.
	ErrCannotConvert = errors.New("cannot convert type")

	// ErrNoMockDataNames indicates no mock data names were provided.
	// Runtime errors
	ErrNoMockDataNames = errors.New("no mock data names specified")

	// ErrNoParameterFound indicates parameter definitions were not found in a code block or list.
	// Parser errors
	ErrNoParameterFound = errors.New("no parameter code block or list found")
	// ErrEmptyContent indicates the Markdown or SQL content was empty.
	ErrEmptyContent = errors.New("empty content")
	// ErrEmptyParameters indicates the parameters block exists but is empty.
	ErrEmptyParameters = errors.New("empty parameters object")
	// ErrFailedToParse indicates a generic parsing failure on input data.
	ErrFailedToParse = errors.New("failed to parse data")
	// ErrEmptyExpectedResults indicates an Expected Results block was present but empty.
	ErrEmptyExpectedResults = errors.New("empty expected results")
	// ErrNoDatasetElement indicates a dataset element was not found where required.
	ErrNoDatasetElement = errors.New("no dataset element found")
	// ErrInvalidCSVFormat indicates the CSV file lacks required rows.
	ErrInvalidCSVFormat = errors.New("CSV must have at least a header row and one data row")
	// ErrTestCaseMissingData indicates a test case lacked required body sections.
	ErrTestCaseMissingData = errors.New("test case is missing required data")
	// ErrDuplicateVerifyQuery indicates multiple verify queries were defined in one test case.
	ErrDuplicateVerifyQuery = errors.New("duplicate verify query in test case")
	// ErrTableNameRequired indicates fixtures CSV requires explicit table name.
	ErrTableNameRequired = errors.New("table name is required for CSV fixtures in test case")

	// ErrNoFunctionDefinition indicates no function definition comment block was detected.
	// Function definition errors
	ErrNoFunctionDefinition = errors.New("no function definition found in SQL comment")
	// ErrUnsupportedParameterType indicates a function parameter type is unsupported.
	ErrUnsupportedParameterType = errors.New("unsupported parameter type")
	// ErrInvalidParameterFormat indicates a malformed function parameter specification.
	ErrInvalidParameterFormat = errors.New("invalid parameter format")

	// ErrNodeNotFound indicates a referenced node was missing from the graph.
	// Node and graph errors
	ErrNodeNotFound = errors.New("node not found")
	// ErrInvalidFieldSourceType indicates an unexpected field source type.
	ErrInvalidFieldSourceType = errors.New("invalid field source type")
	// ErrInvalidStatement indicates a statement is invalid for subquery extraction.
	ErrInvalidStatement = errors.New("invalid statement for subquery parsing")
	// ErrUnexpectedReturnType indicates an unexpected dynamic return type was encountered.
	ErrUnexpectedReturnType = errors.New("unexpected return type")
	// ErrSubqueryExtraction indicates subquery extraction failed.
	ErrSubqueryExtraction = errors.New("failed to extract subqueries")

	// ErrValueIsNil indicates a value was unexpectedly nil.
	// Array and value errors
	ErrValueIsNil = errors.New("value is nil")
	// ErrNotSliceOrArray indicates a non-slice/array value was provided where one is required.
	ErrNotSliceOrArray = errors.New("value is not a slice or array")
	// ErrArrayElementNotObject indicates an array element was not a JSON object structure.
	ErrArrayElementNotObject = errors.New("array element is not an object")

	// ErrUnexpectedCharacter indicates a lexer encountered an unexpected character.
	// Tokenizer errors
	ErrUnexpectedCharacter = errors.New("unexpected character")
	// ErrUnterminatedString indicates a string literal was not properly terminated.
	ErrUnterminatedString = errors.New("unterminated string literal")
	// ErrUnterminatedComment indicates a block comment was not properly terminated.
	ErrUnterminatedComment = errors.New("unterminated block comment")
	// ErrInvalidNumber indicates an invalid numeric literal format.
	ErrInvalidNumber = errors.New("invalid number format")
	// ErrInvalidSingleColon indicates a stray single colon token was found.
	ErrInvalidSingleColon = errors.New("invalid single colon")

	// ErrExpressionIndexNotFound indicates an expression index reference was invalid.
	// Query executor errors
	ErrExpressionIndexNotFound = errors.New("expression index not found")

	// ErrNoTestCasesFound indicates no test cases matched the provided pattern.
	// Test runner errors
	ErrNoTestCasesFound = errors.New("no test cases found matching pattern")
	// ErrFixtureOnlyModeRequiresOne indicates fixture-only mode expects exactly one test case.
	ErrFixtureOnlyModeRequiresOne = errors.New("fixture-only mode requires exactly one test case")
	// ErrUnsupportedExecutionMode indicates an unsupported execution mode was selected.
	ErrUnsupportedExecutionMode = errors.New("unsupported execution mode")
	// ErrUnsupportedQueryType indicates an unsupported query type was encountered.
	ErrUnsupportedQueryType = errors.New("unsupported query type")
	// ErrUnsupportedInsertStrategy indicates an unsupported insert strategy was requested.
	ErrUnsupportedInsertStrategy = errors.New("unsupported insert strategy")
	// ErrUpsertNotSupported indicates UPSERT is not supported for the dialect.
	ErrUpsertNotSupported = errors.New("upsert not supported for dialect")
	// ErrDeleteStrategyNotImplemented indicates the delete strategy is not yet implemented.
	ErrDeleteStrategyNotImplemented = errors.New("delete strategy not yet implemented")
	// ErrTruncateNotSupported indicates TRUNCATE is not supported for the dialect.
	ErrTruncateNotSupported = errors.New("truncate not supported for dialect")
	// ErrPostgresUpsertNotImplemented indicates PostgreSQL upsert is not implemented.
	ErrPostgresUpsertNotImplemented = errors.New("postgres upsert not yet implemented")
	// ErrMysqlUpsertNotImplemented indicates MySQL upsert is not implemented.
	ErrMysqlUpsertNotImplemented = errors.New("mysql upsert not yet implemented")
	// ErrSqliteUpsertNotImplemented indicates SQLite upsert is not implemented.
	ErrSqliteUpsertNotImplemented = errors.New("sqlite upsert not yet implemented")
	// ErrResultRowCountMismatch indicates the number of result rows mismatched expectations.
	ErrResultRowCountMismatch = errors.New("result row count mismatch")
	// ErrInvalidValidationSpecFormat indicates a validation spec key had invalid format.
	ErrInvalidValidationSpecFormat = errors.New("invalid validation spec format")
	// ErrUnsupportedValidationStrategy indicates a validation strategy keyword is unsupported.
	ErrUnsupportedValidationStrategy = errors.New("unsupported validation strategy")
	// ErrExpectedDataMustBeArray indicates direct result validation expected an array of objects.
	ErrExpectedDataMustBeArray = errors.New("expected data must be an array of objects for direct result validation")
	// ErrExpectedNumericMustBeMap indicates numeric validation expected a map.
	ErrExpectedNumericMustBeMap = errors.New("expected numeric validation must be a map")
	// ErrLastInsertIdNotImplemented indicates last_insert_id validation is not implemented.
	ErrLastInsertIdNotImplemented = errors.New("last_insert_id validation not yet implemented")
	// ErrUnsupportedNumericValidationKey indicates an unsupported numeric validation key.
	ErrUnsupportedNumericValidationKey = errors.New("unsupported numeric validation key")
	// ErrTableStateValidationItemMustBeObject indicates a table state item must be an object.
	ErrTableStateValidationItemMustBeObject = errors.New("table state validation item must be an object")
	// ErrTableStateValidationMustBeArray indicates table state validation expected an object array or single object.
	ErrTableStateValidationMustBeArray = errors.New("table state validation must be an array of objects or a single object")
	// ErrTableRowCountMismatch indicates a table row count mismatched expectations.
	ErrTableRowCountMismatch = errors.New("table row count mismatch")
	// ErrMissingRowInTable indicates an expected row was not found.
	ErrMissingRowInTable = errors.New("missing row in table")
	// ErrExistenceValidationItemMustBeObject indicates an existence validation item must be an object.
	ErrExistenceValidationItemMustBeObject = errors.New("existence validation item must be an object")
	// ErrExistenceValidationMustBeArray indicates existence validation expected an array or single object.
	ErrExistenceValidationMustBeArray = errors.New("existence validation must be an array of objects or a single object")
	// ErrExistenceValidationRequiresExists indicates the 'exists' field is required.
	ErrExistenceValidationRequiresExists = errors.New("existence validation requires 'exists' field with boolean value")
	// ErrExistenceValidationRequiresCondition indicates at least one condition field is required.
	ErrExistenceValidationRequiresCondition = errors.New("existence validation requires at least one condition field")
	// ErrExistenceValidationMismatch indicates existence validation failed.
	ErrExistenceValidationMismatch = errors.New("existence validation mismatch")
	// ErrCountMismatch indicates a row count validation failed.
	ErrCountMismatch = errors.New("count mismatch")
	// ErrMissingField indicates a field was missing from a result row.
	ErrMissingField = errors.New("missing field")
	// ErrFieldValueMismatch indicates a field value mismatched expectations.
	ErrFieldValueMismatch = errors.New("field value mismatch")
	// ErrCannotConvertToInt64 indicates a numeric conversion failed.
	ErrCannotConvertToInt64 = errors.New("cannot convert to int64")

	// ErrUnsupportedType indicates a type is not supported for code generation.
	// Code generation errors
	ErrUnsupportedType = errors.New("unsupported type")
	// ErrUnsupportedResponseAffinity indicates a declared response affinity is unsupported.
	ErrUnsupportedResponseAffinity = errors.New("unsupported response affinity")
	// ErrDialectMustBeSpecified indicates a dialect is required but missing.
	ErrDialectMustBeSpecified = errors.New("dialect must be specified (postgres, mysql, sqlite)")

	// ErrUnsupportedDMLStatementType indicates the DML statement type is unsupported.
	// Type inference errors
	ErrUnsupportedDMLStatementType = errors.New("unsupported DML statement type")
	// ErrInsertStatementMissingInto indicates an INSERT lacked INTO clause.
	ErrInsertStatementMissingInto = errors.New("INSERT statement missing INTO clause")
	// ErrUpdateStatementMissingUpdate indicates an UPDATE lacked UPDATE keyword.
	ErrUpdateStatementMissingUpdate = errors.New("UPDATE statement missing UPDATE clause")
	// ErrDeleteStatementMissingFrom indicates a DELETE lacked FROM clause.
	ErrDeleteStatementMissingFrom = errors.New("DELETE statement missing FROM clause")
	// ErrColumnDoesNotExist indicates a referenced column does not exist.
	ErrColumnDoesNotExist = errors.New("column does not exist in table")
	// ErrSchemaDoesNotExist indicates a referenced schema does not exist.
	ErrSchemaDoesNotExist = errors.New("schema does not exist")
	// ErrTableDoesNotExist indicates a referenced table does not exist.
	ErrTableDoesNotExist = errors.New("table does not exist in schema")
	// ErrDependencyNodeNotFound indicates a dependency node was missing.
	ErrDependencyNodeNotFound = errors.New("dependency node not found")
	// ErrNoStatementFoundForSubquery indicates a subquery node had no statement.
	ErrNoStatementFoundForSubquery = errors.New("no statement found for subquery node")
	// ErrUnsupportedSubqueryStatementType indicates a subquery statement type is unsupported.
	ErrUnsupportedSubqueryStatementType = errors.New("unsupported subquery statement type")
	// ErrStatementIsNotSelect indicates a non-SELECT statement where SELECT expected.
	ErrStatementIsNotSelect = errors.New("statement is not a SELECT statement")
	// ErrUnsupportedStatementType indicates a statement type is unsupported.
	ErrUnsupportedStatementType = errors.New("unsupported statement type")
	// ErrSchemaValidationFailed indicates schema validation failed.
	ErrSchemaValidationFailed = errors.New("schema validation failed")
	// ErrColumnNotFoundInSubquery indicates a column was not found in a subquery.
	ErrColumnNotFoundInSubquery = errors.New("column not found in subquery")
	// ErrTableNotFoundInSchema indicates a table was not found in schema.
	ErrTableNotFoundInSchema = errors.New("table not found in schema")
	// ErrColumnAmbiguousInSubqueries indicates a column reference was ambiguous.
	ErrColumnAmbiguousInSubqueries = errors.New("column is ambiguous in subqueries")
	// ErrColumnNotFoundInAnyTable indicates a column was not found in any available table.
	ErrColumnNotFoundInAnyTable = errors.New("column not found in any available table")
	// ErrColumnAmbiguous indicates a column reference appears in multiple tables.
	ErrColumnAmbiguous = errors.New("column is ambiguous, found in tables")
	// ErrInvalidTableReference indicates a malformed table reference.
	ErrInvalidTableReference = errors.New("invalid table reference")
	// ErrEmptyTokenList indicates the token list was unexpectedly empty.
	ErrEmptyTokenList = errors.New("empty token list")
	// ErrNotCaseExpression indicates a CASE expression parse failed.
	ErrNotCaseExpression = errors.New("not a CASE expression")
	// ErrConfigFileNotFound indicates a configuration file could not be located.
	ErrConfigFileNotFound = errors.New("configuration file not found")
	// ErrNoResponseFields indicates a generated query had no response fields.
	ErrNoResponseFields = errors.New("no response fields found")
)
