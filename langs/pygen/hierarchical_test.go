package pygen

import (
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestDetectHierarchicalStructure_NoHierarchy(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "user_id", Type: "int", IsNullable: false},
		{Name: "username", Type: "string", IsNullable: false},
	}

	nodes, rootFields, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected no hierarchical nodes, got %d", len(nodes))
	}

	if len(rootFields) != 2 {
		t.Errorf("Expected 2 root fields, got %d", len(rootFields))
	}
}

func TestDetectHierarchicalStructure_SingleLevel(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "board_id", Type: "int", IsNullable: false},
		{Name: "board_name", Type: "string", IsNullable: false},
		{Name: "list__list_id", Type: "int", IsNullable: false},
		{Name: "list__list_name", Type: "string", IsNullable: false},
	}

	nodes, rootFields, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 1 hierarchical node (list)
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 hierarchical node, got %d", len(nodes))
	}

	// Check root fields
	if len(rootFields) != 2 {
		t.Errorf("Expected 2 root fields, got %d", len(rootFields))
	}

	// Check list node
	listNode, ok := nodes["list"]
	if !ok {
		t.Fatal("Expected 'list' node not found")
	}

	if len(listNode.Fields) != 2 {
		t.Errorf("Expected 2 fields in list node, got %d", len(listNode.Fields))
	}

	// Check field names are converted to snake_case
	if listNode.Fields[0].Name != "list_id" {
		t.Errorf("Expected field name 'list_id', got '%s'", listNode.Fields[0].Name)
	}
}

func TestDetectHierarchicalStructure_MultiLevel(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "board_id", Type: "int", IsNullable: false},
		{Name: "board_name", Type: "string", IsNullable: false},
		{Name: "list__list_id", Type: "int", IsNullable: false},
		{Name: "list__list_name", Type: "string", IsNullable: false},
		{Name: "list__card__card_id", Type: "int", IsNullable: false},
		{Name: "list__card__card_title", Type: "string", IsNullable: false},
	}

	nodes, _, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 2 hierarchical nodes (list and list__card)
	if len(nodes) != 2 {
		t.Fatalf("Expected 2 hierarchical nodes, got %d", len(nodes))
	}

	// Check list node
	listNode, ok := nodes["list"]
	if !ok {
		t.Fatal("Expected 'list' node not found")
	}

	// Check list__card node
	cardNode, ok := nodes["list__card"]
	if !ok {
		t.Fatal("Expected 'list__card' node not found")
	}

	// Check parent-child relationship
	if len(listNode.Children) != 1 {
		t.Errorf("Expected 1 child in list node, got %d", len(listNode.Children))
	}

	if _, ok := listNode.Children["card"]; !ok {
		t.Error("Expected 'card' child in list node")
	}

	// Check card fields
	if len(cardNode.Fields) != 2 {
		t.Errorf("Expected 2 fields in card node, got %d", len(cardNode.Fields))
	}
}

func TestGenerateHierarchicalStructs_SingleLevel(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "board_id", Type: "int", IsNullable: false},
		{Name: "board_name", Type: "string", IsNullable: false},
		{Name: "list__list_id", Type: "int", IsNullable: false},
		{Name: "list__list_name", Type: "string", IsNullable: false},
	}

	nodes, rootFields, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	structs, mainStruct, err := generateHierarchicalStructs("get_board_with_lists", nodes, rootFields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 1 child struct
	if len(structs) != 1 {
		t.Fatalf("Expected 1 child struct, got %d", len(structs))
	}

	// Check main struct
	if mainStruct.ClassName != "GetBoardWithListsResult" {
		t.Errorf("Expected main class name 'GetBoardWithListsResult', got '%s'", mainStruct.ClassName)
	}

	// Main struct should have: 2 root fields + 1 list field
	if len(mainStruct.Fields) != 3 {
		t.Errorf("Expected 3 fields in main struct, got %d", len(mainStruct.Fields))
	}

	// Check list field type
	listField := mainStruct.Fields[2]
	if listField.Name != "list" {
		t.Errorf("Expected list field name 'list', got '%s'", listField.Name)
	}

	if listField.TypeHint != "List[GetBoardWithListsResultList]" {
		t.Errorf("Expected list field type 'List[GetBoardWithListsResultList]', got '%s'", listField.TypeHint)
	}
}

func TestGenerateHierarchicalStructs_MultiLevel(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "board_id", Type: "int", IsNullable: false},
		{Name: "list__list_id", Type: "int", IsNullable: false},
		{Name: "list__card__card_id", Type: "int", IsNullable: false},
	}

	nodes, rootFields, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	structs, _, err := generateHierarchicalStructs("get_board_hierarchy", nodes, rootFields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 2 child structs (list and card)
	if len(structs) != 2 {
		t.Fatalf("Expected 2 child structs, got %d", len(structs))
	}

	// Check that card struct is generated before list struct (deeper first)
	if structs[0].ClassName != "GetBoardHierarchyResultListCard" {
		t.Errorf("Expected first struct to be card, got '%s'", structs[0].ClassName)
	}

	if structs[1].ClassName != "GetBoardHierarchyResultList" {
		t.Errorf("Expected second struct to be list, got '%s'", structs[1].ClassName)
	}

	// Check that list struct has card field
	listStruct := structs[1]
	hasCardField := false

	for _, field := range listStruct.Fields {
		if field.Name == "card" {
			hasCardField = true

			if field.TypeHint != "List[GetBoardHierarchyResultListCard]" {
				t.Errorf("Expected card field type 'List[GetBoardHierarchyResultListCard]', got '%s'", field.TypeHint)
			}
		}
	}

	if !hasCardField {
		t.Error("Expected list struct to have card field")
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "User"},
		{"user_id", "UserId"},
		{"board_list", "BoardList"},
		{"list", "List"},
		{"card_title", "CardTitle"},
		{"id", "Id"},
	}

	for _, tt := range tests {
		result := toPascalCase(tt.input)
		if result != tt.expected {
			t.Errorf("toPascalCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestProcessResponseStruct_WithHierarchy(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "get_board_with_lists",
		Responses: []intermediate.Response{
			{Name: "board_id", Type: "int", IsNullable: false},
			{Name: "board_name", Type: "string", IsNullable: false},
			{Name: "list__list_id", Type: "int", IsNullable: false},
			{Name: "list__list_name", Type: "string", IsNullable: false},
		},
	}

	result, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return main struct with hierarchical structure
	if result.ClassName != "GetBoardWithListsResult" {
		t.Errorf("Expected class name 'GetBoardWithListsResult', got '%s'", result.ClassName)
	}

	// Should have 3 fields: board_id, board_name, list
	if len(result.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(result.Fields))
	}
}
