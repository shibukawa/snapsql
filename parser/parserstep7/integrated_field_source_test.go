package parserstep7

import (
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Test integrated parser with field source management
func TestSubqueryParserIntegratedFieldSourceManagement(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Create a statement with CTE
	cte := &cmn.CTEDefinition{
		Name: "user_stats",
		Select: &mockStatementNode{
			cte: &cmn.WithClause{
				CTEs: []cmn.CTEDefinition{},
			},
		},
	}

	mainStmt := &mockStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{*cte},
		},
	}

	// Parse the statement
	result, err := parser.ParseStatement(mainStmt)
	assert.NoError(t, err)
	assert.False(t, result.HasErrors)
	assert.True(t, result.DependencyGraph != nil)

	// Verify scope hierarchy was built
	scopeVisualization := parser.GetScopeVisualization()
	assert.True(t, scopeVisualization != "")

	// Test adding field sources and table references
	nodes := result.DependencyGraph.GetAllNodes()
	assert.True(t, len(nodes) > 0)

	// Get the first node ID for testing
	var firstNodeID string
	for nodeID := range nodes {
		firstNodeID = nodeID
		break
	}

	// Add a field source
	fieldSource := &FieldSource{
		Name:       "user_id",
		Alias:      "id",
		SourceType: SourceTypeTable,
		Scope:      firstNodeID,
	}
	err = parser.AddFieldSourceToNode(firstNodeID, fieldSource)
	assert.NoError(t, err)

	// Add a table reference
	tableRef := &TableReference{
		Name:     "u",
		RealName: "users",
		Schema:   "public",
	}
	err = parser.AddTableReferenceToNode(firstNodeID, tableRef)
	assert.NoError(t, err)

	// Test field resolution
	resolved, err := parser.ResolveFieldReference(firstNodeID, "user_id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "user_id", resolved[0].Name)

	// Test field access validation
	err = parser.ValidateFieldAccess(firstNodeID, "user_id")
	assert.NoError(t, err)

	err = parser.ValidateFieldAccess(firstNodeID, "nonexistent_field")
	assert.Error(t, err)

	// Test getting accessible fields
	accessibleFields, err := parser.GetAccessibleFieldsForNode(firstNodeID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(accessibleFields))
	assert.Equal(t, "user_id", accessibleFields[0].Name)
}

// Test ParseResult field source management methods
func TestParseResultFieldSourceMethods(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Create a simple statement
	stmt := &mockStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{},
		},
	}

	// Parse the statement
	result, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)
	assert.False(t, result.HasErrors)

	// Get a node ID
	nodes := result.DependencyGraph.GetAllNodes()
	assert.True(t, len(nodes) > 0)

	var firstNodeID string
	for nodeID := range nodes {
		firstNodeID = nodeID
		break
	}

	// Add field source through the dependency graph
	fieldSource := &FieldSource{
		Name:       "test_field",
		SourceType: SourceTypeExpression,
		Scope:      firstNodeID,
	}
	err = result.DependencyGraph.AddFieldSourceToNode(firstNodeID, fieldSource)
	assert.NoError(t, err)

	// Test ParseResult methods
	accessibleFields, err := result.GetFieldSourcesForNode(firstNodeID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(accessibleFields))
	assert.Equal(t, "test_field", accessibleFields[0].Name)

	// Test field validation through ParseResult
	err = result.ValidateFieldAccessForNode(firstNodeID, "test_field")
	assert.NoError(t, err)

	err = result.ValidateFieldAccessForNode(firstNodeID, "invalid_field")
	assert.Error(t, err)

	// Test field resolution through ParseResult
	resolved, err := result.ResolveFieldForNode(firstNodeID, "test_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "test_field", resolved[0].Name)

	// Test scope hierarchy visualization
	hierarchy := result.GetScopeHierarchy()
	assert.True(t, hierarchy != "")
	assert.Contains(t, hierarchy, "test_field")
}

// Test error cases for integrated field source management
func TestSubqueryParserIntegratedFieldSourceErrors(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Test methods with no statement parsed
	err := parser.ValidateFieldAccess("nonexistent", "field")
	assert.Error(t, err)

	_, err = parser.ResolveFieldReference("nonexistent", "field")
	assert.Error(t, err)

	_, err = parser.GetAccessibleFieldsForNode("nonexistent")
	assert.Error(t, err)

	// Test adding to non-existent node
	fieldSource := &FieldSource{
		Name:       "test_field",
		SourceType: SourceTypeTable,
	}
	err = parser.AddFieldSourceToNode("nonexistent", fieldSource)
	assert.Error(t, err)

	tableRef := &TableReference{
		Name:     "t",
		RealName: "test",
	}
	err = parser.AddTableReferenceToNode("nonexistent", tableRef)
	assert.Error(t, err)
}

