package parserstep7

import (
	"errors"
	"fmt"
	"sync"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// SubqueryParser handles subquery parsing and dependency resolution with advanced features
type SubqueryParserAdvanced struct {
	idGenerator     *IDGenerator
	dependencyGraph *OptimizedDependencyGraph
	fieldSources    map[string]*FieldSource
	errorReporter   *ErrorReporter
	optimizer       *PerformanceOptimizer
	visualizer      *DependencyVisualizer
	mu              sync.RWMutex
}

// NewSubqueryParserAdvanced creates a new advanced subquery parser
func NewSubqueryParserAdvanced() *SubqueryParserAdvanced {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)

	parser := &SubqueryParserAdvanced{
		idGenerator:     NewIDGenerator(),
		dependencyGraph: NewOptimizedDependencyGraph(optimizer),
		fieldSources:    make(map[string]*FieldSource),
		errorReporter:   NewErrorReporter(),
		optimizer:       optimizer,
	}

	// Set up visualizer with default options
	visualOptions := DefaultVisualizationOptions()
	parser.visualizer = NewDependencyVisualizer(parser.dependencyGraph.DependencyGraph, visualOptions)

	return parser
}

// ParseSubqueries parses subqueries with enhanced error handling and performance optimization
func (sp *SubqueryParserAdvanced) ParseSubqueries(stmt cmn.StatementNode) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Clear previous state
	sp.errorReporter.Clear()
	sp.fieldSources = make(map[string]*FieldSource)
	sp.idGenerator.ResetAll()

	// 1. Build dependency graph by detecting subqueries
	if err := sp.buildDependencyGraph(stmt); err != nil {
		return err
	}

	// 2. Build performance indexes
	sp.dependencyGraph.BuildIndexes()

	// 3. Analyze complex dependencies
	analyzer := NewDependencyAnalyzer(sp.dependencyGraph.DependencyGraph)
	analysisResult := analyzer.AnalyzeComplexDependencies()

	// 4. Check for circular dependencies with detailed error reporting
	if len(analysisResult.CircularPaths) > 0 {
		sp.reportCircularDependencies(analysisResult.CircularPaths)
		return ErrCircularDependency
	}

	// 5. Determine optimized processing order
	processingOrder, err := sp.dependencyGraph.GetProcessingOrderOptimized()
	if err != nil {
		return err
	}

	// 6. Parse subqueries in dependency order with enhanced error handling
	for _, nodeID := range processingOrder {
		if err := sp.parseSubqueryNodeEnhanced(nodeID); err != nil {
			// Continue processing other nodes to collect all errors
			pos := Position{File: "unknown"}
			sp.errorReporter.AddError(ErrorTypeInvalidSubquery, err.Error(), pos)
		}
	}

	// 7. Build field sources with validation
	if err := sp.buildFieldSourcesEnhanced(stmt); err != nil {
		return err
	}

	// 8. Return aggregated errors if any
	if sp.errorReporter.HasErrors() {
		return fmt.Errorf("subquery parsing failed with %d errors: %s",
			len(sp.errorReporter.GetErrors()), sp.errorReporter.String())
	}

	return nil
}

// GetFieldSources returns the computed field sources with enhanced metadata
func (sp *SubqueryParserAdvanced) GetFieldSources() map[string]*FieldSource {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*FieldSource)
	for k, v := range sp.fieldSources {
		result[k] = v
	}
	return result
}

// GetDependencyGraph returns the dependency graph for external analysis
func (sp *SubqueryParserAdvanced) GetDependencyGraph() *OptimizedDependencyGraph {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.dependencyGraph
}

// GetErrorReporter returns the error reporter for detailed error analysis
func (sp *SubqueryParserAdvanced) GetErrorReporter() *ErrorReporter {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.errorReporter
}

// GetPerformanceStats returns performance statistics
func (sp *SubqueryParserAdvanced) GetPerformanceStats() PerformanceMetrics {
	return sp.optimizer.stats.GetStats()
}

// GenerateVisualization creates a visual representation of dependencies
func (sp *SubqueryParserAdvanced) GenerateVisualization(format VisualizationFormat) (string, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	options := DefaultVisualizationOptions()
	options.Format = format

	visualizer := NewDependencyVisualizer(sp.dependencyGraph.DependencyGraph, options)
	return visualizer.Generate()
}

