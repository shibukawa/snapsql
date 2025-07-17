package typeinference

import (
	"errors"
	"fmt"
)

// TypeInferenceError represents an error that occurred during type inference
type TypeInferenceError struct {
	FieldName string      // Field name where the error occurred
	Position  int         // Position of the field in the statement
	ErrorType string      // Type of error (e.g., "unknown_function", "type_mismatch")
	Message   string      // Human-readable error message
	Details   interface{} // Additional details about the error
}

// Error implements the error interface
func (e *TypeInferenceError) Error() string {
	if e.FieldName != "" {
		return fmt.Sprintf("type inference error in field '%s': %s", e.FieldName, e.Message)
	}
	return fmt.Sprintf("type inference error: %s", e.Message)
}

// ValidationError represents an error that occurred during schema validation
type ValidationError struct {
	TableName  string // Table name where the error occurred
	FieldName  string // Field name where the error occurred
	Position   int    // Position information
	ErrorType  string // Type of error (e.g., "unknown_table", "unknown_column")
	Message    string // Human-readable error message
	Suggestion string // Suggestion for fixing the error
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.TableName != "" && e.FieldName != "" {
		return fmt.Sprintf("validation error in table '%s', field '%s': %s", e.TableName, e.FieldName, e.Message)
	} else if e.FieldName != "" {
		return fmt.Sprintf("validation error in field '%s': %s", e.FieldName, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// AsTypeInferenceErrors extracts TypeInferenceError instances from a joined error
func AsTypeInferenceErrors(err error) []*TypeInferenceError {
	if err == nil {
		return nil
	}

	var inferenceErrors []*TypeInferenceError

	// Handle joined errors first
	var joinedErr interface{ Unwrap() []error }
	if errors.As(err, &joinedErr) {
		for _, e := range joinedErr.Unwrap() {
			if subErrors := AsTypeInferenceErrors(e); len(subErrors) > 0 {
				inferenceErrors = append(inferenceErrors, subErrors...)
			}
		}
		return inferenceErrors
	}

	// Handle single TypeInferenceError
	var singleInferenceError *TypeInferenceError
	if errors.As(err, &singleInferenceError) {
		return []*TypeInferenceError{singleInferenceError}
	}

	return inferenceErrors
}

// AsValidationErrors extracts ValidationError instances from a joined error
func AsValidationErrors(err error) []*ValidationError {
	if err == nil {
		return nil
	}

	var validationErrors []*ValidationError

	// Handle joined errors first
	var joinedErr interface{ Unwrap() []error }
	if errors.As(err, &joinedErr) {
		for _, e := range joinedErr.Unwrap() {
			if subErrors := AsValidationErrors(e); len(subErrors) > 0 {
				validationErrors = append(validationErrors, subErrors...)
			}
		}
		return validationErrors
	}

	// Handle single ValidationError
	var singleValidationError *ValidationError
	if errors.As(err, &singleValidationError) {
		return []*ValidationError{singleValidationError}
	}

	return validationErrors
}
