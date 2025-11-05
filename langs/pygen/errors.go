package pygen

import (
	"errors"
	"fmt"
)

var (
	// ErrDialectRequired indicates that dialect must be specified
	ErrDialectRequired = errors.New("dialect must be specified")

	// ErrUnsupportedDialect indicates an unsupported database dialect
	ErrUnsupportedDialect = errors.New("unsupported dialect")

	// ErrUnsupportedType indicates an unsupported type conversion
	ErrUnsupportedType = errors.New("unsupported type")

	// ErrInvalidParameter indicates an invalid parameter
	ErrInvalidParameter = errors.New("invalid parameter")

	// ErrNoResponseFields indicates no response fields are defined
	ErrNoResponseFields = errors.New("no response fields")

	// ErrGeneratePythonCode indicates a general code generation error
	ErrGeneratePythonCode = errors.New("failed to generate Python code")

	// ErrInvalidConfiguration indicates invalid configuration
	ErrInvalidConfiguration = errors.New("invalid configuration")
)

// UnsupportedTypeError represents an error for unsupported types with helpful hints
type UnsupportedTypeError struct {
	Type    string
	Context string
	Message string
	Hints   []string
}

func (e *UnsupportedTypeError) Error() string {
	msg := e.Message
	if len(e.Hints) > 0 {
		msg += "\n\nHint: " + e.Hints[0]
		if len(e.Hints) > 1 {
			msg += "\nFor more information, check the documentation"
		}
	}

	return msg
}

// NewUnsupportedTypeError creates a new UnsupportedTypeError with context-appropriate hints
func NewUnsupportedTypeError(typeName, context string) *UnsupportedTypeError {
	err := &UnsupportedTypeError{
		Type:    typeName,
		Context: context,
		Message: fmt.Sprintf("unsupported %s type '%s'", context, typeName),
	}

	// Add context-specific hints
	switch context {
	case "parameter":
		err.Hints = []string{
			"Basic types: int, string, bool, float, decimal, timestamp (aliases: date, time, datetime), bytes, any",
			"Arrays: string[], int[], etc.",
			"Optional types: handled via Optional[T] in Python",
		}
	case "response":
		err.Hints = []string{
			"Supported types: int, string, bool, float, decimal, datetime, bytes, any",
			"Arrays: List[T] in Python",
			"Nullable: Optional[T] in Python",
		}
	case "type conversion":
		err.Hints = []string{
			"Supported basic types: int, int32, int64, string, bool, float, float32, float64, double, decimal, timestamp, date, time, datetime, bytes, any",
			"Array types: append [] to any basic type (e.g., int[], string[])",
			"Nullable types: use nullable parameter in ConvertToPythonType function",
		}
	default:
		err.Hints = []string{
			"Check the documentation for supported type formats",
		}
	}

	return err
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