// Test ParseResult error cases
func TestParseResultFieldSourceErrorCases(t *testing.T) {
	// Test ParseResult with nil dependency graph
	result := &ParseResult{
		DependencyGraph: nil,
		ProcessingOrder: []string{},
		HasErrors:       false,
		Errors:          nil,
	}

	_, err := result.GetFieldSourcesForNode("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dependency graph available")

	err = result.ValidateFieldAccessForNode("test", "field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dependency graph available")

	_, err = result.ResolveFieldForNode("test", "field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dependency graph available")

	hierarchy := result.GetScopeHierarchy()
	assert.Equal(t, "No dependency graph available", hierarchy)
}

// Test complex scenario with multiple nodes and field sources
func TestSubqueryParserIntegratedComplexFieldSourceScenario(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Create a statement with nested CTEs
	innerCTE := &cmn.CTEDefinition{
		Name: "inner_cte",
		Select: &mockStatementNode{
			cte: &cmn.WithClause{
				CTEs: []cmn.CTEDefinition{},
			},
		},
	}

	outerCTE := &cmn.CTEDefinition{
		Name: "outer_cte",
		Select: &mockStatementNode{
			cte: &cmn.WithClause{
				CTEs: []cmn.CTEDefinition{*innerCTE},
			},
		},
	}

	mainStmt := &mockStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{*outerCTE},
		},
	}

	// Parse the statement
	result, err := parser.ParseStatement(mainStmt)
	assert.NoError(t, err)
	assert.False(t, result.HasErrors)

	// Get all nodes
	nodes := result.DependencyGraph.GetAllNodes()
	assert.True(t, len(nodes) >= 1)

	// Add field sources to different nodes
	nodeCount := 0
	for nodeID := range nodes {
		fieldSource := &FieldSource{
			Name:       fmt.Sprintf("field_%d", nodeCount),
			SourceType: SourceTypeTable,
			Scope:      nodeID,
		}
		err = parser.AddFieldSourceToNode(nodeID, fieldSource)
		assert.NoError(t, err)

		tableRef := &TableReference{
			Name:     fmt.Sprintf("t%d", nodeCount),
			RealName: fmt.Sprintf("table_%d", nodeCount),
			Schema:   "public",
		}
		err = parser.AddTableReferenceToNode(nodeID, tableRef)
		assert.NoError(t, err)

		nodeCount++
		if nodeCount >= 3 { // Limit to avoid excessive testing
			break
		}
	}

	// Test field resolution across nodes
	for nodeID := range nodes {
		accessibleFields, err := parser.GetAccessibleFieldsForNode(nodeID)
		assert.NoError(t, err)
		assert.True(t, len(accessibleFields) >= 1)
		break // Test first node only
	}

	// Test scope visualization includes all added elements
	scopeVisualization := parser.GetScopeVisualization()
	assert.True(t, scopeVisualization != "")
	assert.Contains(t, scopeVisualization, "field_0")

	// Test dependency visualization
	depVisualization := parser.GetDependencyVisualization()
	assert.True(t, depVisualization != "")
	assert.Contains(t, depVisualization, "Dependency Graph:")
}

// Test parser reset functionality with field sources
func TestSubqueryParserIntegratedResetWithFieldSources(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Parse a statement and add field sources
	stmt := &mockStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{},
		},
	}

	result, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)
	assert.False(t, result.HasErrors)

	// Add field source
	nodes := result.DependencyGraph.GetAllNodes()
	var firstNodeID string
	for nodeID := range nodes {
		firstNodeID = nodeID
		break
	}

	fieldSource := &FieldSource{
		Name:       "test_field",
		SourceType: SourceTypeTable,
		Scope:      firstNodeID,
	}
	err = parser.AddFieldSourceToNode(firstNodeID, fieldSource)
	assert.NoError(t, err)

	// Verify field exists
	accessibleFields, err := parser.GetAccessibleFieldsForNode(firstNodeID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(accessibleFields))

	// Reset parser
	parser.Reset()

	// Verify field sources are cleared
	err = parser.ValidateFieldAccess(firstNodeID, "test_field")
	assert.Error(t, err) // Should fail because parser was reset

	// Parse new statement and verify clean state
	result2, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)
	assert.False(t, result2.HasErrors)

	// Get new node ID (should be different or clean)
	newNodes := result2.DependencyGraph.GetAllNodes()
	assert.True(t, len(newNodes) > 0)

	for newNodeID := range newNodes {
		accessibleFields, err := parser.GetAccessibleFieldsForNode(newNodeID)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(accessibleFields)) // Should be empty after reset
		break
	}
}
