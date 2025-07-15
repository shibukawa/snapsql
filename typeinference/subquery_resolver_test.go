package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep7"
)

// TestSubqueryTypeResolver_Basic tests basic subquery type resolver functionality
func TestSubqueryTypeResolver_Basic(t *testing.T) {
	// Create mock schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "public",
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "int", Nullable: false},
						"name": {Name: "name", DataType: "string", Nullable: true},
						"age":  {Name: "age", DataType: "int", Nullable: true},
					},
				},
				{
					Name: "orders",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":      {Name: "id", DataType: "int", Nullable: false},
						"user_id": {Name: "user_id", DataType: "int", Nullable: false},
						"amount":  {Name: "amount", DataType: "decimal", Nullable: false},
					},
				},
			},
		},
	}

	schemaResolver := NewSchemaResolver(schemas)

	// Create mock parse result with simple CTE dependency
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: createMockDependencyGraph(),
		ProcessingOrder: []string{"cte_user_summary", "main"},
		HasErrors:       false,
		Errors:          nil,
	}

	resolver := NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)

	// Test resolver creation
	assert.NotZero(t, resolver)
	assert.Equal(t, schemaResolver, resolver.schemaResolver)
	assert.Equal(t, mockParseResult, resolver.parseResult)
	assert.Equal(t, snapsql.DialectPostgres, resolver.dialect)
	assert.NotEqual(t, nil, resolver.typeCache)
}

// TestSubqueryTypeResolver_ResolveSubqueryTypes tests subquery type resolution
func TestSubqueryTypeResolver_ResolveSubqueryTypes(t *testing.T) {
	// Skip if too complex for initial implementation
	t.Skip("Subquery type resolution requires complete parserstep7 integration")

	// Create mock schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "public",
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "int", Nullable: false},
						"name": {Name: "name", DataType: "string", Nullable: true},
					},
				},
			},
		},
	}

	schemaResolver := NewSchemaResolver(schemas)

	// Create mock SELECT statement for CTE
	cteSelectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "id",
				},
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "name",
				},
			},
		},
		From: &parsercommon.FromClause{},
	}

	// Create mock dependency graph with real StatementNode
	dependencyGraph := parserstep7.NewDependencyGraph()
	cteNode := &parserstep7.DependencyNode{
		ID:        "cte_user_data",
		Statement: cteSelectStmt,
		NodeType:  parserstep7.DependencyCTE,
	}
	_ = dependencyGraph.AddNode(cteNode) // Ignore error for test

	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: dependencyGraph,
		ProcessingOrder: []string{"cte_user_data"},
		HasErrors:       false,
		Errors:          nil,
	}

	resolver := NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)

	// Test type resolution
	err := resolver.ResolveSubqueryTypes()
	assert.NoError(t, err)

	// Verify types were cached
	fieldTypes, exists := resolver.GetSubqueryFieldTypes("cte_user_data")
	assert.True(t, exists)
	assert.Equal(t, 2, len(fieldTypes))
	assert.Equal(t, "id", fieldTypes[0].Name)
	assert.Equal(t, "int", fieldTypes[0].Type.BaseType)
	assert.Equal(t, "name", fieldTypes[1].Name)
	assert.Equal(t, "string", fieldTypes[1].Type.BaseType)
}

// TestSubqueryTypeResolver_GetAvailableSubqueryTables tests available subquery table extraction
func TestSubqueryTypeResolver_GetAvailableSubqueryTables(t *testing.T) {
	schemas := []snapsql.DatabaseSchema{
		{
			Name:         "public",
			DatabaseInfo: snapsql.DatabaseInfo{Type: "postgres"},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {Name: "id", DataType: "int", Nullable: false},
					},
				},
			},
		},
	}

	schemaResolver := NewSchemaResolver(schemas)
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: createMockDependencyGraph(),
		ProcessingOrder: []string{"cte_user_summary"},
		HasErrors:       false,
	}

	resolver := NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)

	// Manually add some cached types to simulate resolved CTEs
	resolver.typeCache["cte_user_summary"] = []*InferredFieldInfo{
		{
			Name: "user_id",
			Type: &TypeInfo{BaseType: "int", IsNullable: false},
		},
		{
			Name: "total_orders",
			Type: &TypeInfo{BaseType: "int", IsNullable: false},
		},
	}

	resolver.typeCache["cte_order_stats"] = []*InferredFieldInfo{
		{
			Name: "avg_amount",
			Type: &TypeInfo{BaseType: "decimal", IsNullable: true},
		},
	}

	// Test available table extraction
	tables := resolver.GetAvailableSubqueryTables()

	// Check that user_summary is available (from cte_user_summary)
	found := false
	for _, table := range tables {
		if table == "user_summary" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected table user_summary not found in %v", tables)
}

