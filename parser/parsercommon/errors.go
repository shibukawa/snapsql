package parsercommon

import "fmt"

// Sentinel errors used throughout parser diagnostics.
var (
	ErrEmptyExplangExpression    = fmt.Errorf("%w: empty explang expression", ErrInvalidForSnapSQL)
	ErrNilParameterValues        = fmt.Errorf("%w: parameter values not initialized", ErrInvalidForSnapSQL)
	ErrExplangMissingIdentifier  = fmt.Errorf("%w: explang identifier missing", ErrInvalidForSnapSQL)
	ErrExplangInvalidSafeAccess  = fmt.Errorf("%w: explang safe access misuse", ErrInvalidForSnapSQL)
	ErrExplangInvalidIndex       = fmt.Errorf("%w: explang index invalid", ErrInvalidForSnapSQL)
	ErrExplangUnsupportedStep    = fmt.Errorf("%w: explang unsupported step", ErrInvalidForSnapSQL)
)
