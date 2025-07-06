package parsercommon

import (
	"errors"
	"strings"
)

// ParseError aggregates multiple parsing errors.
type ParseError struct {
	Errors []error
}

// Error implements the error interface for ParseError.
func (e *ParseError) Error() string {
	if len(e.Errors) == 0 {
		return "no parse errors"
	}
	var sb strings.Builder
	sb.WriteString("Multiple parse errors:")
	for i, err := range e.Errors {
		sb.WriteString("\n  [")
		sb.WriteString(string(rune('1' + i)))
		sb.WriteString("] ")
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// Add appends an error to the ParseError.
func (e *ParseError) Add(err error) {
	if err == nil {
		return
	}
	if perr, ok := err.(*ParseError); ok {
		e.Errors = append(e.Errors, perr.Errors...)
	} else {
		e.Errors = append(e.Errors, err)
	}
}

// AsParseError is a helper to extract *ParseError from error using errors.As.
func AsParseError(err error) (*ParseError, bool) {
	var perr *ParseError
	if errors.As(err, &perr) {
		return perr, true
	}
	return nil, false
}
