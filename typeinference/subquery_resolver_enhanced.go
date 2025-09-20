package typeinference

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// EnhancedSubqueryResolver provides complete subquery type inference
// using StatementNode information and dependency analysis
type EnhancedSubqueryResolver struct {
	schemaResolver     *SchemaResolver                          // Schema resolver for type lookup
	statementNode      parser.StatementNode                     // Parsed SQL AST
	dialect            snapsql.Dialect                          // Database dialect
	typeCache          map[string][]*InferredFieldInfo          // Cache subquery field types by dependency node ID
	fieldResolverCache map[string]map[string]*InferredFieldInfo // nodeID -> fieldName -> type info
}

// NewEnhancedSubqueryResolver creates an enhanced subquery resolver
func NewEnhancedSubqueryResolver(
	schemaResolver *SchemaResolver,
	statementNode parser.StatementNode,
	dialect snapsql.Dialect,
) *EnhancedSubqueryResolver {
	return &EnhancedSubqueryResolver{
		schemaResolver:     schemaResolver,
		statementNode:      statementNode,
		dialect:            dialect,
		typeCache:          make(map[string][]*InferredFieldInfo),
		fieldResolverCache: make(map[string]map[string]*InferredFieldInfo),
	}
}

// ResolveSubqueryTypesComplete performs complete subquery type resolution
// including field-level type inference and table reference resolution
func (esr *EnhancedSubqueryResolver) ResolveSubqueryTypesComplete() error {
	if esr.statementNode == nil {
		return nil
	}

	// Get dependency graph from StatementNode
	depGraph := esr.statementNode.GetSubqueryDependencies()
	if depGraph == nil {
		return nil
	}

	// Get processing order from StatementNode
	processingOrder := esr.statementNode.GetProcessingOrder()
	if len(processingOrder) == 0 {
		return nil
	}

	// Process each subquery in dependency order
	for _, nodeID := range processingOrder {
		err := esr.resolveSubqueryNodeComplete(nodeID, depGraph)
		if err != nil {
			return fmt.Errorf("failed to resolve subquery node %s: %w", nodeID, err)
		}
	}

	return nil
}

// resolveSubqueryNodeComplete performs complete type resolution for a single subquery node
func (esr *EnhancedSubqueryResolver) resolveSubqueryNodeComplete(nodeID string, depGraph *parser.SQDependencyGraph) error {
	node := depGraph.GetNode(nodeID)
	if node == nil {
		return fmt.Errorf("%w: %s", snapsql.ErrDependencyNodeNotFound, nodeID)
	}

	// Skip main query - it's handled by the main inference engine
	if node.NodeType == parser.DependencyMain {
		return nil
	}

	// Get statement from dependency node
	if node.Statement == nil {
		return fmt.Errorf("%w: %s", snapsql.ErrNoStatementFoundForSubquery, nodeID)
	}

	// Create context with proper table resolution
	context := esr.createEnhancedContext(node, depGraph)

	// Create specialized sub-engine for this subquery
	subEngine := esr.createEnhancedSubEngine(node, context)

	// Perform type inference based on statement type
	var (
		fieldInfos []*InferredFieldInfo
		err        error
	)

	switch stmt := node.Statement.(type) {
	case *parser.SelectStatement:
		fieldInfos, err = esr.inferSelectSubquery(stmt, subEngine, node)
	default:
		// For CTE and other complex statements, try to extract the main SELECT
		if selectStmt, ok := esr.extractSelectFromStatement(node.Statement); ok {
			fieldInfos, err = esr.inferSelectSubquery(selectStmt, subEngine, node)
		} else {
			return fmt.Errorf("%w: %T", snapsql.ErrUnsupportedSubqueryStatementType, stmt)
		}
	}

	if err != nil {
		return fmt.Errorf("type inference failed for subquery %s: %w", nodeID, err)
	}

	// Cache the results
	esr.typeCache[nodeID] = fieldInfos
	esr.cacheFieldMapping(nodeID, fieldInfos)

	return nil
}

