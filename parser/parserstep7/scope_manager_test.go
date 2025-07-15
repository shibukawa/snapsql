package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

// Test basic scope creation and navigation
func TestScopeManagerBasicOperations(t *testing.T) {
	sm := NewScopeManager()

	// Test initial state
	assert.Equal(t, "root", sm.GetCurrentScope().ID)
	assert.Equal(t, (*Scope)(nil), sm.GetCurrentScope().ParentScope)

	// Create child scope
	childScope := sm.CreateScope("child1")
	assert.Equal(t, "child1", childScope.ID)
	assert.Equal(t, sm.rootScope, childScope.ParentScope)

	// Enter child scope
	err := sm.EnterScope("child1")
	assert.NoError(t, err)
	assert.Equal(t, "child1", sm.GetCurrentScope().ID)

	// Exit to parent scope
	err = sm.ExitScope()
	assert.NoError(t, err)
	assert.Equal(t, "root", sm.GetCurrentScope().ID)
}

// Test scope hierarchy creation
func TestScopeHierarchy(t *testing.T) {
	sm := NewScopeManager()

	// Create nested scopes
	child1 := sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	child2 := sm.CreateScope("child2")
	err = sm.EnterScope("child2")
	assert.NoError(t, err)

	grandchild := sm.CreateScope("grandchild")

	// Verify hierarchy
	assert.Equal(t, "child2", sm.GetCurrentScope().ID)
	assert.Equal(t, child1, sm.GetCurrentScope().ParentScope)
	assert.Equal(t, sm.rootScope, child1.ParentScope)
	assert.Equal(t, child2, grandchild.ParentScope)
}

// Test field resolution
func TestFieldResolution(t *testing.T) {
	sm := NewScopeManager()

	// Add field to root scope
	rootField := &FieldSource{
		Name:       "root_field",
		Alias:      "",
		SourceType: SourceTypeTable,
		Scope:      "root",
	}
	sm.AddFieldToScope(rootField)

	// Create child scope and add field
	sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	childField := &FieldSource{
		Name:       "child_field",
		Alias:      "cf",
		SourceType: SourceTypeExpression,
		Scope:      "child1",
	}
	sm.AddFieldToScope(childField)

	// Test field resolution from child scope
	resolved, err := sm.ResolveField("root_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "root_field", resolved[0].Name)

	resolved, err = sm.ResolveField("child_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "child_field", resolved[0].Name)

	// Test alias resolution
	resolved, err = sm.ResolveField("cf")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "child_field", resolved[0].Name)

	// Test non-existent field
	_, err = sm.ResolveField("nonexistent")
	assert.Error(t, err)
}

// Test table alias management
func TestTableAliasManagement(t *testing.T) {
	sm := NewScopeManager()

	// Add table alias to root scope
	tableRef := &TableReference{
		Name:     "u",
		RealName: "users",
		Schema:   "public",
	}
	sm.AddTableAlias("u", tableRef)

	// Test alias resolution
	resolved, err := sm.ResolveTableAlias("u")
	assert.NoError(t, err)
	assert.Equal(t, "users", resolved.RealName)
	assert.Equal(t, "public", resolved.Schema)

	// Create child scope with different alias
	sm.CreateScope("child1")
	err = sm.EnterScope("child1")
	assert.NoError(t, err)

	childTableRef := &TableReference{
		Name:     "p",
		RealName: "posts",
		Schema:   "public",
	}
	sm.AddTableAlias("p", childTableRef)

	// Test both aliases are accessible
	resolved, err = sm.ResolveTableAlias("u")
	assert.NoError(t, err)
	assert.Equal(t, "users", resolved.RealName)

	resolved, err = sm.ResolveTableAlias("p")
	assert.NoError(t, err)
	assert.Equal(t, "posts", resolved.RealName)

	// Test non-existent alias
	_, err = sm.ResolveTableAlias("nonexistent")
	assert.Error(t, err)
}

// Test subquery alias management
func TestSubqueryAliasManagement(t *testing.T) {
	sm := NewScopeManager()

	// Add subquery alias
	sm.AddSubqueryAlias("sq1", "subquery_001")

	// Test alias resolution
	resolved, err := sm.ResolveSubqueryAlias("sq1")
	assert.NoError(t, err)
	assert.Equal(t, "subquery_001", resolved)

	// Create child scope
	sm.CreateScope("child1")
	err = sm.EnterScope("child1")
	assert.NoError(t, err)

	sm.AddSubqueryAlias("sq2", "subquery_002")

	// Test both aliases are accessible
	resolved, err = sm.ResolveSubqueryAlias("sq1")
	assert.NoError(t, err)
	assert.Equal(t, "subquery_001", resolved)

	resolved, err = sm.ResolveSubqueryAlias("sq2")
	assert.NoError(t, err)
	assert.Equal(t, "subquery_002", resolved)

	// Test non-existent alias
	_, err = sm.ResolveSubqueryAlias("nonexistent")
	assert.Error(t, err)
}

// Test getting all fields in scope
func TestGetAllFieldsInScope(t *testing.T) {
	sm := NewScopeManager()

	// Add field to root scope
	rootField := &FieldSource{
		Name:       "root_field",
		SourceType: SourceTypeTable,
		Scope:      "root",
	}
	sm.AddFieldToScope(rootField)

	// Create child scope and add field
	sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	childField := &FieldSource{
		Name:       "child_field",
		SourceType: SourceTypeExpression,
		Scope:      "child1",
	}
	sm.AddFieldToScope(childField)

	// Get all fields accessible from child scope
	allFields, err := sm.GetAllFieldsInScope("child1")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(allFields))

	// Verify fields are in correct order (child scope fields first)
	assert.Equal(t, "child_field", allFields[0].Name)
	assert.Equal(t, "root_field", allFields[1].Name)
}