// GenerateDebugInfo creates comprehensive debug information
func (sp *SubqueryParserAdvanced) GenerateDebugInfo() string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.visualizer.GenerateDebugInfo()
}

// buildDependencyGraph detects subqueries and builds the dependency graph
func (sp *SubqueryParserAdvanced) buildDependencyGraph(stmt cmn.StatementNode) error {
	// Create main query node
	mainID := sp.idGenerator.Generate("main")
	mainNode := &DependencyNode{
		ID:        mainID,
		Statement: stmt,
		NodeType:  DependencyMain,
	}

	if err := sp.dependencyGraph.AddNode(mainNode); err != nil {
		return fmt.Errorf("failed to add main node: %w", err)
	}

	// TODO: Implement actual subquery detection
	// This would traverse the AST and identify:
	// - CTE definitions in WITH clauses
	// - Subqueries in FROM clauses
	// - Dependencies between them

	return nil
}

// parseSubqueryNodeEnhanced parses a single subquery node with enhanced error handling
func (sp *SubqueryParserAdvanced) parseSubqueryNodeEnhanced(nodeID string) error {
	// TODO: Implement enhanced subquery parsing
	// This would include:
	// - Detailed syntax validation
	// - Scope checking
	// - Type inference preparation
	// - Field availability validation

	return nil
}

// buildFieldSourcesEnhanced builds field sources with enhanced validation
func (sp *SubqueryParserAdvanced) buildFieldSourcesEnhanced(stmt cmn.StatementNode) error {
	// TODO: Implement enhanced field source building
	// This would include:
	// - Field source type detection
	// - Scope validation
	// - Reference resolution
	// - Type compatibility checking

	return nil
}

// reportCircularDependencies reports circular dependency errors with detailed paths
func (sp *SubqueryParserAdvanced) reportCircularDependencies(circularPaths [][]string) {
	collector := NewErrorCollector(sp.errorReporter)

	for _, path := range circularPaths {
		pos := Position{File: "query"} // In real implementation, would extract actual position
		collector.ReportCircularDependency(path, pos)
	}
}

// ValidateFieldReference validates a field reference against available sources
func (sp *SubqueryParserAdvanced) ValidateFieldReference(fieldName string, currentScope string) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Find available fields in current scope
	var availableFields []string
	for fieldID, source := range sp.fieldSources {
		// TODO: Implement proper scope checking
		if source != nil {
			availableFields = append(availableFields, fieldID)
		}
	}

	// Check if field is available
	for _, available := range availableFields {
		if available == fieldName {
			return nil // Field found
		}
	}

	// Field not found, report error
	collector := NewErrorCollector(sp.errorReporter)
	pos := Position{File: "query"} // In real implementation, would extract actual position
	collector.ReportUnresolvedReference(fieldName, availableFields, pos)

	return fmt.Errorf("field '%s' not found in scope '%s'", fieldName, currentScope)
}

// GetSubqueryDependencies returns dependencies for a specific subquery
func (sp *SubqueryParserAdvanced) GetSubqueryDependencies(subqueryID string) ([]string, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	node, exists := sp.dependencyGraph.nodes[subqueryID]
	if !exists {
		return nil, fmt.Errorf("subquery '%s' not found", subqueryID)
	}

	return node.Dependencies, nil
}

// AnalyzeDependencyComplexity analyzes the complexity of the dependency graph
func (sp *SubqueryParserAdvanced) AnalyzeDependencyComplexity() *DependencyAnalysisResult {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	analyzer := NewDependencyAnalyzer(sp.dependencyGraph.DependencyGraph)
	return analyzer.AnalyzeComplexDependencies()
}

// OptimizeForPerformance applies performance optimizations
func (sp *SubqueryParserAdvanced) OptimizeForPerformance(config PerformanceConfig) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.optimizer = NewPerformanceOptimizer(config)
	sp.dependencyGraph.optimizer = sp.optimizer
}

// SetVisualizationOptions configures visualization options
func (sp *SubqueryParserAdvanced) SetVisualizationOptions(options VisualizationOptions) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.visualizer = NewDependencyVisualizer(sp.dependencyGraph.DependencyGraph, options)
}

// Additional errors for enhanced parsing
var (
	ErrAdvancedCircularDependency = errors.New("advanced circular dependency detected")
	ErrAdvancedInvalidSubquery    = errors.New("advanced invalid subquery")
	ErrAdvancedScopeViolation     = errors.New("advanced scope violation")
)