// createEnhancedContext creates inference context with complete table resolution
func (esr *EnhancedSubqueryResolver) createEnhancedContext(node *parser.SQDependencyNode, depGraph *parser.SQDependencyGraph) *InferenceContext {
	context := &InferenceContext{
		Dialect:       esr.dialect,
		TableAliases:  make(map[string]string),
		CurrentTables: []string{},
		SubqueryDepth: esr.calculateEnhancedDepth(node, depGraph),
	}

	// Add tables from node's table references
	for _, tableRef := range node.TableRefs {
		if tableRef.Name != "" {
			context.CurrentTables = append(context.CurrentTables, tableRef.Name)
			// Add alias mapping if available (using RealName as the actual table)
			if tableRef.RealName != "" && tableRef.RealName != tableRef.Name {
				context.TableAliases[tableRef.Name] = tableRef.RealName
			}
		}
	}

	// Add CTE tables from dependencies
	for _, depID := range node.Dependencies {
		if depNode := depGraph.GetNode(depID); depNode != nil {
			if depNode.NodeType == parser.DependencyCTE {
				cteName := esr.extractCTEName(depNode)
				if cteName != "" {
					context.CurrentTables = append(context.CurrentTables, cteName)
				}
			}
		}
	}

	return context
}

// createEnhancedSubEngine creates a specialized sub-engine for subquery inference
func (esr *EnhancedSubqueryResolver) createEnhancedSubEngine(node *parser.SQDependencyNode, context *InferenceContext) *TypeInferenceEngine2 {
	subEngine := &TypeInferenceEngine2{
		databaseSchemas: esr.schemaResolver.schemas,
		schemaResolver:  esr.schemaResolver,
		statementNode:   node.Statement,
		context:         context,
		fieldNameGen:    NewFieldNameGenerator(),
		enhancedGen:     NewEnhancedFieldNameGenerator(),
		typeCache:       make(map[string][]*InferredFieldInfo),
	}

	// Don't create recursive subquery resolver for sub-engine to avoid infinite recursion
	subEngine.enhancedResolver = nil

	return subEngine
}

// inferSelectSubquery performs type inference for SELECT subqueries
func (esr *EnhancedSubqueryResolver) inferSelectSubquery(
	stmt *parser.SelectStatement,
	subEngine *TypeInferenceEngine2,
	node *parser.SQDependencyNode,
) ([]*InferredFieldInfo, error) {
	// Extract table aliases from FROM clause
	subEngine.extractTableAliases(stmt)

	// Add available subquery tables from dependencies
	esr.addDependentSubqueryTables(subEngine, node)

	// Perform SELECT type inference
	return subEngine.inferSelectStatement(stmt)
}

// addDependentSubqueryTables adds tables from dependent subqueries to the engine context
func (esr *EnhancedSubqueryResolver) addDependentSubqueryTables(subEngine *TypeInferenceEngine2, node *parser.SQDependencyNode) {
	for _, depID := range node.Dependencies {
		if _, exists := esr.typeCache[depID]; exists {
			// Add this dependency as an available "table"
			cteName := esr.extractCTENameFromNodeID(depID)
			if cteName != "" {
				subEngine.context.CurrentTables = append(subEngine.context.CurrentTables, cteName)
			}
		}
	}
}

// calculateEnhancedDepth calculates subquery nesting depth with dependency analysis
func (esr *EnhancedSubqueryResolver) calculateEnhancedDepth(node *parser.SQDependencyNode, depGraph *parser.SQDependencyGraph) int {
	visited := make(map[string]bool)
	return esr.calculateDepthWithDependencies(node.ID, depGraph, visited, 0)
}

// calculateDepthWithDependencies recursively calculates depth with proper dependency tracking
func (esr *EnhancedSubqueryResolver) calculateDepthWithDependencies(
	nodeID string,
	depGraph *parser.SQDependencyGraph,
	visited map[string]bool,
	currentDepth int,
) int {
	if visited[nodeID] {
		return currentDepth
	}

	visited[nodeID] = true

	node := depGraph.GetNode(nodeID)
	if node == nil {
		return currentDepth
	}

	maxDepth := currentDepth

	for _, depID := range node.Dependencies {
		depDepth := esr.calculateDepthWithDependencies(depID, depGraph, visited, currentDepth+1)
		if depDepth > maxDepth {
			maxDepth = depDepth
		}
	}

	return maxDepth
}

// extractCTEName extracts CTE name from dependency node
func (esr *EnhancedSubqueryResolver) extractCTEName(node *parser.SQDependencyNode) string {
	// Look for CTE name in table references
	for _, tableRef := range node.TableRefs {
		if tableRef.IsSubquery && tableRef.Name != "" {
			return tableRef.Name
		}
	}

	// Fallback: extract from node ID
	return esr.extractCTENameFromNodeID(node.ID)
}

// extractCTENameFromNodeID extracts CTE name from node ID
func (esr *EnhancedSubqueryResolver) extractCTENameFromNodeID(nodeID string) string {
	if strings.HasPrefix(nodeID, "cte_") {
		return strings.TrimPrefix(nodeID, "cte_")
	}

	if strings.HasPrefix(nodeID, "with_") {
		return strings.TrimPrefix(nodeID, "with_")
	}

	return nodeID
}