// Test field access validation
func TestFieldAccessValidation(t *testing.T) {
	sm := NewScopeManager()

	// Add field to root scope
	rootField := &FieldSource{
		Name:       "accessible_field",
		SourceType: SourceTypeTable,
		Scope:      "root",
	}
	sm.AddFieldToScope(rootField)

	// Create child scope
	sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	// Test valid field access
	err = sm.ValidateFieldAccess("accessible_field")
	assert.NoError(t, err)

	// Test invalid field access
	err = sm.ValidateFieldAccess("inaccessible_field")
	assert.Error(t, err)
}

// Test scope hierarchy visualization
func TestScopeHierarchyVisualization(t *testing.T) {
	sm := NewScopeManager()

	// Add field to root scope
	rootField := &FieldSource{
		Name:       "root_field",
		SourceType: SourceTypeTable,
		Scope:      "root",
	}
	sm.AddFieldToScope(rootField)

	// Add table alias
	tableRef := &TableReference{
		Name:     "u",
		RealName: "users",
		Schema:   "public",
	}
	sm.AddTableAlias("u", tableRef)

	// Create child scope
	sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	childField := &FieldSource{
		Name:       "child_field",
		SourceType: SourceTypeExpression,
		Scope:      "child1",
	}
	sm.AddFieldToScope(childField)

	// Get hierarchy representation
	hierarchy := sm.GetScopeHierarchy()
	assert.NotEqual(t, "", hierarchy)

	// Verify it contains expected elements
	assert.Contains(t, hierarchy, "Scope: root")
	assert.Contains(t, hierarchy, "Scope: child1")
	assert.Contains(t, hierarchy, "root_field")
	assert.Contains(t, hierarchy, "child_field")
	assert.Contains(t, hierarchy, "Table Aliases: u")
}

// Test scope manager reset
func TestScopeManagerReset(t *testing.T) {
	sm := NewScopeManager()

	// Create complex scope hierarchy
	sm.CreateScope("child1")
	err := sm.EnterScope("child1")
	assert.NoError(t, err)

	field := &FieldSource{
		Name:       "test_field",
		SourceType: SourceTypeTable,
		Scope:      "child1",
	}
	sm.AddFieldToScope(field)

	// Reset scope manager
	sm.Reset()

	// Verify reset to initial state
	assert.Equal(t, "root", sm.GetCurrentScope().ID)
	assert.Equal(t, 0, len(sm.GetCurrentScope().AvailableFields))
	assert.Equal(t, 0, len(sm.GetCurrentScope().TableAliases))
	assert.Equal(t, 0, len(sm.GetCurrentScope().SubqueryAliases))
	assert.Equal(t, 0, len(sm.GetCurrentScope().ChildScopes))

	// Verify child scope no longer exists
	_, err = sm.GetScope("child1")
	assert.Error(t, err)
}

