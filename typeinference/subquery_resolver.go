package typeinference2

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parserstep7"
)

// SubqueryTypeResolver handles type inference for subqueries using parserstep7 results
type SubqueryTypeResolver struct {
	schemaResolver *SchemaResolver
	parseResult    *parserstep7.ParseResult
	dialect        snapsql.Dialect
	typeCache      map[string][]*InferredFieldInfo // Cache subquery field types by dependency node ID
}

// NewSubqueryTypeResolver creates a new subquery type resolver
func NewSubqueryTypeResolver(
	schemaResolver *SchemaResolver,
	parseResult *parserstep7.ParseResult,
	dialect snapsql.Dialect,
) *SubqueryTypeResolver {
	return &SubqueryTypeResolver{
		schemaResolver: schemaResolver,
		parseResult:    parseResult,
		dialect:        dialect,
		typeCache:      make(map[string][]*InferredFieldInfo),
	}
}

// ResolveSubqueryTypes resolves types for all subqueries in dependency order
func (r *SubqueryTypeResolver) ResolveSubqueryTypes() error {
	if r.parseResult == nil {
		return fmt.Errorf("no parse result available")
	}

	// Get processing order from dependency graph
	processingOrder := r.parseResult.ProcessingOrder
	if len(processingOrder) == 0 {
		// No subqueries to process
		return nil
	}

	// Process each subquery in dependency order
	for _, nodeID := range processingOrder {
		if err := r.resolveSubqueryNodeTypes(nodeID); err != nil {
			return fmt.Errorf("failed to resolve types for subquery node %s: %w", nodeID, err)
		}
	}

	return nil
}

// resolveSubqueryNodeTypes resolves types for a specific subquery node
func (r *SubqueryTypeResolver) resolveSubqueryNodeTypes(nodeID string) error {
	// Get dependency node information
	dependencyGraph := r.parseResult.DependencyGraph
	if dependencyGraph == nil {
		return fmt.Errorf("dependency graph not available")
	}

	node := dependencyGraph.GetNode(nodeID)
	if node == nil {
		return fmt.Errorf("dependency node %s not found", nodeID)
	}

	// Skip main query - it will be handled by the main type inference engine
	if node.NodeType == parserstep7.DependencyMain {
		return nil
	}

	// Create a sub-engine for this subquery
	subEngine := r.createSubEngine(node)

	// Infer types for this subquery
	fieldInfos, err := subEngine.InferSelectTypes()
	if err != nil {
		return fmt.Errorf("type inference failed for subquery %s: %w", nodeID, err)
	}

	// Cache the results
	r.typeCache[nodeID] = fieldInfos

	return nil
}

// createSubEngine creates a type inference sub-engine for a specific subquery node
func (r *SubqueryTypeResolver) createSubEngine(node *parserstep7.DependencyNode) *TypeInferenceEngine2 {
	// Create context for subquery with available tables from its scope
	context := &InferenceContext{
		Dialect:       r.dialect,
		TableAliases:  make(map[string]string),
		CurrentTables: r.extractAvailableTablesFromScope(node),
		SubqueryDepth: r.calculateSubqueryDepth(node),
	}

	// Create sub-engine with limited scope
	subEngine := &TypeInferenceEngine2{
		databaseSchemas: r.schemaResolver.schemas,
		schemaResolver:  r.schemaResolver,
		statementNode:   node.Statement,
		subqueryInfo:    nil, // No nested subquery analysis for sub-engines
		context:         context,
		fieldNameGen:    NewFieldNameGenerator(),
		enhancedGen:     NewEnhancedFieldNameGenerator(),
		typeCache:       make(map[string][]*InferredFieldInfo),
	}

	return subEngine
}

// extractAvailableTablesFromScope extracts available table names from node scope
func (r *SubqueryTypeResolver) extractAvailableTablesFromScope(node *parserstep7.DependencyNode) []string {
	var tables []string

	// Add tables from field sources (table references in this node)
	for _, tableRef := range node.TableRefs {
		if tableRef.Name != "" {
			tables = append(tables, tableRef.Name)
		}
	}

	// Add tables from dependent subqueries (available in scope)
	for _, depID := range node.Dependencies {
		if depNode := r.parseResult.DependencyGraph.GetNode(depID); depNode != nil {
			// For CTE, add the CTE name as available table
			if depNode.NodeType == parserstep7.DependencyCTE {
				// Extract CTE name from node ID (format: "cte_<name>")
				if strings.HasPrefix(depNode.ID, "cte_") {
					cteName := strings.TrimPrefix(depNode.ID, "cte_")
					tables = append(tables, cteName)
				}
			}
		}
	}

	return tables
}

