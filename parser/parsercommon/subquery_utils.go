package parsercommon

import (
	"errors"
	"fmt"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
)

var ErrInvalidTableReferenceType = errors.New("invalid table reference type")

// SubqueryGraphUtils provides utility functions for extracting information from SQDependencyGraph
// without exposing internal implementation details

// GetDependencyNodes returns all nodes in the dependency graph
func GetDependencyNodes(dg *SQDependencyGraph) map[string]*SQDependencyNode {
	if dg == nil {
		return make(map[string]*SQDependencyNode)
	}
	return dg.GetAllNodes()
}

// GetNodeByID returns a specific node by ID
func GetNodeByID(dg *SQDependencyGraph, nodeID string) (*SQDependencyNode, bool) {
	if dg == nil {
		return nil, false
	}
	nodes := dg.GetAllNodes()
	node, exists := nodes[nodeID]
	return node, exists
}

// AddNodeToDependencyGraph adds a new node to the dependency graph
func AddNodeToDependencyGraph(dg *SQDependencyGraph, node *SQDependencyNode) {
	if dg == nil || node == nil {
		return
	}
	dg.AddNode(node)
}

// AddFieldSourceToGraphNode adds a field source to a specific node in the dependency graph
func AddFieldSourceToGraphNode(dg *SQDependencyGraph, nodeID string, fieldSource *SQFieldSource) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}
	return dg.AddFieldSourceToNode(nodeID, fieldSource)
}

// GetAccessibleFieldsFromGraph returns all field sources available to a specific node
func GetAccessibleFieldsFromGraph(dg *SQDependencyGraph, nodeID string) ([]*SQFieldSource, error) {
	if dg == nil {
		return nil, ErrNoDependencyGraph
	}
	return dg.GetAccessibleFieldsForNode(nodeID)
}

// ValidateFieldAccessInGraph validates if a field can be accessed from a specific node
func ValidateFieldAccessInGraph(dg *SQDependencyGraph, nodeID, fieldName string) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}
	return dg.ValidateFieldAccessForNode(nodeID, fieldName)
}

// ResolveFieldInGraph resolves a field reference from a specific node's perspective
func ResolveFieldInGraph(dg *SQDependencyGraph, nodeID, fieldName string) ([]*SQFieldSource, error) {
	if dg == nil {
		return nil, ErrNoDependencyGraph
	}
	return dg.ResolveFieldInNode(nodeID, fieldName)
}

// GetScopeHierarchyFromGraph returns a visualization of the scope hierarchy
func GetScopeHierarchyFromGraph(dg *SQDependencyGraph) string {
	if dg == nil {
		return "No dependency graph available"
	}
	return dg.GetScopeHierarchyVisualization()
}

// HasCircularDependencyInGraph checks if the dependency graph has circular dependencies
func HasCircularDependencyInGraph(dg *SQDependencyGraph) bool {
	if dg == nil {
		return false
	}

	nodes := dg.GetAllNodes()
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// Check each node for circular dependencies using DFS
	for nodeID := range nodes {
		if !visited[nodeID] {
			if hasCircularDependencyDFS(nodes, nodeID, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCircularDependencyDFS performs depth-first search to detect circular dependencies
func hasCircularDependencyDFS(nodes map[string]*SQDependencyNode, nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	node, exists := nodes[nodeID]
	if !exists {
		return false
	}

	// Check all dependencies
	for _, depID := range node.Dependencies {
		if !visited[depID] {
			if hasCircularDependencyDFS(nodes, depID, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true // Circular dependency detected
		}
	}

	recStack[nodeID] = false
	return false
}

// GetProcessingOrderFromGraph returns the processing order for nodes in the dependency graph
func GetProcessingOrderFromGraph(dg *SQDependencyGraph) ([]string, error) {
	if dg == nil {
		return nil, ErrNoDependencyGraph
	}

	nodes := dg.GetAllNodes()
	if len(nodes) == 0 {
		return []string{}, nil
	}

	// Check for circular dependencies first
	if HasCircularDependencyInGraph(dg) {
		return nil, ErrCircularDependency
	}

	// Perform topological sort
	visited := make(map[string]bool)
	var result []string

	// Visit all nodes
	for nodeID := range nodes {
		if !visited[nodeID] {
			if err := topologicalSortDFS(nodes, nodeID, visited, &result); err != nil {
				return nil, err
			}
		}
	}

	// Reverse the result to get correct order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// topologicalSortDFS performs depth-first search for topological sorting
func topologicalSortDFS(nodes map[string]*SQDependencyNode, nodeID string, visited map[string]bool, result *[]string) error {
	visited[nodeID] = true

	node, exists := nodes[nodeID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	// Visit all dependencies first
	for _, depID := range node.Dependencies {
		if !visited[depID] {
			if err := topologicalSortDFS(nodes, depID, visited, result); err != nil {
				return err
			}
		}
	}

	// Add current node to result
	*result = append(*result, nodeID)
	return nil
}

// BuildScopeHierarchyInGraph builds scope hierarchy for the dependency graph
func BuildScopeHierarchyInGraph(dg *SQDependencyGraph) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}

	nodes := dg.GetAllNodes()
	if len(nodes) == 0 {
		return nil
	}

	// For now, this is a simple implementation
	// In a full implementation, this would build proper scope relationships
	for _, node := range nodes {
		if node.Scope == nil {
			// Create a basic scope for the node
			node.Scope = &SQScope{
				ID:              node.ID,
				AvailableFields: node.FieldSources,
				TableAliases:    make(map[string]*SQTableReference),
				SubqueryAliases: make(map[string]string),
			}
		}
	}

	return nil
}

// CreateDependencyVisualization returns a basic text visualization of the dependency graph
func CreateDependencyVisualization(dg *SQDependencyGraph) string {
	if dg == nil {
		return "No dependency graph available"
	}

	nodes := dg.GetAllNodes()
	if len(nodes) == 0 {
		return "No dependencies found"
	}

	var result strings.Builder
	result.WriteString("Dependency Graph:\n")

	for _, node := range nodes {
		result.WriteString(fmt.Sprintf("- %s (%s)\n", node.ID, node.NodeType.String()))
		if len(node.Dependencies) > 0 {
			result.WriteString(fmt.Sprintf("  Dependencies: %s\n", strings.Join(node.Dependencies, ", ")))
		}
		if len(node.FieldSources) > 0 {
			result.WriteString(fmt.Sprintf("  Fields: %d\n", len(node.FieldSources)))
		}
	}

	return result.String()
}

// AddDependencyToGraph adds a dependency relationship between two nodes
func AddDependencyToGraph(dg *SQDependencyGraph, fromNodeID, toNodeID string) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}

	nodes := dg.GetAllNodes()
	fromNode, exists := nodes[fromNodeID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, fromNodeID)
	}

	// Check if toNode exists
	if _, exists := nodes[toNodeID]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, toNodeID)
	}

	// Add dependency if not already present
	for _, dep := range fromNode.Dependencies {
		if dep == toNodeID {
			return nil // Already exists
		}
	}

	fromNode.Dependencies = append(fromNode.Dependencies, toNodeID)
	return nil
}

