package parserstep7

import (
	"errors"
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Sentinel errors
var (
	ErrInvalidStatement = errors.New("invalid statement for subquery parsing")
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
func (spi *SubqueryParserIntegrated) ParseStatement(stmt cmn.StatementNode) (*ParseResult, error) {
	spi.errorHandler.Clear()

	if stmt == nil {
		return nil, ErrInvalidStatement
	}

	// 1. Extract subqueries and build dependency graph
	if err := spi.integrator.ExtractSubqueries(stmt); err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return nil, err
	}

	// 2. Build field sources
	if err := spi.integrator.BuildFieldSources(); err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return nil, err
	}

	// 3. Get dependency graph
	graph := spi.integrator.GetDependencyGraph()

	// 4. Build scope hierarchy for field source management
	if err := graph.BuildScopeHierarchy(); err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, fmt.Sprintf("failed to build scope hierarchy: %v", err), Position{})
		return nil, err
	}

	// 5. Check for circular dependencies
	if graph.HasCircularDependency() {
		spi.errorHandler.AddError(ErrorTypeCircularDependency, "circular dependencies detected", Position{})
		return &ParseResult{
			DependencyGraph: graph,
			HasErrors:       true,
			Errors:          spi.errorHandler.GetErrors(),
		}, ErrCircularDependency
	}

	// 6. Get processing order
	order, err := graph.GetProcessingOrder()
	if err != nil {
		spi.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
		return nil, err
	}

	// 7. Store results directly in the StatementNode
	fieldSources := make(map[string]cmn.FieldSourceInterface)
	tableReferences := make(map[string]cmn.TableReferenceInterface)

	// Convert and store field sources from dependency graph nodes
	allNodes := graph.GetAllNodes()
	for nodeID, node := range allNodes {
		for i, fs := range node.FieldSources {
			// Use node ID and field index as key since FieldSource doesn't have ID
			key := fmt.Sprintf("%s_field_%d", nodeID, i)
			fieldSources[key] = cmn.FieldSourceInterface(fs)
		}
		for i, tr := range node.TableRefs {
			// Use node ID and table index as key since TableReference doesn't have ID
			key := fmt.Sprintf("%s_table_%d", nodeID, i)
			tableReferences[key] = cmn.TableReferenceInterface(tr)
		}
	}

	// Store in StatementNode
	stmt.SetFieldSources(fieldSources)
	stmt.SetTableReferences(tableReferences)
	stmt.SetSubqueryDependencies(cmn.DependencyGraphInterface(graph))

	return &ParseResult{
		DependencyGraph: graph,
		ProcessingOrder: order,
		HasErrors:       spi.errorHandler.HasErrors(),
		Errors:          spi.errorHandler.GetErrors(),
	}, nil
}

// ParseResult contains the complete result of subquery parsing
type ParseResult struct {
	DependencyGraph *DependencyGraph // Original dependency graph
	ProcessingOrder []string         // Recommended processing order
	HasErrors       bool             // Whether errors occurred
	Errors          []*ParseError    // List of errors
}

// String returns a summary of the parse result
func (pr *ParseResult) String() string {
	if pr.HasErrors {
		return "ParseResult: Has errors"
	}

	nodeCount := len(pr.DependencyGraph.GetAllNodes())
	return fmt.Sprintf("ParseResult: %d nodes, %d processing steps", nodeCount, len(pr.ProcessingOrder))
}

// GetFieldSourcesForNode returns all field sources available to a specific node
func (pr *ParseResult) GetFieldSourcesForNode(nodeID string) ([]*FieldSource, error) {
	if pr.DependencyGraph == nil {
		return nil, fmt.Errorf("no dependency graph available")
	}
	return pr.DependencyGraph.GetAccessibleFieldsForNode(nodeID)
}

// ValidateFieldAccessForNode validates if a field can be accessed from a specific node
func (pr *ParseResult) ValidateFieldAccessForNode(nodeID, fieldName string) error {
	if pr.DependencyGraph == nil {
		return fmt.Errorf("no dependency graph available")
	}
	return pr.DependencyGraph.ValidateFieldAccessForNode(nodeID, fieldName)
}

// ResolveFieldForNode resolves a field reference from a specific node's perspective
func (pr *ParseResult) ResolveFieldForNode(nodeID, fieldName string) ([]*FieldSource, error) {
	if pr.DependencyGraph == nil {
		return nil, fmt.Errorf("no dependency graph available")
	}
	return pr.DependencyGraph.ResolveFieldInNode(nodeID, fieldName)
}

// GetScopeHierarchy returns a visualization of the scope hierarchy
func (pr *ParseResult) GetScopeHierarchy() string {
	if pr.DependencyGraph == nil {
		return "No dependency graph available"
	}
	return pr.DependencyGraph.GetScopeHierarchyVisualization()
}

// GetDependencyVisualization returns a basic text visualization of the dependency graph
func (spi *SubqueryParserIntegrated) GetDependencyVisualization() string {
	graph := spi.integrator.GetDependencyGraph()
	nodes := graph.GetAllNodes()

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
	return graph.GetScopeHierarchyVisualization()
}

// AddFieldSourceToNode adds a field source to a specific node
func (spi *SubqueryParserIntegrated) AddFieldSourceToNode(nodeID string, fieldSource *FieldSource) error {
	graph := spi.integrator.GetDependencyGraph()
	return graph.AddFieldSourceToNode(nodeID, fieldSource)
}

// AddTableReferenceToNode adds a table reference to a specific node
func (spi *SubqueryParserIntegrated) AddTableReferenceToNode(nodeID string, tableRef *TableReference) error {
	graph := spi.integrator.GetDependencyGraph()
	return graph.AddTableReferenceToNode(nodeID, tableRef)
}

// GetAccessibleFieldsForNode returns all fields accessible from a specific node
func (spi *SubqueryParserIntegrated) GetAccessibleFieldsForNode(nodeID string) ([]*FieldSource, error) {
	graph := spi.integrator.GetDependencyGraph()
	return graph.GetAccessibleFieldsForNode(nodeID)
}

// ValidateFieldAccess validates if a field can be accessed from a specific node
func (spi *SubqueryParserIntegrated) ValidateFieldAccess(nodeID, fieldName string) error {
	graph := spi.integrator.GetDependencyGraph()
	return graph.ValidateFieldAccessForNode(nodeID, fieldName)
}

// ResolveFieldReference resolves a field reference from a specific node's perspective
func (spi *SubqueryParserIntegrated) ResolveFieldReference(nodeID, fieldName string) ([]*FieldSource, error) {
	graph := spi.integrator.GetDependencyGraph()
	return graph.ResolveFieldInNode(nodeID, fieldName)
}

// Reset resets all internal state
func (spi *SubqueryParserIntegrated) Reset() {
	spi.parser = NewSubqueryParser()
	spi.integrator = NewASTIntegrator(spi.parser)
	spi.errorHandler.Clear()
}
