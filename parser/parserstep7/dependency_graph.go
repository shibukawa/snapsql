package parserstep7

import (
	"errors"
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// DependencyGraph manages dependencies between subqueries
type DependencyGraph struct {
	nodes        map[string]*DependencyNode
	edges        map[string][]string
	scopeManager *ScopeManager
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           string            // Unique ID
	Statement    cmn.StatementNode // Statement reference
	NodeType     DependencyType    // Node type
	Dependencies []string          // Dependent node IDs
	FieldSources []*FieldSource    // Field sources produced by this node
	TableRefs    []*TableReference // Table references used by this node
	Scope        *Scope            // Scope information for this node
}

// DependencyType represents the type of dependency node
type DependencyType int

const (
	DependencyCTE            DependencyType = iota // CTE
	DependencySubquery                             // FROM/SELECT clause subquery
	DependencyMain                                 // Main query
	DependencyFromSubquery                         // FROM clause subquery
	DependencySelectSubquery                       // SELECT clause subquery
)

func (dt DependencyType) String() string {
	switch dt {
	case DependencyCTE:
		return "CTE"
	case DependencySubquery:
		return "Subquery"
	case DependencyMain:
		return "Main"
	case DependencyFromSubquery:
		return "FromSubquery"
	case DependencySelectSubquery:
		return "SelectSubquery"
	default:
		return "Unknown"
	}
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:        make(map[string]*DependencyNode),
		edges:        make(map[string][]string),
		scopeManager: NewScopeManager(),
	}
}

// AddNode adds a node to the dependency graph
func (dg *DependencyGraph) AddNode(node *DependencyNode) error {
	if _, exists := dg.nodes[node.ID]; exists {
		return errors.New("node already exists: " + node.ID)
	}
	dg.nodes[node.ID] = node
	dg.edges[node.ID] = []string{}
	return nil
}

// AddDependency adds a dependency edge from 'from' to 'to'
func (dg *DependencyGraph) AddDependency(from, to string) error {
	if _, exists := dg.nodes[from]; !exists {
		return errors.New("source node not found: " + from)
	}
	if _, exists := dg.nodes[to]; !exists {
		return errors.New("target node not found: " + to)
	}

	dg.edges[from] = append(dg.edges[from], to)
	return nil
}

// GetProcessingOrder returns the processing order using topological sorting
func (dg *DependencyGraph) GetProcessingOrder() ([]string, error) {
	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)

	// Calculate in-degrees
	for nodeID := range dg.nodes {
		inDegree[nodeID] = 0
	}
	for _, edges := range dg.edges {
		for _, to := range edges {
			inDegree[to]++
		}
	}

	// Add nodes with in-degree 0 to queue
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree of neighboring nodes
		for _, neighbor := range dg.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(dg.nodes) {
		return nil, ErrCircularDependency
	}

	return result, nil
}

// HasCircularDependency checks if there are circular dependencies
func (dg *DependencyGraph) HasCircularDependency() bool {
	_, err := dg.GetProcessingOrder()
	return err != nil
}

// GetNode returns a node by ID
func (dg *DependencyGraph) GetNode(id string) *DependencyNode {
	return dg.nodes[id]
}

// GetAllNodes returns all nodes
func (dg *DependencyGraph) GetAllNodes() map[string]*DependencyNode {
	return dg.nodes
}

// GetScopeManager returns the scope manager
func (dg *DependencyGraph) GetScopeManager() *ScopeManager {
	return dg.scopeManager
}

// CreateScopeForNode creates a scope for a node and associates it
func (dg *DependencyGraph) CreateScopeForNode(nodeID string) error {
	node := dg.nodes[nodeID]
	if node == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Create scope for this node
	scope := dg.scopeManager.CreateScope(nodeID)
	node.Scope = scope

	return nil
}

// AddFieldSourceToNode adds a field source to a node's scope
func (dg *DependencyGraph) AddFieldSourceToNode(nodeID string, fieldSource *FieldSource) error {
	node := dg.nodes[nodeID]
	if node == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Add to node's field sources
	node.FieldSources = append(node.FieldSources, fieldSource)

	// Add to scope if scope exists
	if node.Scope != nil {
		// Enter the node's scope temporarily
		currentScope := dg.scopeManager.GetCurrentScope()
		err := dg.scopeManager.EnterScope(nodeID)
		if err != nil {
			return fmt.Errorf("failed to enter scope %s: %w", nodeID, err)
		}

		dg.scopeManager.AddFieldToScope(fieldSource)

		// Return to previous scope
		if currentScope.ID != nodeID {
			err = dg.scopeManager.EnterScope(currentScope.ID)
			if err != nil {
				return fmt.Errorf("failed to return to scope %s: %w", currentScope.ID, err)
			}
		}
	}

	return nil
}

