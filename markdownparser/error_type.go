package markdownparser

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidErrorType is returned when an invalid error type is specified
var ErrInvalidErrorType = errors.New("invalid error type")

// ErrorType represents a database error type that can occur at runtime
type ErrorType string

const (
	ErrorTypeUniqueViolation           ErrorType = "unique violation"
	ErrorTypeForeignKeyViolation       ErrorType = "foreign key violation"
	ErrorTypeNotNullViolation          ErrorType = "not null violation"
	ErrorTypeCheckViolation            ErrorType = "check violation"
	ErrorTypeNotFound                  ErrorType = "not found"
	ErrorTypeDataTooLong               ErrorType = "data too long"
	ErrorTypeNumericOverflow           ErrorType = "numeric overflow"
	ErrorTypeInvalidTextRepresentation ErrorType = "invalid text representation"
)

// validErrorTypes contains all valid error types for validation
var validErrorTypes = map[string]bool{
	string(ErrorTypeUniqueViolation):           true,
	string(ErrorTypeForeignKeyViolation):       true,
	string(ErrorTypeNotNullViolation):          true,
	string(ErrorTypeCheckViolation):            true,
	string(ErrorTypeNotFound):                  true,
	string(ErrorTypeDataTooLong):               true,
	string(ErrorTypeNumericOverflow):           true,
	string(ErrorTypeInvalidTextRepresentation): true,
}

// normalizeErrorType normalizes error type strings to a canonical form
// - Converts to lowercase: "Unique Violation" → "unique violation"
// - Converts underscores to spaces: "unique_violation" → "unique violation"
// - Converts hyphens to spaces: "unique-violation" → "unique violation"
// - Collapses multiple spaces: "foreign  key  violation" → "foreign key violation"
func normalizeErrorType(input string) string {
	// Convert to lowercase
	normalized := strings.ToLower(input)

	// Replace underscores and hyphens with spaces
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")

	// Collapse multiple spaces into one
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Trim leading/trailing spaces
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// ParseExpectedError parses and validates an error type string from test case
// Returns the normalized error type or an error if invalid
func ParseExpectedError(content string) (*string, error) {
	errorType := normalizeErrorType(strings.TrimSpace(content))

	// Validate against known error types
	if !validErrorTypes[errorType] {
		return nil, fmt.Errorf("%w: %s (original: %s)", ErrInvalidErrorType, errorType, content)
	}

	return &errorType, nil
}
