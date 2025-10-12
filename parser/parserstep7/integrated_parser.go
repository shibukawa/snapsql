package parserstep7

import (
	"fmt"

	snapsql "github.com/shibukawa/snapsql"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Sentinel errors
var (
	ErrInvalidStatement = snapsql.ErrInvalidStatement
)

// SubqueryParserIntegrated combines all parserstep7 functionality
type SubqueryParserIntegrated struct {
	parser       *SubqueryParser
	integrator   *ASTIntegrator
	errorHandler *ErrorReporter
}

// NewSubqueryParserIntegrated creates a new integrated subquery parser
func NewSubqueryParserIntegrated() *SubqueryParserIntegrated {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)
	errorHandler := NewErrorReporter()

	return &SubqueryParserIntegrated{
		parser:       parser,
		integrator:   integrator,
		errorHandler: errorHandler,
	}
}

// ParseStatement parses a statement and extracts all subquery dependencies
// This method stores the results directly in the StatementNode for easy access
func (spi *SubqueryParserIntegrated) ParseStatement(stmt cmn.StatementNode, functionDef interface{}) error {
	spi.errorHandler.Clear()

	if stmt == nil {
		return ErrInvalidStatement
	}

	// 0. Extract main query table references first
	mainTableRefs := spi.integrator.ExtractMainTableReferences(stmt)

	// 1. Extract subqueries and build dependency graph
	if err := spi.integrator.ExtractSubqueries(stmt); err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return err
	}

	// 2. Build field sources
	if err := spi.integrator.BuildFieldSources(); err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return err
	}

	// 3. Get dependency graph
	graph := spi.integrator.GetDependencyGraph()

	// 4. Check for circular dependencies
	if cmn.HasCircularDependencyInGraph(graph) {
		spi.errorHandler.AddError(ErrorTypeCircularDependency, "circular dependencies detected", Position{})
		return ErrCircularDependency
	}

	// 6. Get processing order using Kahn's algorithm from dependency graph
	order, err := graph.GetProcessingOrder()
	if err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return err
	}

	// 7. Store results directly in the StatementNode
	fieldSources := make(map[string]*cmn.SQFieldSource)
	tableReferences := make(map[string]*cmn.SQTableReference)

	// Add main query table references first
	for i, tr := range mainTableRefs {
		key := fmt.Sprintf("main_table_%d", i)
		tableReferences[key] = tr
	}

	// Convert and store field sources from dependency graph nodes
	allNodes := cmn.GetDependencyNodes(graph)
	for nodeID, node := range allNodes {
		for i, fs := range node.FieldSources {
			// Use node ID and field index as key since FieldSource doesn't have ID
			key := fmt.Sprintf("%s_field_%d", nodeID, i)
			fieldSources[key] = fs
		}

		for i, tr := range node.TableRefs {
			// Use node ID and table index as key since TableReference doesn't have ID
			key := fmt.Sprintf("%s_table_%d", nodeID, i)
			tableReferences[key] = tr
		}
	}

	// Store in StatementNode
	cmn.SetFieldSources(stmt, fieldSources)
	cmn.SetTableReferences(stmt, tableReferences)
	cmn.SetSubqueryDependencies(stmt, graph)
	cmn.SetProcessingOrder(stmt, order)

	// 8. Store DerivedTables (CTE and subquery information) in SubqueryAnalysisResult
	derivedTables := spi.parser.GetDerivedTables()
	analysis := &cmn.SubqueryAnalysisResult{
		HasSubqueries:    len(allNodes) > 0,
		SubqueryTables:   []string{},
		FieldSources:     fieldSources,
		TableReferences:  tableReferences,
		DependencyInfo:   graph,
		ProcessingOrder:  order,
		ValidationErrors: []cmn.ValidationError{},
		HasErrors:        spi.errorHandler.HasErrors(),
		DerivedTables:    derivedTables,
	}
	cmn.SetSubqueryAnalysis(stmt, analysis)

	return nil
}

// GetDependencyVisualization returns a basic text visualization of the dependency graph
func (spi *SubqueryParserIntegrated) GetDependencyVisualization() string {
	graph := spi.integrator.GetDependencyGraph()
	nodes := cmn.GetDependencyNodes(graph)

	if len(nodes) == 0 {
		return "No dependencies found"
	}

	result := "Dependency Graph:\n"
	for _, node := range nodes {
		result += fmt.Sprintf("- %s (%s)\n", node.ID, node.NodeType.String())
	}

	return result
}

// GetScopeVisualization returns a visualization of the scope hierarchy
func (spi *SubqueryParserIntegrated) GetScopeVisualization() string {
	graph := spi.integrator.GetDependencyGraph()
	return cmn.GetScopeHierarchyVisualizationFromGraph(graph)
}

// AddFieldSourceToNode adds a field source to a specific node
func (spi *SubqueryParserIntegrated) AddFieldSourceToNode(nodeID string, fieldSource *FieldSource) error {
	graph := spi.integrator.GetDependencyGraph()
	return cmn.AddFieldSourceToNodeInGraph(graph, nodeID, fieldSource)
}

// AddTableReferenceToNode adds a table reference to a specific node
func (spi *SubqueryParserIntegrated) AddTableReferenceToNode(nodeID string, tableRef *TableReference) error {
	graph := spi.integrator.GetDependencyGraph()
	return cmn.AddTableReferenceToNodeInGraph(graph, nodeID, tableRef)
}

// GetAccessibleFieldsForNode returns all fields accessible from a specific node
func (spi *SubqueryParserIntegrated) GetAccessibleFieldsForNode(nodeID string) ([]*FieldSource, error) {
	graph := spi.integrator.GetDependencyGraph()

	result, err := cmn.GetAccessibleFieldsForNodeInGraph(graph, nodeID)
	if err != nil {
		return nil, err
	}

	if fields, ok := result.([]*FieldSource); ok {
		return fields, nil
	}

	return nil, snapsql.ErrUnexpectedReturnType
}

// ValidateFieldAccess validates if a field can be accessed from a specific node
func (spi *SubqueryParserIntegrated) ValidateFieldAccess(nodeID, fieldName string) error {
	graph := spi.integrator.GetDependencyGraph()
	return cmn.ValidateFieldAccessForNodeInGraph(graph, nodeID, fieldName)
}

// ResolveFieldReference resolves a field reference from a specific node's perspective
func (spi *SubqueryParserIntegrated) ResolveFieldReference(nodeID, fieldName string) ([]*FieldSource, error) {
	graph := spi.integrator.GetDependencyGraph()

	result, err := cmn.ResolveFieldInNodeFromGraph(graph, nodeID, fieldName)
	if err != nil {
		return nil, err
	}

	if fields, ok := result.([]*FieldSource); ok {
		return fields, nil
	}

	return nil, snapsql.ErrUnexpectedReturnType
}

// Reset resets all internal state
func (spi *SubqueryParserIntegrated) Reset() {
	spi.parser = NewSubqueryParser()
	spi.integrator = NewASTIntegrator(spi.parser)
	spi.errorHandler.Clear()
}
