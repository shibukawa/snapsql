package parserstep6

import "errors"

// Sentinel errors for parserstep6
var (
	ErrInvalidDirectiveFormat    = errors.New("invalid directive format")
	ErrInvalidIfDirectiveFormat  = errors.New("invalid if directive format")
	ErrInvalidForDirectiveFormat = errors.New("invalid for directive format")
	ErrTemplateValidationFailed  = errors.New("template validation failed")
	ErrNamespaceNotSet           = errors.New("namespace not set")
	ErrInvalidExpression         = errors.New("invalid expression")
)
