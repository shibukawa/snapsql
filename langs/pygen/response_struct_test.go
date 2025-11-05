package pygen

import (
	"errors"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestProcessResponseStruct_NoFields(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "delete_user",
		Responses:    []intermediate.Response{},
	}

	result, err := processResponseStruct(format)
	if !errors.Is(err, ErrNoResponseFields) {
		t.Errorf("Expected ErrNoResponseFields, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %+v", result)
	}
}

func TestProcessResponseStruct_FlatStructure(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "get_user_by_id",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
			{Name: "email", Type: "string", IsNullable: true},
			{Name: "created_at", Type: "timestamp", IsNullable: false},
		},
	}

	result, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check class name
	expectedClassName := "GetUserByIdResult"
	if result.ClassName != expectedClassName {
		t.Errorf("Expected class name %s, got %s", expectedClassName, result.ClassName)
	}

	// Check field count
	if len(result.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(result.Fields))
	}

	// Check field details
	tests := []struct {
		index      int
		name       string
		typeHint   string
		hasDefault bool
		defaultVal string
	}{
		{0, "user_id", "int", false, ""},
		{1, "username", "str", false, ""},
		{2, "email", "Optional[str]", true, "None"},
		{3, "created_at", "datetime", false, ""},
	}

	for _, tt := range tests {
		field := result.Fields[tt.index]
		if field.Name != tt.name {
			t.Errorf("Field %d: expected name %s, got %s", tt.index, tt.name, field.Name)
		}

		if field.TypeHint != tt.typeHint {
			t.Errorf("Field %d: expected type %s, got %s", tt.index, tt.typeHint, field.TypeHint)
		}

		if field.HasDefault != tt.hasDefault {
			t.Errorf("Field %d: expected hasDefault %v, got %v", tt.index, tt.hasDefault, field.HasDefault)
		}

		if tt.hasDefault && field.Default != tt.defaultVal {
			t.Errorf("Field %d: expected default %s, got %s", tt.index, tt.defaultVal, field.Default)
		}
	}
}

func TestGenerateClassName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"get_user", "GetUserResult"},
		{"get_user_by_id", "GetUserByIdResult"},
		{"list_all_users", "ListAllUsersResult"},
		{"create", "CreateResult"},
		{"update_user_email", "UpdateUserEmailResult"},
	}

	for _, tt := range tests {
		result := generateClassName(tt.input)
		if result != tt.expected {
			t.Errorf("generateClassName(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestProcessResponseStruct_WithArrayTypes(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "get_user_tags",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "tags", Type: "string[]", IsNullable: false},
			{Name: "scores", Type: "int[]", IsNullable: true},
		},
	}

	result, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check field types
	if result.Fields[1].TypeHint != "List[str]" {
		t.Errorf("Expected tags type List[str], got %s", result.Fields[1].TypeHint)
	}

	if result.Fields[2].TypeHint != "Optional[List[int]]" {
		t.Errorf("Expected scores type Optional[List[int]], got %s", result.Fields[2].TypeHint)
	}
}

func TestProcessResponseStruct_WithDecimalAndDatetime(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "get_order",
		Responses: []intermediate.Response{
			{Name: "order_id", Type: "int", IsNullable: false},
			{Name: "amount", Type: "decimal", IsNullable: false},
			{Name: "created_at", Type: "timestamp", IsNullable: false},
			{Name: "updated_at", Type: "datetime", IsNullable: true},
		},
	}

	result, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check decimal type
	if result.Fields[1].TypeHint != "Decimal" {
		t.Errorf("Expected amount type Decimal, got %s", result.Fields[1].TypeHint)
	}

	// Check datetime types (both timestamp and datetime should map to datetime)
	if result.Fields[2].TypeHint != "datetime" {
		t.Errorf("Expected created_at type datetime, got %s", result.Fields[2].TypeHint)
	}

	if result.Fields[3].TypeHint != "Optional[datetime]" {
		t.Errorf("Expected updated_at type Optional[datetime], got %s", result.Fields[3].TypeHint)
	}
}