// AddTableReferenceToGraphNode adds a table reference to a specific node in the dependency graph
func AddTableReferenceToGraphNode(dg *SQDependencyGraph, nodeID string, tableRef *SQTableReference) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}

	nodes := dg.GetAllNodes()
	node, exists := nodes[nodeID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	node.TableRefs = append(node.TableRefs, tableRef)
	return nil
}

// GetScopeHierarchyVisualizationFromGraph returns scope hierarchy visualization
func GetScopeHierarchyVisualizationFromGraph(dg *SQDependencyGraph) string {
	if dg == nil {
		return "No scope hierarchy available"
	}
	// Simple visualization implementation
	nodes := dg.GetAllNodes()
	if len(nodes) == 0 {
		return "No nodes in dependency graph"
	}

	result := "Scope Hierarchy:\n"
	for nodeID, node := range nodes {
		result += fmt.Sprintf("- %s (%s)\n", nodeID, node.NodeType.String())
		if len(node.Dependencies) > 0 {
			result += "  Dependencies: " + fmt.Sprintf("%v", node.Dependencies) + "\n"
		}
	}
	return result
}

// AddFieldSourceToNodeInGraph adds a field source to a specific node (legacy compatibility)
func AddFieldSourceToNodeInGraph(dg *SQDependencyGraph, nodeID string, fieldSource interface{}) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}
	// Convert interface{} to proper type if needed
	if fs, ok := fieldSource.(*SQFieldSource); ok {
		return AddFieldSourceToGraphNode(dg, nodeID, fs)
	}
	return snapsql.ErrInvalidFieldSourceType
}

// AddTableReferenceToNodeInGraph adds a table reference to a specific node
func AddTableReferenceToNodeInGraph(dg *SQDependencyGraph, nodeID string, tableRef interface{}) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}
	// Delegate to the specific implementation with type assertion check
	if sqTableRef, ok := tableRef.(*SQTableReference); ok {
		return AddTableReferenceToGraphNode(dg, nodeID, sqTableRef)
	}
	return fmt.Errorf("%w: %T", ErrInvalidTableReferenceType, tableRef)
}

// GetAccessibleFieldsForNodeInGraph returns accessible fields for a node (legacy compatibility)
func GetAccessibleFieldsForNodeInGraph(dg *SQDependencyGraph, nodeID string) (interface{}, error) {
	if dg == nil {
		return nil, ErrNoDependencyGraph
	}
	return GetAccessibleFieldsFromGraph(dg, nodeID)
}

// ValidateFieldAccessForNodeInGraph validates field access
func ValidateFieldAccessForNodeInGraph(dg *SQDependencyGraph, nodeID, fieldName string) error {
	if dg == nil {
		return ErrNoDependencyGraph
	}
	// Simple implementation for now
	_, err := GetAccessibleFieldsFromGraph(dg, nodeID)
	return err
}

// ResolveFieldInNodeFromGraph resolves field references
func ResolveFieldInNodeFromGraph(dg *SQDependencyGraph, nodeID, fieldName string) (interface{}, error) {
	if dg == nil {
		return nil, ErrNoDependencyGraph
	}
	// Simple implementation for now
	return GetAccessibleFieldsFromGraph(dg, nodeID)
}

// GetDependencyEdges returns the edges map for dependency analysis
func GetDependencyEdges(dg *SQDependencyGraph) map[string][]string {
	if dg == nil {
		return make(map[string][]string)
	}
	// Note: This accesses a private field, which should be made public or
	// the SQDependencyGraph should provide a public method for this
	return make(map[string][]string) // Return empty for now
}