// AddTableReferenceToNode adds a table reference to a node
func (dg *DependencyGraph) AddTableReferenceToNode(nodeID string, tableRef *TableReference) error {
	node := dg.nodes[nodeID]
	if node == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Add to node's table references
	node.TableRefs = append(node.TableRefs, tableRef)

	// Add table alias to scope if scope exists and table has an alias
	if node.Scope != nil && tableRef.Name != "" {
		// Enter the node's scope temporarily
		currentScope := dg.scopeManager.GetCurrentScope()
		err := dg.scopeManager.EnterScope(nodeID)
		if err != nil {
			return fmt.Errorf("failed to enter scope %s: %w", nodeID, err)
		}

		dg.scopeManager.AddTableAlias(tableRef.Name, tableRef)

		// Return to previous scope
		if currentScope.ID != nodeID {
			err = dg.scopeManager.EnterScope(currentScope.ID)
			if err != nil {
				return fmt.Errorf("failed to return to scope %s: %w", currentScope.ID, err)
			}
		}
	}

	return nil
}

// ResolveFieldInNode resolves a field reference from a specific node's perspective
func (dg *DependencyGraph) ResolveFieldInNode(nodeID, fieldName string) ([]*FieldSource, error) {
	node := dg.nodes[nodeID]
	if node == nil {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	if node.Scope == nil {
		return nil, fmt.Errorf("node %s has no associated scope", nodeID)
	}

	// Enter the node's scope temporarily
	currentScope := dg.scopeManager.GetCurrentScope()
	err := dg.scopeManager.EnterScope(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to enter scope %s: %w", nodeID, err)
	}

	// Resolve field
	resolved, err := dg.scopeManager.ResolveField(fieldName)

	// Return to previous scope
	if currentScope.ID != nodeID {
		returnErr := dg.scopeManager.EnterScope(currentScope.ID)
		if returnErr != nil {
			return nil, fmt.Errorf("failed to return to scope %s: %w", currentScope.ID, returnErr)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("field '%s' not accessible from node %s: %w", fieldName, nodeID, err)
	}

	return resolved, nil
}

// ValidateFieldAccessForNode validates field access from a specific node
func (dg *DependencyGraph) ValidateFieldAccessForNode(nodeID, fieldName string) error {
	_, err := dg.ResolveFieldInNode(nodeID, fieldName)
	return err
}

// GetAccessibleFieldsForNode returns all fields accessible from a specific node
func (dg *DependencyGraph) GetAccessibleFieldsForNode(nodeID string) ([]*FieldSource, error) {
	node := dg.nodes[nodeID]
	if node == nil {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	if node.Scope == nil {
		return nil, fmt.Errorf("node %s has no associated scope", nodeID)
	}

	return dg.scopeManager.GetAllFieldsInScope(nodeID)
}

// BuildScopeHierarchy builds the scope hierarchy based on dependency relationships
func (dg *DependencyGraph) BuildScopeHierarchy() error {
	// Reset scope manager
	dg.scopeManager.Reset()

	// Create scopes for all nodes
	for nodeID := range dg.nodes {
		err := dg.CreateScopeForNode(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create scope for node %s: %w", nodeID, err)
		}
	}

	// Build parent-child relationships based on dependencies
	// Parent nodes should have child scopes for their dependent nodes
	for _, node := range dg.nodes {
		if len(node.Dependencies) > 0 {
			// This node depends on others, so it should be a child scope
			// Find a suitable parent (typically the first dependency for simplicity)
			parentNodeID := node.Dependencies[0]
			parentNode := dg.nodes[parentNodeID]

			if parentNode != nil && parentNode.Scope != nil && node.Scope != nil {
				// Set parent-child relationship
				node.Scope.ParentScope = parentNode.Scope
				parentNode.Scope.ChildScopes = append(parentNode.Scope.ChildScopes, node.Scope)
			}
		}
	}

	return nil
}

// GetScopeHierarchyVisualization returns a visualization of the scope hierarchy
func (dg *DependencyGraph) GetScopeHierarchyVisualization() string {
	return dg.scopeManager.GetScopeHierarchy()
}
