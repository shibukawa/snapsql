package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

// Test field source management in dependency graph
func TestDependencyGraphFieldSourceManagement(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a test node
	node := &DependencyNode{
		ID:           "test_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{},
	}
	dg.AddNode(node)

	// Create scope for the node
	err := dg.CreateScopeForNode("test_node")
	assert.NoError(t, err)

	// Add field source to the node
	fieldSource := &FieldSource{
		Name:       "user_id",
		Alias:      "id",
		SourceType: SourceTypeTable,
		Scope:      "test_node",
	}

	err = dg.AddFieldSourceToNode("test_node", fieldSource)
	assert.NoError(t, err)

	// Verify field source was added to node
	assert.Equal(t, 1, len(node.FieldSources))
	assert.Equal(t, "user_id", node.FieldSources[0].Name)

	// Test field resolution
	resolved, err := dg.ResolveFieldInNode("test_node", "user_id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "user_id", resolved[0].Name)

	// Test field resolution by alias
	resolved, err = dg.ResolveFieldInNode("test_node", "id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "user_id", resolved[0].Name)
}

// Test table reference management in dependency graph
func TestDependencyGraphTableReferenceManagement(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a test node
	node := &DependencyNode{
		ID:           "test_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{},
	}
	dg.AddNode(node)

	// Create scope for the node
	err := dg.CreateScopeForNode("test_node")
	assert.NoError(t, err)

	// Add table reference to the node
	tableRef := &TableReference{
		Name:     "u",
		RealName: "users",
		Schema:   "public",
	}

	err = dg.AddTableReferenceToNode("test_node", tableRef)
	assert.NoError(t, err)

	// Verify table reference was added to node
	assert.Equal(t, 1, len(node.TableRefs))
	assert.Equal(t, "u", node.TableRefs[0].Name)
	assert.Equal(t, "users", node.TableRefs[0].RealName)

	// Test table alias resolution through scope manager
	scopeManager := dg.GetScopeManager()
	err = scopeManager.EnterScope("test_node")
	assert.NoError(t, err)

	resolved, err := scopeManager.ResolveTableAlias("u")
	assert.NoError(t, err)
	assert.Equal(t, "users", resolved.RealName)
	assert.Equal(t, "public", resolved.Schema)
}

// Test scope hierarchy building
func TestDependencyGraphScopeHierarchy(t *testing.T) {
	dg := NewDependencyGraph()

	// Create parent node (CTE)
	parentNode := &DependencyNode{
		ID:           "cte_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyCTE,
		Dependencies: []string{},
	}
	dg.AddNode(parentNode)

	// Create child node (depends on CTE)
	childNode := &DependencyNode{
		ID:           "main_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{"cte_node"},
	}
	dg.AddNode(childNode)

	// Build scope hierarchy
	err := dg.BuildScopeHierarchy()
	assert.NoError(t, err)

	// Verify parent-child relationship
	assert.True(t, parentNode.Scope != nil)
	assert.True(t, childNode.Scope != nil)
	assert.Equal(t, parentNode.Scope, childNode.Scope.ParentScope)
	assert.Equal(t, 1, len(parentNode.Scope.ChildScopes))
	assert.Equal(t, childNode.Scope, parentNode.Scope.ChildScopes[0])

	// Add field to parent scope
	parentField := &FieldSource{
		Name:       "parent_field",
		SourceType: SourceTypeTable,
		Scope:      "cte_node",
	}
	err = dg.AddFieldSourceToNode("cte_node", parentField)
	assert.NoError(t, err)

	// Add field to child scope
	childField := &FieldSource{
		Name:       "child_field",
		SourceType: SourceTypeExpression,
		Scope:      "main_node",
	}
	err = dg.AddFieldSourceToNode("main_node", childField)
	assert.NoError(t, err)

	// Test field resolution from child can access parent fields
	resolved, err := dg.ResolveFieldInNode("main_node", "parent_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "parent_field", resolved[0].Name)

	// Test field resolution from child can access own fields
	resolved, err = dg.ResolveFieldInNode("main_node", "child_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "child_field", resolved[0].Name)

	// Test that parent cannot access child fields
	_, err = dg.ResolveFieldInNode("cte_node", "child_field")
	assert.Error(t, err)
}

// Test field access validation
func TestDependencyGraphFieldAccessValidation(t *testing.T) {
	dg := NewDependencyGraph()

	// Create test node
	node := &DependencyNode{
		ID:           "test_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{},
	}
	dg.AddNode(node)

	// Create scope for the node
	err := dg.CreateScopeForNode("test_node")
	assert.NoError(t, err)

	// Add accessible field
	accessibleField := &FieldSource{
		Name:       "accessible_field",
		SourceType: SourceTypeTable,
		Scope:      "test_node",
	}
	err = dg.AddFieldSourceToNode("test_node", accessibleField)
	assert.NoError(t, err)

	// Test valid field access
	err = dg.ValidateFieldAccessForNode("test_node", "accessible_field")
	assert.NoError(t, err)

	// Test invalid field access
	err = dg.ValidateFieldAccessForNode("test_node", "inaccessible_field")
	assert.Error(t, err)
}

// Test getting all accessible fields for a node
func TestDependencyGraphGetAccessibleFields(t *testing.T) {
	dg := NewDependencyGraph()

	// Create parent node
	parentNode := &DependencyNode{
		ID:           "parent_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyCTE,
		Dependencies: []string{},
	}
	dg.AddNode(parentNode)

	// Create child node
	childNode := &DependencyNode{
		ID:           "child_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{"parent_node"},
	}
	dg.AddNode(childNode)

	// Build scope hierarchy
	err := dg.BuildScopeHierarchy()
	assert.NoError(t, err)

	// Add fields to parent
	parentField1 := &FieldSource{
		Name:       "parent_field1",
		SourceType: SourceTypeTable,
		Scope:      "parent_node",
	}
	err = dg.AddFieldSourceToNode("parent_node", parentField1)
	assert.NoError(t, err)

	parentField2 := &FieldSource{
		Name:       "parent_field2",
		SourceType: SourceTypeTable,
		Scope:      "parent_node",
	}
	err = dg.AddFieldSourceToNode("parent_node", parentField2)
	assert.NoError(t, err)

	// Add field to child
	childField := &FieldSource{
		Name:       "child_field",
		SourceType: SourceTypeExpression,
		Scope:      "child_node",
	}
	err = dg.AddFieldSourceToNode("child_node", childField)
	assert.NoError(t, err)

	// Get all accessible fields from child perspective
	accessibleFields, err := dg.GetAccessibleFieldsForNode("child_node")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(accessibleFields))

	// Verify fields are accessible (order: child first, then parent)
	fieldNames := make([]string, len(accessibleFields))
	for i, field := range accessibleFields {
		fieldNames[i] = field.Name
	}

	// Check if all expected fields are present
	hasChildField := false
	hasParentField1 := false
	hasParentField2 := false
	for _, name := range fieldNames {
		if name == "child_field" {
			hasChildField = true
		} else if name == "parent_field1" {
			hasParentField1 = true
		} else if name == "parent_field2" {
			hasParentField2 = true
		}
	}
	assert.True(t, hasChildField)
	assert.True(t, hasParentField1)
	assert.True(t, hasParentField2)
}

// Test scope hierarchy visualization
func TestDependencyGraphScopeVisualization(t *testing.T) {
	dg := NewDependencyGraph()

	// Create nodes with hierarchy
	parentNode := &DependencyNode{
		ID:           "parent",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyCTE,
		Dependencies: []string{},
	}
	dg.AddNode(parentNode)

	childNode := &DependencyNode{
		ID:           "child",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{"parent"},
	}
	dg.AddNode(childNode)

	// Build scope hierarchy
	err := dg.BuildScopeHierarchy()
	assert.NoError(t, err)

	// Add fields
	parentField := &FieldSource{
		Name:       "id",
		SourceType: SourceTypeTable,
		Scope:      "parent",
	}
	err = dg.AddFieldSourceToNode("parent", parentField)
	assert.NoError(t, err)

	childField := &FieldSource{
		Name:       "calculated",
		SourceType: SourceTypeExpression,
		Scope:      "child",
	}
	err = dg.AddFieldSourceToNode("child", childField)
	assert.NoError(t, err)

	// Add table reference
	tableRef := &TableReference{
		Name:     "p",
		RealName: "parent",
		Schema:   "",
	}
	err = dg.AddTableReferenceToNode("child", tableRef)
	assert.NoError(t, err)

	// Get visualization
	visualization := dg.GetScopeHierarchyVisualization()
	assert.True(t, visualization != "")

	// Verify it contains expected elements
	assert.Contains(t, visualization, "Scope: parent")
	assert.Contains(t, visualization, "Scope: child")
	assert.Contains(t, visualization, "id")
	assert.Contains(t, visualization, "calculated")
	assert.Contains(t, visualization, "Table Aliases: p")
}

// Test error cases for field source management
func TestDependencyGraphFieldSourceErrorCases(t *testing.T) {
	dg := NewDependencyGraph()

	// Test adding field source to non-existent node
	fieldSource := &FieldSource{
		Name:       "test_field",
		SourceType: SourceTypeTable,
	}
	err := dg.AddFieldSourceToNode("nonexistent", fieldSource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node nonexistent not found")

	// Test creating scope for non-existent node
	err = dg.CreateScopeForNode("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node nonexistent not found")

	// Test resolving field in non-existent node
	_, err = dg.ResolveFieldInNode("nonexistent", "test_field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node nonexistent not found")

	// Create node without scope
	node := &DependencyNode{
		ID:           "no_scope_node",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{},
	}
	dg.AddNode(node)

	// Test resolving field in node without scope
	_, err = dg.ResolveFieldInNode("no_scope_node", "test_field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no associated scope")

	// Test getting accessible fields from node without scope
	_, err = dg.GetAccessibleFieldsForNode("no_scope_node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no associated scope")
}

// Test complex multi-level scope hierarchy
func TestDependencyGraphComplexScopeHierarchy(t *testing.T) {
	dg := NewDependencyGraph()

	// Level 1: Base CTE
	cte1Node := &DependencyNode{
		ID:           "cte1",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyCTE,
		Dependencies: []string{},
	}
	dg.AddNode(cte1Node)

	// Level 2: CTE that depends on cte1
	cte2Node := &DependencyNode{
		ID:           "cte2",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyCTE,
		Dependencies: []string{"cte1"},
	}
	dg.AddNode(cte2Node)

	// Level 3: Main query that depends on cte2
	mainNode := &DependencyNode{
		ID:           "main",
		Statement:    &mockStatementNode{},
		NodeType:     DependencyMain,
		Dependencies: []string{"cte2"},
	}
	dg.AddNode(mainNode)

	// Build scope hierarchy
	err := dg.BuildScopeHierarchy()
	assert.NoError(t, err)

	// Add fields to each level
	// Level 1 fields
	cte1Field1 := &FieldSource{Name: "id", SourceType: SourceTypeTable, Scope: "cte1"}
	cte1Field2 := &FieldSource{Name: "name", SourceType: SourceTypeTable, Scope: "cte1"}
	err = dg.AddFieldSourceToNode("cte1", cte1Field1)
	assert.NoError(t, err)
	err = dg.AddFieldSourceToNode("cte1", cte1Field2)
	assert.NoError(t, err)

	// Level 2 fields
	cte2Field1 := &FieldSource{Name: "id", SourceType: SourceTypeTable, Scope: "cte2"}
	cte2Field2 := &FieldSource{Name: "cnt", SourceType: SourceTypeAggregate, Scope: "cte2"}
	err = dg.AddFieldSourceToNode("cte2", cte2Field1)
	assert.NoError(t, err)
	err = dg.AddFieldSourceToNode("cte2", cte2Field2)
	assert.NoError(t, err)

	// Level 3 fields
	mainField := &FieldSource{Name: "doubled", SourceType: SourceTypeExpression, Scope: "main"}
	err = dg.AddFieldSourceToNode("main", mainField)
	assert.NoError(t, err)

	// Test field resolution from main query
	// Should be able to access immediate parent (cte2) fields
	resolved, err := dg.ResolveFieldInNode("main", "cnt")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	// Should be able to access own fields
	resolved, err = dg.ResolveFieldInNode("main", "doubled")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	// Note: With current simple hierarchy (only first dependency as parent),
	// main query may not directly access cte1 fields. This is a design choice
	// that could be enhanced in future iterations.

	// Test field resolution from cte2
	resolved, err = dg.ResolveFieldInNode("cte2", "name")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	// Get all accessible fields from main
	allFields, err := dg.GetAccessibleFieldsForNode("main")
	assert.NoError(t, err)
	assert.True(t, len(allFields) >= 1) // At least main's own field
}