// cacheFieldMapping caches field-level type information for quick lookup
func (esr *EnhancedSubqueryResolver) cacheFieldMapping(nodeID string, fieldInfos []*InferredFieldInfo) {
	fieldMap := make(map[string]*InferredFieldInfo)
	for _, fieldInfo := range fieldInfos {
		fieldMap[fieldInfo.Name] = fieldInfo
		if fieldInfo.Alias != "" {
			fieldMap[fieldInfo.Alias] = fieldInfo
		}
	}

	esr.fieldResolverCache[nodeID] = fieldMap
}

// ResolveSubqueryFieldType resolves field type from a specific subquery
func (esr *EnhancedSubqueryResolver) ResolveSubqueryFieldType(subqueryName, fieldName string) (*InferredFieldInfo, bool) {
	// Find the subquery node by name
	for nodeID, fieldMap := range esr.fieldResolverCache {
		cteName := esr.extractCTENameFromNodeID(nodeID)
		if cteName == subqueryName {
			if fieldInfo, exists := fieldMap[fieldName]; exists {
				return fieldInfo, true
			}
		}
	}

	return nil, false
}

// GetCompleteSubqueryInformation returns comprehensive subquery analysis
func (esr *EnhancedSubqueryResolver) GetCompleteSubqueryInformation() *SubqueryAnalysisResult {
	if esr.statementNode == nil {
		return &SubqueryAnalysisResult{
			HasSubqueries: false,
		}
	}

	analysis := esr.statementNode.GetSubqueryAnalysis()
	if analysis == nil {
		return &SubqueryAnalysisResult{
			HasSubqueries: false,
		}
	}

	// Enhance with resolved type information
	enhancedAnalysis := *analysis
	enhancedAnalysis.SubqueryTables = esr.GetAvailableSubqueryTables()

	return &enhancedAnalysis
}

// GetAvailableSubqueryTables returns table names available from resolved subqueries
func (esr *EnhancedSubqueryResolver) GetAvailableSubqueryTables() []string {
	var tables []string

	if esr.statementNode == nil {
		return tables
	}

	dependencyGraph := esr.statementNode.GetSubqueryDependencies()
	if dependencyGraph == nil {
		return tables
	}

	// Add CTE names as available tables
	for nodeID := range esr.typeCache {
		if node := dependencyGraph.GetNode(nodeID); node != nil {
			if node.NodeType == parsercommon.SQDependencyCTE {
				// Extract CTE name from node ID
				cteName := esr.extractCTENameFromNodeID(nodeID)
				if cteName != "" {
					tables = append(tables, cteName)
				}
			}
		}
	}

	return tables
}

// extractSelectFromStatement attempts to extract a SELECT statement from various statement types
func (esr *EnhancedSubqueryResolver) extractSelectFromStatement(stmt parser.StatementNode) (*parser.SelectStatement, bool) {
	switch s := stmt.(type) {
	case *parser.SelectStatement:
		return s, true
	default:
		// For other statement types, could try to extract inner SELECT
		// This would require more complex analysis based on the actual AST structure
		return nil, false
	}
}

// ValidateSubqueryReferences validates that subquery references are properly resolved
func (esr *EnhancedSubqueryResolver) ValidateSubqueryReferences() []ValidationError {
	var errors []ValidationError

	if esr.statementNode == nil {
		return errors
	}

	dependencyGraph := esr.statementNode.GetSubqueryDependencies()
	if dependencyGraph == nil {
		return errors
	}

	// Check that all referenced subqueries have type information
	for nodeID, node := range dependencyGraph.GetAllNodes() {
		if node.NodeType == parsercommon.SQDependencyMain {
			continue // Skip main query
		}

		if _, exists := esr.typeCache[nodeID]; !exists {
			errors = append(errors, ValidationError{
				Position:   -1,
				ErrorType:  "subquery_not_resolved",
				Message:    fmt.Sprintf("Subquery node %s has no type information", nodeID),
				TableName:  "",
				FieldName:  "",
				Suggestion: fmt.Sprintf("Ensure subquery %s is properly analyzed", nodeID),
			})
		}
	}

	return errors
}

// ResolveSubqueryReference resolves a table reference that might be a subquery
func (esr *EnhancedSubqueryResolver) ResolveSubqueryReference(tableName string) ([]*InferredFieldInfo, bool) {
	// Check if this table name corresponds to a CTE
	for nodeID, fieldInfos := range esr.typeCache {
		cteName := esr.extractCTENameFromNodeID(nodeID)
		if cteName == tableName {
			return fieldInfos, true
		}
	}

	return nil, false
}
