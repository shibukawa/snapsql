package parsercommon

import "errors"

// Sentinel errors - Parser related
var (
	// ErrExpectedDocumentNode indicates the root YAML document node was not found.
	ErrExpectedDocumentNode = errors.New("expected document node")
	// ErrExpectedMappingForParams indicates parameters section should be a mapping node.
	ErrExpectedMappingForParams = errors.New("expected mapping node for parameters")
	// ErrExpectedSequenceNode indicates a YAML sequence node was required.
	ErrExpectedSequenceNode = errors.New("expected sequence node")
	// ErrUnsupportedParameterType indicates an unsupported parameter node type.
	ErrUnsupportedParameterType = errors.New("unsupported parameter node type")

	// ErrEnvironmentCELNotInit indicates the CEL environment was not initialized.
	ErrEnvironmentCELNotInit = errors.New("environment CEL not initialized")
	// ErrParameterCELNotInit indicates the CEL parameter environment was not initialized.
	ErrParameterCELNotInit = errors.New("parameter CEL not initialized")
	// ErrNoOutputType indicates no output type was found for an expression.
	ErrNoOutputType = errors.New("no output type for expression")
	// ErrExpressionValidationFailed indicates validation failed in both environment and parameter contexts.
	ErrExpressionValidationFailed = errors.New("expression validation failed for both environment and parameter contexts")
	// ErrExpressionNotList indicates the expression result was not a list.
	ErrExpressionNotList = errors.New("expression result is not a list")
)