// Test error cases
func TestScopeManagerErrorCases(t *testing.T) {
	sm := NewScopeManager()

	// Test entering non-existent scope
	err := sm.EnterScope("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scope nonexistent not found")

	// Test exiting root scope
	err = sm.ExitScope()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exit root scope")

	// Test getting non-existent scope
	_, err = sm.GetScope("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scope nonexistent not found")

	// Test getting all fields in non-existent scope
	_, err = sm.GetAllFieldsInScope("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scope nonexistent not found")
}

// Test complex nested scope scenario
func TestComplexNestedScenario(t *testing.T) {
	sm := NewScopeManager()

	// Setup: Root scope with base tables
	userTable := &TableReference{
		Name:     "u",
		RealName: "users",
		Schema:   "public",
	}
	sm.AddTableAlias("u", userTable)

	userIdField := &FieldSource{
		Name:        "id",
		SourceType:  SourceTypeTable,
		TableSource: userTable,
		Scope:       "root",
	}
	sm.AddFieldToScope(userIdField)

	// Level 1: CTE scope
	sm.CreateScope("cte_scope")
	err := sm.EnterScope("cte_scope")
	assert.NoError(t, err)

	aggField := &FieldSource{
		Name:       "user_count",
		Alias:      "cnt",
		SourceType: SourceTypeAggregate,
		ExprSource: "COUNT(*)",
		Scope:      "cte_scope",
	}
	sm.AddFieldToScope(aggField)

	// Level 2: Subquery scope
	sm.CreateScope("subquery_scope")
	err = sm.EnterScope("subquery_scope")
	assert.NoError(t, err)

	calcField := &FieldSource{
		Name:       "calculated_value",
		SourceType: SourceTypeExpression,
		ExprSource: "cnt * 2",
		Scope:      "subquery_scope",
	}
	sm.AddFieldToScope(calcField)

	// Test field resolution from deepest scope
	// Should be able to access all parent scope fields
	resolved, err := sm.ResolveField("id")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	resolved, err = sm.ResolveField("cnt")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	resolved, err = sm.ResolveField("calculated_value")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))

	// Test table alias resolution
	tableRef, err := sm.ResolveTableAlias("u")
	assert.NoError(t, err)
	assert.Equal(t, "users", tableRef.RealName)

	// Get all accessible fields
	allFields, err := sm.GetAllFieldsInScope("subquery_scope")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(allFields))

	// Verify hierarchy
	hierarchy := sm.GetScopeHierarchy()
	assert.Contains(t, hierarchy, "Scope: root")
	assert.Contains(t, hierarchy, "Scope: cte_scope")
	assert.Contains(t, hierarchy, "Scope: subquery_scope")
}

// Test scope isolation
func TestScopeIsolation(t *testing.T) {
	sm := NewScopeManager()

	// Create two sibling scopes
	sm.CreateScope("sibling1")
	sm.CreateScope("sibling2")

	// Add field to sibling1
	err := sm.EnterScope("sibling1")
	assert.NoError(t, err)

	sibling1Field := &FieldSource{
		Name:       "sibling1_field",
		SourceType: SourceTypeTable,
		Scope:      "sibling1",
	}
	sm.AddFieldToScope(sibling1Field)

	// Switch to sibling2
	err = sm.ExitScope()
	assert.NoError(t, err)
	err = sm.EnterScope("sibling2")
	assert.NoError(t, err)

	// Add field to sibling2
	sibling2Field := &FieldSource{
		Name:       "sibling2_field",
		SourceType: SourceTypeTable,
		Scope:      "sibling2",
	}
	sm.AddFieldToScope(sibling2Field)

	// Test that sibling1's field is not accessible from sibling2
	_, err = sm.ResolveField("sibling1_field")
	assert.Error(t, err)

	// Test that sibling2's field is accessible
	resolved, err := sm.ResolveField("sibling2_field")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
	assert.Equal(t, "sibling2_field", resolved[0].Name)
}
