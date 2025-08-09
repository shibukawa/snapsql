package typeinference

import (
	"errors"
	"testing"
)

func TestTypeInferenceError(t *testing.T) {
	err := &TypeInferenceError{
		FieldName: "user_id",
		Position:  1,
		ErrorType: "unknown_function",
		Message:   "function 'UNKNOWN_FUNC' not found",
	}

	expected := "type inference error in field 'user_id': function 'UNKNOWN_FUNC' not found"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		TableName:  "users",
		FieldName:  "invalid_column",
		Position:   2,
		ErrorType:  "unknown_column",
		Message:    "column 'invalid_column' does not exist",
		Suggestion: "did you mean 'user_id'?",
	}

	expected := "validation error in table 'users', field 'invalid_column': column 'invalid_column' does not exist"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestAsTypeInferenceErrors_SingleError(t *testing.T) {
	originalErr := &TypeInferenceError{
		FieldName: "test_field",
		ErrorType: "test_error",
		Message:   "test message",
	}

	results := AsTypeInferenceErrors(originalErr)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0] != originalErr {
		t.Errorf("Expected original error, got different error")
	}
}

func TestAsValidationErrors_SingleError(t *testing.T) {
	originalErr := &ValidationError{
		TableName: "test_table",
		FieldName: "test_field",
		ErrorType: "test_error",
		Message:   "test message",
	}

	results := AsValidationErrors(originalErr)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0] != originalErr {
		t.Errorf("Expected original error, got different error")
	}
}

func TestAsTypeInferenceErrors_JoinedErrors(t *testing.T) {
	inferenceErr1 := &TypeInferenceError{
		FieldName: "field1",
		ErrorType: "error1",
		Message:   "message1",
	}
	inferenceErr2 := &TypeInferenceError{
		FieldName: "field2",
		ErrorType: "error2",
		Message:   "message2",
	}
	validationErr := &ValidationError{
		TableName: "table1",
		FieldName: "field3",
		ErrorType: "validation_error",
		Message:   "validation message",
	}

	joinedErr := errors.Join(inferenceErr1, validationErr, inferenceErr2)

	results := AsTypeInferenceErrors(joinedErr)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0] != inferenceErr1 {
		t.Errorf("Expected first inference error")
	}

	if results[1] != inferenceErr2 {
		t.Errorf("Expected second inference error")
	}
}

func TestAsTypeInferenceErrors_NilError(t *testing.T) {
	results := AsTypeInferenceErrors(nil)
	if results != nil {
		t.Error("Expected nil result for nil error")
	}
}

func TestAsValidationErrors_NilError(t *testing.T) {
	results := AsValidationErrors(nil)
	if results != nil {
		t.Error("Expected nil result for nil error")
	}
}
