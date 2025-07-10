package parsercommon

import "errors"

var (
	// ErrInvalidSQL is returned when the SQL syntax is invalid
	ErrInvalidSQL        = errors.New("invalid SQL syntax")
	ErrInvalidForSnapSQL = errors.New("invalid SQL syntax for SnapSQL")
)