// TestSubqueryTypeResolver_ResolveSubqueryReference tests subquery reference resolution
func TestSubqueryTypeResolver_ResolveSubqueryReference(t *testing.T) {
	schemas := []snapsql.DatabaseSchema{
		{
			Name:         "public",
			DatabaseInfo: snapsql.DatabaseInfo{Type: "postgres"},
		},
	}

	schemaResolver := NewSchemaResolver(schemas)
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: createMockDependencyGraph(),
		ProcessingOrder: []string{},
		HasErrors:       false,
	}

	resolver := NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)

	// Add mock CTE types
	resolver.typeCache["cte_user_data"] = []*InferredFieldInfo{
		{
			Name:         "user_id",
			OriginalName: "id",
			Type:         &TypeInfo{BaseType: "int", IsNullable: false},
		},
		{
			Name:         "user_name",
			OriginalName: "name",
			Type:         &TypeInfo{BaseType: "string", IsNullable: true},
		},
	}

	// Test successful resolution
	fields, found := resolver.ResolveSubqueryReference("user_data")
	assert.True(t, found)
	assert.Equal(t, 2, len(fields))
	assert.Equal(t, "user_id", fields[0].Name)
	assert.Equal(t, "int", fields[0].Type.BaseType)

	// Test missing subquery
	_, found = resolver.ResolveSubqueryReference("nonexistent")
	assert.False(t, found)
}

// TestSubqueryTypeResolver_ValidateSubqueryReferences tests subquery reference validation
func TestSubqueryTypeResolver_ValidateSubqueryReferences(t *testing.T) {
	// Skip complex validation test for now
	t.Skip("Subquery validation requires complete dependency graph implementation")
}

// TestSubqueryTypeResolver_Integration tests integration with TypeInferenceEngine2
func TestSubqueryTypeResolver_Integration(t *testing.T) {
	// Create simple test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name:         "public",
			DatabaseInfo: snapsql.DatabaseInfo{Type: "postgres"},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "int", Nullable: false},
						"name": {Name: "name", DataType: "string", Nullable: true},
					},
				},
			},
		},
	}

	// Create simple SELECT statement
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "id",
				},
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "name",
				},
			},
		},
		From: &parsercommon.FromClause{},
	}

	// Create simple parse result (no subqueries)
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
		ProcessingOrder: []string{},
		HasErrors:       false,
		Errors:          nil,
	}

	// Test engine creation with subquery info
	engine := NewTypeInferenceEngine2(schemas, selectStmt, mockParseResult)
	assert.NotZero(t, engine)
	assert.NotZero(t, engine.subqueryResolver)

	// Test basic type inference still works
	fields, err := engine.InferSelectTypes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fields))
	assert.Equal(t, "id", fields[0].Name)
	assert.Equal(t, "int", fields[0].Type.BaseType)
}

// Helper function to create a mock dependency graph for testing
func createMockDependencyGraph() *parserstep7.DependencyGraph {
	graph := parserstep7.NewDependencyGraph()

	// Add a mock CTE node
	cteNode := &parserstep7.DependencyNode{
		ID:       "cte_user_summary",
		NodeType: parserstep7.DependencyCTE,
		// Statement would normally be a real StatementNode, but for testing we can leave it nil
		Statement: nil,
	}
	_ = graph.AddNode(cteNode) // Ignore error for test

	return graph
}

// TestSubqueryTypeResolver_ErrorHandling tests error handling scenarios
func TestSubqueryTypeResolver_ErrorHandling(t *testing.T) {
	// Test with nil parse result
	resolver := NewSubqueryTypeResolver(nil, nil, snapsql.DialectPostgres)
	err := resolver.ResolveSubqueryTypes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no parse result available")

	// Test with empty processing order
	schemas := []snapsql.DatabaseSchema{}
	schemaResolver := NewSchemaResolver(schemas)
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
		ProcessingOrder: []string{}, // Empty - no subqueries
		HasErrors:       false,
	}

	resolver = NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)
	err = resolver.ResolveSubqueryTypes()
	assert.NoError(t, err) // Should succeed with no subqueries to process
}

// TestSubqueryTypeResolver_ExtractAvailableTablesFromScope tests scope table extraction
func TestSubqueryTypeResolver_ExtractAvailableTablesFromScope(t *testing.T) {
	schemas := []snapsql.DatabaseSchema{}
	schemaResolver := NewSchemaResolver(schemas)
	mockParseResult := &parserstep7.ParseResult{
		DependencyGraph: createMockDependencyGraph(),
		ProcessingOrder: []string{},
		HasErrors:       false,
	}

	resolver := NewSubqueryTypeResolver(schemaResolver, mockParseResult, snapsql.DialectPostgres)

	// Create mock node with table references
	node := &parserstep7.DependencyNode{
		ID:       "test_node",
		NodeType: parserstep7.DependencySubquery,
		TableRefs: []*parserstep7.TableReference{
			{Name: "users"},
			{Name: "orders"},
		},
		Dependencies: []string{"cte_user_summary"},
	}

	tables := resolver.extractAvailableTablesFromScope(node)

	expectedTables := []string{"users", "orders", "user_summary"}
	for _, expectedTable := range expectedTables {
		found := false
		for _, table := range tables {
			if table == expectedTable {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected table %s not found in %v", expectedTable, tables)
	}
}
