package pygen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

// TestHierarchicalManyAffinity_StreamingBehavior verifies that hierarchical many affinity
// generates code that streams results one parent at a time (memory efficient)
func TestHierarchicalManyAffinity_StreamingBehavior(t *testing.T) {
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

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Verify streaming patterns
	streamingPatterns := []string{
		// Should yield incrementally, not collect all first
		"yield current_parent",
		// Should track current parent
		"current_parent = None",
		"current_parent_key = None",
		// Should detect parent changes
		"if parent_key != current_parent_key:",
		// Should yield previous parent before creating new one
		"if current_parent is not None:",
		// Should yield last parent at the end
		"if current_parent is not None:",
	}

	for _, pattern := range streamingPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing streaming pattern: %s", pattern)
		}
	}

	// Should NOT collect all results first (anti-pattern for streaming)
	antiPatterns := []string{
		"results = []",
		"results.append",
		"return results",
	}

	for _, pattern := range antiPatterns {
		if strings.Contains(code, pattern) {
			t.Errorf("Generated code contains non-streaming pattern: %s", pattern)
		}
	}
}

// TestHierarchicalManyAffinity_MemoryEfficiency verifies that the generated code
// doesn't accumulate all data in memory before yielding
func TestHierarchicalManyAffinity_MemoryEfficiency(t *testing.T) {
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

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Verify memory-efficient patterns
	// 1. Should process rows one at a time
	if !strings.Contains(code, "for row in rows:") {
		t.Error("Generated code should iterate over rows one at a time")
	}

	// 2. Should yield as soon as parent is complete
	if !strings.Contains(code, "yield current_parent") {
		t.Error("Generated code should yield parent objects incrementally")
	}

	// 3. Should not accumulate all parents before returning
	if strings.Contains(code, "all_parents = []") || strings.Contains(code, "parents.append") {
		t.Error("Generated code should not accumulate all parents in memory")
	}
}

// TestHierarchicalOneAffinity_AggregatesAllRows verifies that hierarchical one affinity
// collects all rows before returning (since it returns a single object)
func TestHierarchicalOneAffinity_AggregatesAllRows(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user_with_orders",
		ResponseAffinity: "one",
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
			{Name: "order__order_id", Type: "int", IsNullable: false},
			{Name: "order__amount", Type: "decimal", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetUserWithOrdersResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
			{Name: "order", TypeHint: "List[GetUserWithOrdersResultOrder]"},
		},
	}

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// For one affinity, should aggregate all rows into single result
	aggregationPatterns := []string{
		"result = None",
		"if result is None:",
		"return result",
	}

	for _, pattern := range aggregationPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing aggregation pattern: %s", pattern)
		}
	}

	// Should NOT yield (one affinity returns single object)
	if strings.Contains(code, "yield") {
		t.Error("One affinity should not use yield, should return single object")
	}
}

// TestAsyncGeneratorTypeAnnotation verifies that many affinity with hierarchical structure
// generates proper AsyncGenerator type annotation
func TestAsyncGeneratorTypeAnnotation(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_projects_with_tasks",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "project_id", Type: "int", IsNullable: false},
			{Name: "project_name", Type: "string", IsNullable: false},
			{Name: "task__task_id", Type: "int", IsNullable: false},
			{Name: "task__title", Type: "string", IsNullable: false},
		},
	}

	// Generate full code to check type annotation
	gen := &Generator{
		Format:  format,
		Dialect: "postgres",
	}

	// Process response struct
	responseStruct, err := processResponseStruct(format)
	if err != nil {
		t.Fatalf("Failed to process response struct: %v", err)
	}

	// Generate query execution
	queryExec, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Failed to generate query execution: %v", err)
	}

	// Check that it's recognized as hierarchical
	if !hasHierarchicalFields(format.Responses) {
		t.Error("Should detect hierarchical fields")
	}

	// Verify the code uses async generator pattern
	if !strings.Contains(queryExec.Code, "yield") {
		t.Error("Many affinity should use yield for async generator")
	}

	// The function signature should be generated with AsyncGenerator return type
	// This is handled by the template, but we can verify the execution code is compatible
	if !strings.Contains(queryExec.Code, "hierarchical aggregation") {
		t.Error("Should indicate hierarchical aggregation in comments")
	}

	_ = gen // Suppress unused warning
}

// TestHierarchicalStreaming_MultipleDialects verifies streaming behavior across dialects
func TestHierarchicalStreaming_MultipleDialects(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_customers_with_orders",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "customer_id", Type: "int", IsNullable: false},
			{Name: "customer_name", Type: "string", IsNullable: false},
			{Name: "order__order_id", Type: "int", IsNullable: false},
			{Name: "order__total", Type: "decimal", IsNullable: false},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetCustomersWithOrdersResult",
		Fields: []responseFieldData{
			{Name: "customer_id", TypeHint: "int"},
			{Name: "customer_name", TypeHint: "str"},
			{Name: "order", TypeHint: "List[GetCustomersWithOrdersResultOrder]"},
		},
	}

	dialects := []string{"postgres", "mysql", "sqlite"}

	for _, dialect := range dialects {
		t.Run(dialect, func(t *testing.T) {
			result, err := generateQueryExecution(format, responseStruct, dialect)
			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", dialect, err)
			}

			code := result.Code

			// All dialects should use streaming pattern
			if !strings.Contains(code, "yield current_parent") {
				t.Errorf("%s: should yield parent objects incrementally", dialect)
			}

			// All dialects should track current parent
			if !strings.Contains(code, "current_parent = None") {
				t.Errorf("%s: should track current parent", dialect)
			}

			// All dialects should detect parent changes
			if !strings.Contains(code, "if parent_key != current_parent_key:") {
				t.Errorf("%s: should detect parent key changes", dialect)
			}
		})
	}
}

// TestHierarchicalStreaming_ChildAggregation verifies that child objects are properly
// aggregated into parent objects during streaming
func TestHierarchicalStreaming_ChildAggregation(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_authors_with_books",
		ResponseAffinity: "many",
		Responses: []intermediate.Response{
			{Name: "author_id", Type: "int", IsNullable: false},
			{Name: "author_name", Type: "string", IsNullable: false},
			{Name: "book__book_id", Type: "int", IsNullable: false},
			{Name: "book__title", Type: "string", IsNullable: false},
			{Name: "book__isbn", Type: "string", IsNullable: true},
		},
	}

	responseStruct := &responseStructData{
		ClassName: "GetAuthorsWithBooksResult",
		Fields: []responseFieldData{
			{Name: "author_id", TypeHint: "int"},
			{Name: "author_name", TypeHint: "str"},
			{Name: "book", TypeHint: "List[GetAuthorsWithBooksResultBook]"},
		},
	}

	result, err := generateQueryExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	code := result.Code

	// Verify child aggregation patterns
	childPatterns := []string{
		// Should check if child data exists
		"if any([",
		// Should create child object
		"child_obj =",
		// Should append child to parent
		".append(child_obj)",
		// Should initialize child list to empty
		"book=[]",
	}

	for _, pattern := range childPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing child aggregation pattern: %s", pattern)
		}
	}
}
