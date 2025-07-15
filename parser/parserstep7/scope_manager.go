package parserstep7

import (
	"fmt"
	"strings"
)

// Scope represents a variable scope in SQL execution
type Scope struct {
	ID              string                     // Unique scope ID
	ParentScope     *Scope                     // Parent scope (for nested queries)
	ChildScopes     []*Scope                   // Child scopes
	AvailableFields []*FieldSource             // Fields available in this scope
	TableAliases    map[string]*TableReference // Table aliases in this scope
	SubqueryAliases map[string]string          // Subquery aliases to their IDs
}

// ScopeManager manages scope hierarchy and field visibility
type ScopeManager struct {
	scopes       map[string]*Scope // All scopes by ID
	currentScope *Scope            // Current active scope
	rootScope    *Scope            // Root scope
}

// NewScopeManager creates a new scope manager
func NewScopeManager() *ScopeManager {
	rootScope := &Scope{
		ID:              "root",
		ParentScope:     nil,
		ChildScopes:     make([]*Scope, 0),
		AvailableFields: make([]*FieldSource, 0),
		TableAliases:    make(map[string]*TableReference),
		SubqueryAliases: make(map[string]string),
	}

	return &ScopeManager{
		scopes:       map[string]*Scope{"root": rootScope},
		currentScope: rootScope,
		rootScope:    rootScope,
	}
}

// CreateScope creates a new scope as a child of the current scope
func (sm *ScopeManager) CreateScope(id string) *Scope {
	newScope := &Scope{
		ID:              id,
		ParentScope:     sm.currentScope,
		ChildScopes:     make([]*Scope, 0),
		AvailableFields: make([]*FieldSource, 0),
		TableAliases:    make(map[string]*TableReference),
		SubqueryAliases: make(map[string]string),
	}

	sm.currentScope.ChildScopes = append(sm.currentScope.ChildScopes, newScope)
	sm.scopes[id] = newScope
	return newScope
}

// EnterScope enters the specified scope
func (sm *ScopeManager) EnterScope(scopeID string) error {
	scope, exists := sm.scopes[scopeID]
	if !exists {
		return fmt.Errorf("scope %s not found", scopeID)
	}
	sm.currentScope = scope
	return nil
}

// ExitScope exits to the parent scope
func (sm *ScopeManager) ExitScope() error {
	if sm.currentScope.ParentScope == nil {
		return fmt.Errorf("cannot exit root scope")
	}
	sm.currentScope = sm.currentScope.ParentScope
	return nil
}

// AddFieldToScope adds a field source to the current scope
func (sm *ScopeManager) AddFieldToScope(field *FieldSource) {
	sm.currentScope.AvailableFields = append(sm.currentScope.AvailableFields, field)
}

// AddTableAlias adds a table alias to the current scope
func (sm *ScopeManager) AddTableAlias(alias string, tableRef *TableReference) {
	sm.currentScope.TableAliases[alias] = tableRef
}

// AddSubqueryAlias adds a subquery alias to the current scope
func (sm *ScopeManager) AddSubqueryAlias(alias, subqueryID string) {
	sm.currentScope.SubqueryAliases[alias] = subqueryID
}

// ResolveField resolves a field reference in the current scope hierarchy
func (sm *ScopeManager) ResolveField(fieldName string) ([]*FieldSource, error) {
	var resolved []*FieldSource

	// Search from current scope up to root
	scope := sm.currentScope
	for scope != nil {
		for _, field := range scope.AvailableFields {
			if field.Name == fieldName ||
				(field.Alias != "" && field.Alias == fieldName) {
				resolved = append(resolved, field)
			}
		}
		scope = scope.ParentScope
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("field '%s' not found in any accessible scope", fieldName)
	}

	return resolved, nil
}

// ResolveTableAlias resolves a table alias in the current scope hierarchy
func (sm *ScopeManager) ResolveTableAlias(alias string) (*TableReference, error) {
	// Search from current scope up to root
	scope := sm.currentScope
	for scope != nil {
		if tableRef, exists := scope.TableAliases[alias]; exists {
			return tableRef, nil
		}
		scope = scope.ParentScope
	}

	return nil, fmt.Errorf("table alias '%s' not found in any accessible scope", alias)
}

// ResolveSubqueryAlias resolves a subquery alias in the current scope hierarchy
func (sm *ScopeManager) ResolveSubqueryAlias(alias string) (string, error) {
	// Search from current scope up to root
	scope := sm.currentScope
	for scope != nil {
		if subqueryID, exists := scope.SubqueryAliases[alias]; exists {
			return subqueryID, nil
		}
		scope = scope.ParentScope
	}

	return "", fmt.Errorf("subquery alias '%s' not found in any accessible scope", alias)
}

// GetCurrentScope returns the current active scope
func (sm *ScopeManager) GetCurrentScope() *Scope {
	return sm.currentScope
}

// GetScope returns a scope by ID
func (sm *ScopeManager) GetScope(id string) (*Scope, error) {
	scope, exists := sm.scopes[id]
	if !exists {
		return nil, fmt.Errorf("scope %s not found", id)
	}
	return scope, nil
}

// GetAllFieldsInScope returns all fields accessible from the specified scope
func (sm *ScopeManager) GetAllFieldsInScope(scopeID string) ([]*FieldSource, error) {
	scope, err := sm.GetScope(scopeID)
	if err != nil {
		return nil, err
	}

	var allFields []*FieldSource

	// Collect fields from current scope and all parent scopes
	current := scope
	for current != nil {
		allFields = append(allFields, current.AvailableFields...)
		current = current.ParentScope
	}

	return allFields, nil
}

// ValidateFieldAccess validates if a field can be accessed from the current scope
func (sm *ScopeManager) ValidateFieldAccess(fieldName string) error {
	_, err := sm.ResolveField(fieldName)
	return err
}

// GetScopeHierarchy returns a string representation of the scope hierarchy
func (sm *ScopeManager) GetScopeHierarchy() string {
	var builder strings.Builder
	sm.buildScopeTree(sm.rootScope, 0, &builder)
	return builder.String()
}

// buildScopeTree recursively builds the scope tree representation
func (sm *ScopeManager) buildScopeTree(scope *Scope, depth int, builder *strings.Builder) {
	indent := strings.Repeat("  ", depth)
	builder.WriteString(fmt.Sprintf("%s- Scope: %s\n", indent, scope.ID))

	if len(scope.AvailableFields) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Fields: ", indent))
		fieldNames := make([]string, len(scope.AvailableFields))
		for i, field := range scope.AvailableFields {
			fieldNames[i] = field.Name
		}
		builder.WriteString(strings.Join(fieldNames, ", "))
		builder.WriteString("\n")
	}

	if len(scope.TableAliases) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Table Aliases: ", indent))
		var aliases []string
		for alias := range scope.TableAliases {
			aliases = append(aliases, alias)
		}
		builder.WriteString(strings.Join(aliases, ", "))
		builder.WriteString("\n")
	}

	for _, child := range scope.ChildScopes {
		sm.buildScopeTree(child, depth+1, builder)
	}
}

// Reset resets the scope manager to initial state
func (sm *ScopeManager) Reset() {
	sm.rootScope = &Scope{
		ID:              "root",
		ParentScope:     nil,
		ChildScopes:     make([]*Scope, 0),
		AvailableFields: make([]*FieldSource, 0),
		TableAliases:    make(map[string]*TableReference),
		SubqueryAliases: make(map[string]string),
	}
	sm.scopes = map[string]*Scope{"root": sm.rootScope}
	sm.currentScope = sm.rootScope
}
