package parser

import "errors"

// Sentinel errors - Parser related
var (
	// Basic parser errors (already defined in parser.go)
	// ErrUnexpectedToken     = errors.New("unexpected token")
	// ErrUnexpectedEOF       = errors.New("unexpected end of file")
	// ErrMismatchedParens    = errors.New("mismatched parentheses")
	// ErrMismatchedQuotes    = errors.New("mismatched quotes")
	// ErrInvalidSyntax       = errors.New("invalid syntax")
	// ErrMismatchedDirective = errors.New("mismatched SnapSQL directive")

	// Syntax parsing errors
	ErrExpectedSelect       = errors.New("expected SELECT")
	ErrExpectedBy           = errors.New("expected BY after GROUP/ORDER")
	ErrExpectedExpression   = errors.New("expected expression")
	ErrSelectMustHaveFields = errors.New("SELECT clause must have at least one field")
	ErrExpectedEndDirective = errors.New("expected end directive")
	ErrExpectedIdentifier   = errors.New("expected identifier")
	ErrExpectedValue        = errors.New("expected value")
	ErrExpectedTableName    = errors.New("expected table name")

	// INSERT statement errors
	ErrExpectedInsert      = errors.New("expected INSERT")
	ErrExpectedInto        = errors.New("expected INTO after INSERT")
	ErrExpectedCloseParen  = errors.New("expected ')'")
	ErrExpectedOpenParen   = errors.New("expected '('")
	ErrExpectedValues      = errors.New("expected VALUES or SELECT after table name")
	ErrExpectedValuesAfter = errors.New("expected '(' after VALUES")

	// UPDATE statement errors
	ErrExpectedUpdate = errors.New("expected UPDATE")
	ErrExpectedSet    = errors.New("expected SET after table name")
	ErrExpectedEquals = errors.New("expected '=' after column name")

	// DELETE statement errors
	ErrExpectedDelete = errors.New("expected DELETE")
	ErrExpectedFrom   = errors.New("expected FROM after DELETE")

	// CTE related errors
	ErrExpectedCTEName                = errors.New("expected CTE name")
	ErrExpectedAsAfterCTEName         = errors.New("expected 'AS' after CTE name")
	ErrExpectedOpenParenAsAfterCTE    = errors.New("expected '(' after 'AS' in CTE definition")
	ErrExpectedCloseParenAfterCTEList = errors.New("expected ')' after CTE column list")
	ErrExpectedColumnName             = errors.New("expected column name")
	ErrExpectedColumnAfterComma       = errors.New("expected column name after ','")

	// YAML/Schema related errors
	ErrExpectedDocumentNode     = errors.New("expected document node")
	ErrExpectedMappingNode      = errors.New("expected mapping node")
	ErrExpectedMappingForParams = errors.New("expected mapping node for parameters")
	ErrExpectedSequenceNode     = errors.New("expected sequence node")
	ErrUnsupportedParameterType = errors.New("unsupported parameter node type")

	// CEL related errors
	ErrEnvironmentCELNotInit      = errors.New("environment CEL not initialized")
	ErrParameterCELNotInit        = errors.New("parameter CEL not initialized")
	ErrNoOutputType               = errors.New("no output type for expression")
	ErrExpressionValidationFailed = errors.New("expression validation failed for both environment and parameter contexts")
	ErrExpressionNotList          = errors.New("expression result is not a list")
)
