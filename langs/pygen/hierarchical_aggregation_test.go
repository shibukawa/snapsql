package pygen

import (
	"errors"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestHasHierarchicalFields(t *testing.T) {
	tests := []struct {
		name      string
		responses []intermediate.Response
		expected  bool
	}{
		{
			name: "no hierarchical fields",
			responses: []intermediate.Response{
				{Name: "user_id", Type: "int"},
				{Name: "username", Type: "string"},
			},
			expected: false,
		},
		{
			name: "with hierarchical fields",
			responses: []intermediate.Response{
				{Name: "user_id", Type: "int"},
				{Name: "order__order_id", Type: "int"},
			},
			expected: true,
		},
		{
			name: "multi-level hierarchical",
			responses: []intermediate.Response{
				{Name: "board_id", Type: "int"},
				{Name: "list__list_id", Type: "int"},
				{Name: "list__card__card_id", Type: "int"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasHierarchicalFields(tt.responses)
			if result != tt.expected {
				t.Errorf("hasHierarchicalFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindParentKeyFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []hierarchicalField
		expected int // number of key fields expected
	}{
		{
			name: "single id field",
			fields: []hierarchicalField{
				{Name: "id", Type: "int", JSONTag: "id"},
				{Name: "name", Type: "string", JSONTag: "name"},
			},
			expected: 1,
		},
		{
			name: "user_id field",
			fields: []hierarchicalField{
				{Name: "user_id", Type: "int", JSONTag: "user_id"},
				{Name: "username", Type: "string", JSONTag: "username"},
			},
			expected: 1,
		},
		{
			name: "no id field",
			fields: []hierarchicalField{
				{Name: "name", Type: "string", JSONTag: "name"},
				{Name: "email", Type: "string", JSONTag: "email"},
			},
			expected: 0,
		},
		{
			name: "board_id field",
			fields: []hierarchicalField{
				{Name: "board_id", Type: "int", JSONTag: "board_id"},
				{Name: "board_name", Type: "string", JSONTag: "board_name"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findParentKeyFields(tt.fields)
			if len(result) != tt.expected {
				t.Errorf("findParentKeyFields() returned %d fields, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestGenerateChildClassName(t *testing.T) {
	tests := []struct {
		name             string
		parentClassName  string
		pathSegments     []string
		expectedContains string
	}{
		{
			name:             "single level",
			parentClassName:  "GetBoardResult",
			pathSegments:     []string{"list"},
			expectedContains: "GetBoardResultList",
		},
		{
			name:             "two levels",
			parentClassName:  "GetBoardResult",
			pathSegments:     []string{"list", "card"},
			expectedContains: "GetBoardResultListCard",
		},
		{
			name:             "with underscore",
			parentClassName:  "GetUserResult",
			pathSegments:     []string{"user_order"},
			expectedContains: "GetUserResultUserOrder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateChildClassName(tt.parentClassName, tt.pathSegments)
			if !strings.Contains(result, tt.expectedContains) {
				t.Errorf("generateChildClassName() = %s, want to contain %s", result, tt.expectedContains)
			}
		})
	}
}

func TestGenerateHierarchicalManyExecution_SingleLevel(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_boards_with_lists",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "board_id", Type: "int", IsNullable: false},
			{Name: "board_name", Type: "string", IsNullable: false},
			{Name: "list__list_id", Type: "int", IsNullable: false},
			{Name: "list__list_name", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetBoardsWithListsResult",
		Fields: []responseFieldData{
			{Name: "board_id", TypeHint: "int"},
			{Name: "board_name", TypeHint: "str"},
			{Name: "list", TypeHint: "List[GetBoardsWithListsResultList]"},
		},
	}

	result, err := generateHierarchicalManyExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Check for key patterns in generated code
	expectedPatterns := []string{
		"hierarchical aggregation",
		"current_parent",
		"current_parent_key",
		"parent_key",
		"yield current_parent",
		"GetBoardsWithListsResult",
		"list=[]",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing expected pattern: %s", pattern)
		}
	}
}

func TestGenerateHierarchicalManyExecution_MultiLevel(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_boards_hierarchy",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "board_id", Type: "int", IsNullable: false},
			{Name: "board_name", Type: "string", IsNullable: false},
			{Name: "list__list_id", Type: "int", IsNullable: false},
			{Name: "list__list_name", Type: "string", IsNullable: false},
			{Name: "list__card__card_id", Type: "int", IsNullable: false},
			{Name: "list__card__card_title", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetBoardsHierarchyResult",
		Fields: []responseFieldData{
			{Name: "board_id", TypeHint: "int"},
			{Name: "board_name", TypeHint: "str"},
			{Name: "list", TypeHint: "List[GetBoardsHierarchyResultList]"},
		},
	}

	result, err := generateHierarchicalManyExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Check for hierarchical patterns
	if !strings.Contains(code, "hierarchical aggregation") {
		t.Error("Generated code missing hierarchical aggregation comment")
	}

	if !strings.Contains(code, "current_parent") {
		t.Error("Generated code missing current_parent variable")
	}

	if !strings.Contains(code, "yield current_parent") {
		t.Error("Generated code missing yield statement")
	}
}

func TestGenerateHierarchicalOneExecution_SingleLevel(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_board_with_lists",
		ResponseAffinity: "one",
		Responses: []intermediate.Response{
			{Name: "board_id", Type: "int", IsNullable: false},
			{Name: "board_name", Type: "string", IsNullable: false},
			{Name: "list__list_id", Type: "int", IsNullable: false},
			{Name: "list__list_name", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetBoardWithListsResult",
		Fields: []responseFieldData{
			{Name: "board_id", TypeHint: "int"},
			{Name: "board_name", TypeHint: "str"},
			{Name: "list", TypeHint: "List[GetBoardWithListsResultList]"},
		},
	}

	result, err := generateHierarchicalOneExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Check for key patterns
	expectedPatterns := []string{
		"hierarchical aggregation",
		"one affinity",
		"result = None",
		"if result is None:",
		"return result",
		"GetBoardWithListsResult",
		"list=[]",
		"NotFoundError",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing expected pattern: %s", pattern)
		}
	}
}

func TestGenerateHierarchicalExecution_MySQL(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_users_with_orders",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
			{Name: "order__order_id", Type: "int", IsNullable: false},
			{Name: "order__amount", Type: "decimal", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetUsersWithOrdersResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
			{Name: "order", TypeHint: "List[GetUsersWithOrdersResultOrder]"},
		},
	}

	result, err := generateHierarchicalManyExecution(format, responseStruct, "mysql")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Check for MySQL-specific patterns
	if !strings.Contains(code, "async for row in cursor:") {
		t.Error("Generated code missing MySQL cursor iteration")
	}

	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("Generated code missing MySQL execute statement")
	}
}

func TestGenerateHierarchicalExecution_SQLite(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_projects_with_tasks",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "project_id", Type: "int", IsNullable: false},
			{Name: "project_name", Type: "string", IsNullable: false},
			{Name: "task__task_id", Type: "int", IsNullable: false},
			{Name: "task__task_title", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetProjectsWithTasksResult",
		Fields: []responseFieldData{
			{Name: "project_id", TypeHint: "int"},
			{Name: "project_name", TypeHint: "str"},
			{Name: "task", TypeHint: "List[GetProjectsWithTasksResultTask]"},
		},
	}

	result, err := generateHierarchicalManyExecution(format, responseStruct, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Check for SQLite-specific patterns
	if !strings.Contains(code, "async for row in cursor:") {
		t.Error("Generated code missing SQLite cursor iteration")
	}

	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("Generated code missing SQLite execute statement")
	}
}

func TestGenerateHierarchicalExecution_NoParentKey(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_data",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "name", Type: "string", IsNullable: false},
			{Name: "value", Type: "string", IsNullable: false},
			{Name: "child__child_name", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetDataResult",
		Fields: []responseFieldData{
			{Name: "name", TypeHint: "str"},
			{Name: "value", TypeHint: "str"},
			{Name: "child", TypeHint: "List[GetDataResultChild]"},
		},
	}

	_, err := generateHierarchicalManyExecution(format, responseStruct, "postgres")
	if !errors.Is(err, snapsql.ErrHierarchicalNoParentPrimaryKey) {
		t.Fatalf("expected ErrHierarchicalNoParentPrimaryKey, got: %v", err)
	}
}

func TestGenerateQueryExecution_AutoDetectHierarchical(t *testing.T) {
	// Test that generateQueryExecution automatically detects hierarchical structure
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_users_with_posts",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
			{Name: "post__post_id", Type: "int", IsNullable: false},
			{Name: "post__title", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetUsersWithPostsResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
			{Name: "post", TypeHint: "List[GetUsersWithPostsResultPost]"},
		},
	}

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should generate hierarchical code
	if !strings.Contains(result.Code, "hierarchical aggregation") {
		t.Error("Expected hierarchical aggregation code, got simple iteration")
	}
}

func TestGenerateQueryExecution_SimpleMany(t *testing.T) {
	// Test that simple (non-hierarchical) many affinity still works
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_users",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "ListUsersResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
		},
	}

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should NOT generate hierarchical code
	if strings.Contains(result.Code, "hierarchical aggregation") {
		t.Error("Expected simple iteration code, got hierarchical aggregation")
	}

	// Should have simple yield
	if !strings.Contains(result.Code, "yield") {
		t.Error("Expected yield statement for many affinity")
	}
}
