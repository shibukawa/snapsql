package gogen

import (
	"strings"
	"testing"
)

func TestConvertToGoType_UnknownType(t *testing.T) {
	tests := []struct {
		name        string
		inputType   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid type name with special characters",
			inputType:   "unknown-type",
			expectError: true,
			errorMsg:    "unsupported parameter type 'unknown-type'",
		},
		{
			name:        "invalid type name with numbers at start",
			inputType:   "123invalid",
			expectError: true,
			errorMsg:    "unsupported parameter type '123invalid'",
		},
		{
			name:        "valid custom type",
			inputType:   "CustomType",
			expectError: false,
		},
		{
			name:        "valid unknown but valid identifier type",
			inputType:   "unknowntype",
			expectError: false,
		},
		{
			name:        "valid package qualified type",
			inputType:   "time.Time",
			expectError: false,
		},
		{
			name:        "valid array type",
			inputType:   "string[]",
			expectError: false,
		},
		{
			name:        "invalid array element type",
			inputType:   "123invalid[]",
			expectError: true,
			errorMsg:    "unsupported parameter type '123invalid'",
		},
		{
			name:        "invalid empty type",
			inputType:   "",
			expectError: true,
			errorMsg:    "unsupported parameter type ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToGoType(tt.inputType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}

				// Check if it's our custom error type with hints
				if unsupportedErr, ok := err.(*UnsupportedTypeError); ok {
					if len(unsupportedErr.Hints) == 0 {
						t.Errorf("expected hints to be provided")
					}
					t.Logf("Error with hints: %s", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result == "" {
					t.Errorf("expected non-empty result")
				}
			}
		})
	}
}

func TestIsValidGoTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"valid simple type", "MyType", true},
		{"valid lowercase type", "myType", true},
		{"valid with underscore", "My_Type", true},
		{"valid package qualified", "time.Time", true},
		{"invalid empty", "", false},
		{"invalid starts with number", "123Type", false},
		{"invalid special characters", "My-Type", false},
		{"invalid multiple dots", "a.b.c", false},
		{"valid single letter", "T", true},
		{"valid underscore start", "_Type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGoTypeName(tt.typeName)
			if result != tt.expected {
				t.Errorf("isValidGoTypeName(%q) = %v, expected %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   bool
	}{
		{"valid identifier", "myVar", true},
		{"valid with underscore", "my_var", true},
		{"valid starts with underscore", "_var", true},
		{"valid uppercase", "MyVar", true},
		{"invalid empty", "", false},
		{"invalid starts with number", "123var", false},
		{"invalid with dash", "my-var", false},
		{"invalid with space", "my var", false},
		{"valid single letter", "a", true},
		{"valid single underscore", "_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGoIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("isValidGoIdentifier(%q) = %v, expected %v", tt.identifier, result, tt.expected)
			}
		})
	}
}

func TestUnsupportedTypeError(t *testing.T) {
	tests := []struct {
		name            string
		typeName        string
		context         string
		expectedMessage string
		expectedHints   int
	}{
		{
			name:            "parameter context",
			typeName:        "unknowntype",
			context:         "parameter",
			expectedMessage: "unsupported parameter type 'unknowntype'",
			expectedHints:   4,
		},
		{
			name:            "implicit parameter context",
			typeName:        "badtype",
			context:         "implicit parameter 'created_by'",
			expectedMessage: "unsupported implicit parameter 'created_by' type 'badtype'",
			expectedHints:   2,
		},
		{
			name:            "type context",
			typeName:        "invalidtype",
			context:         "type",
			expectedMessage: "unsupported type type 'invalidtype'",
			expectedHints:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newUnsupportedTypeError(tt.typeName, tt.context)

			if !strings.Contains(err.Message, tt.expectedMessage) {
				t.Errorf("expected message to contain '%s', got '%s'", tt.expectedMessage, err.Message)
			}

			if len(err.Hints) != tt.expectedHints {
				t.Errorf("expected %d hints, got %d", tt.expectedHints, len(err.Hints))
			}

			// Test Error() method includes hint
			errorStr := err.Error()
			if !strings.Contains(errorStr, "Hint:") {
				t.Errorf("expected error string to contain hint, got: %s", errorStr)
			}

			t.Logf("Error: %s", errorStr)
		})
	}
}