// calculateSubqueryDepth calculates the nesting depth of a subquery
func (r *SubqueryTypeResolver) calculateSubqueryDepth(node *parserstep7.DependencyNode) int {
	// Simple depth calculation based on dependency chain length
	depth := 0
	visited := make(map[string]bool)

	return r.calculateDepthRecursive(node.ID, visited, depth)
}

// calculateDepthRecursive recursively calculates subquery depth
func (r *SubqueryTypeResolver) calculateDepthRecursive(nodeID string, visited map[string]bool, currentDepth int) int {
	if visited[nodeID] {
		return currentDepth // Avoid circular dependencies
	}

	visited[nodeID] = true
	maxDepth := currentDepth

	// Check dependencies for deeper nesting
	if node := r.parseResult.DependencyGraph.GetNode(nodeID); node != nil {
		for _, depID := range node.Dependencies {
			depDepth := r.calculateDepthRecursive(depID, visited, currentDepth+1)
			if depDepth > maxDepth {
				maxDepth = depDepth
			}
		}
	}

	return maxDepth
}

// GetSubqueryFieldTypes returns cached field types for a subquery node
func (r *SubqueryTypeResolver) GetSubqueryFieldTypes(nodeID string) ([]*InferredFieldInfo, bool) {
	fieldInfos, exists := r.typeCache[nodeID]
	return fieldInfos, exists
}

// GetAvailableSubqueryTables returns table names available from resolved subqueries
func (r *SubqueryTypeResolver) GetAvailableSubqueryTables() []string {
	var tables []string

	if r.parseResult == nil {
		return tables
	}

	dependencyGraph := r.parseResult.DependencyGraph
	if dependencyGraph == nil {
		return tables
	}

	// Add CTE names as available tables
	for nodeID := range r.typeCache {
		if node := dependencyGraph.GetNode(nodeID); node != nil {
			if node.NodeType == parserstep7.DependencyCTE {
				// Extract CTE name from node ID
				if strings.HasPrefix(nodeID, "cte_") {
					cteName := strings.TrimPrefix(nodeID, "cte_")
					tables = append(tables, cteName)
				}
			}
		}
	}

	return tables
}

// ValidateSubqueryReferences validates that subquery references are properly resolved
func (r *SubqueryTypeResolver) ValidateSubqueryReferences() []ValidationError {
	var errors []ValidationError

	if r.parseResult == nil {
		return errors
	}

	dependencyGraph := r.parseResult.DependencyGraph
	if dependencyGraph == nil {
		return errors
	}

	// Check that all referenced subqueries have type information
	for nodeID, node := range dependencyGraph.GetAllNodes() {
		if node.NodeType == parserstep7.DependencyMain {
			continue // Skip main query
		}

		if _, exists := r.typeCache[nodeID]; !exists {
			errors = append(errors, ValidationError{
				FieldIndex: -1,
				ErrorType:  ValidationErrorType(9), // SubqueryNotResolved (custom type)
				Message:    fmt.Sprintf("Subquery node %s has no type information", nodeID),
				Severity:   Error,
				TableName:  "",
				ColumnName: "",
				Suggestion: fmt.Sprintf("Ensure subquery %s is properly analyzed", nodeID),
			})
		}
	}

	return errors
}

// ResolveSubqueryReference resolves a table reference that might be a subquery
func (r *SubqueryTypeResolver) ResolveSubqueryReference(tableName string) ([]*InferredFieldInfo, bool) {
	// Check if this table name corresponds to a CTE
	for nodeID, fieldInfos := range r.typeCache {
		if strings.HasPrefix(nodeID, "cte_") {
			cteName := strings.TrimPrefix(nodeID, "cte_")
			if cteName == tableName {
				return fieldInfos, true
			}
		}
	}

	return nil, false
}
