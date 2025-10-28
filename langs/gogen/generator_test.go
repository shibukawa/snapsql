package gogen

import (
	"errors"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestConvertToGoTypeFloatNormalization(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"float", "float64"},
		{"float32", "float64"},
		{"float64", "float64"},
	}
	for _, c := range cases {
		got, err := convertToGoType(c.in)
		if err != nil {
			t.Fatalf("convertToGoType(%s) unexpected error: %v", c.in, err)
		}

		if got != c.out {
			t.Errorf("convertToGoType(%s) = %s, want %s", c.in, got, c.out)
		}
	}
}

func TestConvertToGoTypeTemporalAliases(t *testing.T) {
	aliases := []string{"timestamp", "datetime", "date", "time"}

	for _, alias := range aliases {
		got, err := convertToGoType(alias)
		if err != nil {
			t.Fatalf("convertToGoType(%s) unexpected error: %v", alias, err)
		}

		if got != "time.Time" {
			t.Errorf("convertToGoType(%s) = %s, want time.Time", alias, got)
		}
	}
}

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
				var unsupportedErr *UnsupportedTypeError
				if errors.As(err, &unsupportedErr) {
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

func TestProcessCELVariableTemporalAlias(t *testing.T) {
	data, err := processCELVariable(intermediate.CELVariableInfo{Name: "since", Type: "datetime"})
	if err != nil {
		t.Fatalf("processCELVariable returned error: %v", err)
	}

	if data.CelType != "TimestampType" {
		t.Fatalf("expected CelType TimestampType, got %s", data.CelType)
	}

	if data.GoType != "time.Time" {
		t.Fatalf("expected GoType time.Time, got %s", data.GoType)
	}
}

func TestProcessResponseStructRootPkNonNull(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_by_board",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "id", Type: "int", IsNullable: true, HierarchyKeyLevel: 1},
			{Name: "name", Type: "string"},
		},
	}

	respStruct, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("processResponseStruct returned error: %v", err)
	}

	if len(respStruct.Fields) == 0 {
		t.Fatalf("expected fields in response struct")
	}

	field := respStruct.Fields[0]
	if field.Type != "int" {
		t.Errorf("expected root PK field type int, got %s", field.Type)
	}

	if field.IsPointer {
		t.Errorf("expected root PK field to be non-pointer")
	}
}

func TestGenerateQueryExecutionManyIterator(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_by_board",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "string"},
		},
	}

	respStruct, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("processResponseStruct returned error: %v", err)
	}

	data, err := generateQueryExecution(format, respStruct, nil, respStruct.Name, "ListByBoard", "result", true)
	if err != nil {
		t.Fatalf("generateQueryExecution returned error: %v", err)
	}

	if !data.IsIterator {
		t.Fatalf("expected iterator generation for many affinity")
	}

	if len(data.IteratorBody) == 0 {
		t.Fatalf("expected iterator body to be generated")
	}

	expectedYield := "*" + respStruct.Name
	if data.IteratorYieldType != expectedYield {
		t.Errorf("expected iterator yield type %s, got %s", expectedYield, data.IteratorYieldType)
	}
}
