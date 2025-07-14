package parsercommon

import "errors"

// Sentinel errors - Parser related
var (
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
