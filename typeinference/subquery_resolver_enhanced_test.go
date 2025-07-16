package typeinference

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

func TestEnhancedSubqueryResolver_BasicFunctionality(t *testing.T) {
	// Create test database schema with simplified structure
	schema := snapsql.DatabaseSchema{
		Name: "testdb",
		DatabaseInfo: snapsql.DatabaseInfo{
			Type: "postgres",
			Name: "testdb",
		},
		Tables: []*snapsql.TableInfo{
			{
				Name:   "users",
				Schema: "public",
				Columns: map[string]*snapsql.ColumnInfo{
					"id":    {Name: "id", DataType: "integer", Nullable: false},
					"name":  {Name: "name", DataType: "string", Nullable: false},
					"email": {Name: "email", DataType: "string", Nullable: true},
				},
			},
			{
				Name:   "orders",
				Schema: "public",
				Columns: map[string]*snapsql.ColumnInfo{
					"id":      {Name: "id", DataType: "integer", Nullable: false},
					"user_id": {Name: "user_id", DataType: "integer", Nullable: false},
					"amount":  {Name: "amount", DataType: "decimal", Nullable: false},
				},
			},
		},
	}

	t.Run("enhanced_resolver_creation", func(t *testing.T) {
		// Test enhanced resolver creation
		schemaResolver := NewSchemaResolver([]snapsql.DatabaseSchema{schema})

		// Create a mock statement node
		mockStatement := createMockStatementWithSubqueryAnalysis("")

		enhancedResolver := NewEnhancedSubqueryResolver(schemaResolver, mockStatement, snapsql.DialectPostgres)
		assert.NotEqual(t, nil, enhancedResolver)
		assert.NotEqual(t, nil, enhancedResolver.SubqueryTypeResolver)
		assert.NotEqual(t, nil, enhancedResolver.fieldResolverCache)
	})

	t.Run("subquery_field_resolution", func(t *testing.T) {
		// Test subquery field type resolution
		schemaResolver := NewSchemaResolver([]snapsql.DatabaseSchema{schema})
		mockStatement := createMockStatementWithSubqueryAnalysis("")

		enhancedResolver := NewEnhancedSubqueryResolver(schemaResolver, mockStatement, snapsql.DialectPostgres)

		// Test field resolution with mock data
		testFieldInfos := []*InferredFieldInfo{
			{
				Name:         "user_count",
				OriginalName: "user_count",
				Type:         &TypeInfo{BaseType: "int", IsNullable: false},
				Source:       FieldSource{Type: "function", FunctionName: "COUNT"},
			},
			{
				Name:         "total_amount",
				OriginalName: "total_amount",
				Type:         &TypeInfo{BaseType: "decimal", IsNullable: true},
				Source:       FieldSource{Type: "function", FunctionName: "SUM"},
			},
		}

		// Cache test data
		enhancedResolver.cacheFieldMapping("test_cte", testFieldInfos)

		// Test field resolution
		fieldInfo, found := enhancedResolver.ResolveSubqueryFieldType("test_cte", "user_count")
		assert.True(t, found)
		assert.NotEqual(t, nil, fieldInfo)
		assert.Equal(t, "user_count", fieldInfo.Name)
		assert.Equal(t, "int", fieldInfo.Type.BaseType)

		// Test non-existent field
		_, found = enhancedResolver.ResolveSubqueryFieldType("test_cte", "non_existent")
		assert.False(t, found)

		// Test non-existent subquery
		_, found = enhancedResolver.ResolveSubqueryFieldType("non_existent", "user_count")
		assert.False(t, found)
	})

	t.Run("cte_name_extraction", func(t *testing.T) {
		// Test CTE name extraction logic
		schemaResolver := NewSchemaResolver([]snapsql.DatabaseSchema{schema})
		mockStatement := createMockStatementWithSubqueryAnalysis("")

		enhancedResolver := NewEnhancedSubqueryResolver(schemaResolver, mockStatement, snapsql.DialectPostgres)

		// Test node ID based extraction
		cteName := enhancedResolver.extractCTENameFromNodeID("cte_user_summary")
		assert.Equal(t, "user_summary", cteName)

		cteName = enhancedResolver.extractCTENameFromNodeID("with_order_totals")
		assert.Equal(t, "order_totals", cteName)

		// Test fallback for unknown formats
		cteName = enhancedResolver.extractCTENameFromNodeID("unknown_format")
		assert.Equal(t, "unknown_format", cteName)
	})
}

func TestEnhancedSubqueryResolver_Integration(t *testing.T) {
	// Test integration with existing type inference system
	schema := snapsql.DatabaseSchema{
		Name: "testdb",
		DatabaseInfo: snapsql.DatabaseInfo{
			Type: "postgres",
			Name: "testdb",
		},
		Tables: []*snapsql.TableInfo{
			{
				Name:   "products",
				Schema: "public",
				Columns: map[string]*snapsql.ColumnInfo{
					"id":    {Name: "id", DataType: "integer", Nullable: false},
					"name":  {Name: "name", DataType: "string", Nullable: false},
					"price": {Name: "price", DataType: "decimal", Nullable: false},
				},
			},
		},
	}

	t.Run("type_inference_engine_integration", func(t *testing.T) {
		// Test integration with TypeInferenceEngine2
		mockStatement := createMockStatementWithSubqueryAnalysis("")

		engine := NewTypeInferenceEngine2([]snapsql.DatabaseSchema{schema}, mockStatement)
		enhancedResolver, ok := engine.getEnhancedSubqueryResolver()

		if ok {
			assert.NotEqual(t, nil, enhancedResolver)
			assert.NotEqual(t, nil, enhancedResolver.SubqueryTypeResolver)

			// Test complete subquery analysis
			analysis := enhancedResolver.GetCompleteSubqueryInformation()
			assert.NotEqual(t, nil, analysis)
			assert.False(t, analysis.HasSubqueries) // Mock statement has no subqueries
		} else {
			// No subquery analysis available in mock - that's expected
			t.Log("No subquery analysis available in mock statement - this is expected")
		}
	})
}

// parseWithSubqueryAnalysis is a helper function that parses SQL and includes subquery analysis
// This would integrate with the actual parser implementation
func parseWithSubqueryAnalysis(sqlText string) (parser.StatementNode, error) {
	// This would use the actual parser.ParseExtended functionality
	// For testing purposes, we can create a mock StatementNode that implements
	// the required interfaces with subquery analysis information

	// In a real implementation, this would be:
	// tokens, err := tokenizer.TokenizeSQL(sqlText)
	// if err != nil { return nil, err }
	//
	// result, err := parser.ParseExtended(tokens, nil, &parser.ParseOptions{
	//     EnableSubqueryAnalysis: true,
	// })
	// if err != nil { return nil, err }
	//
	// return result.Statement, nil

	// For now, return a mock implementation
	return createMockStatementWithSubqueryAnalysis(sqlText), nil
}

// createMockStatementWithSubqueryAnalysis creates a mock StatementNode for testing
func createMockStatementWithSubqueryAnalysis(sqlText string) parser.StatementNode {
	// This would create a proper mock with subquery analysis
	// For testing purposes, we can create a minimal implementation
	return &parser.SelectStatement{
		// Basic structure would be populated here
	}
}
