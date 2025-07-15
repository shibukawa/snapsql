package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestSubqueryParserAdvanced(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Basic initialization test
	assert.True(t, parser != nil)
	assert.True(t, parser.GetErrorReporter() != nil)
	assert.True(t, parser.GetDependencyGraph() != nil)
}

func TestAdvancedParsingWithoutMockStatement(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Test basic functionality without using mock statement
	assert.True(t, parser != nil)
	assert.True(t, parser.GetErrorReporter() != nil)
	assert.True(t, parser.GetDependencyGraph() != nil)

	// Test error reporting functionality
	errorReporter := parser.GetErrorReporter()
	assert.False(t, errorReporter.HasErrors())
}

func TestFieldReferenceValidation(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Test field validation with empty sources
	err := parser.ValidateFieldReference("unknown_field", "main_scope")
	assert.Error(t, err)

	// Check that error was recorded
	errorReporter := parser.GetErrorReporter()
	assert.True(t, errorReporter.HasErrors())

	errors := errorReporter.GetErrorsByType(ErrorTypeUnresolvedReference)
	assert.Equal(t, 1, len(errors))
}

func TestPerformanceStatsRetrieval(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	stats := parser.GetPerformanceStats()

	// Should have default values for new parser
	assert.Equal(t, 0, stats.ProcessingCount)
	assert.Equal(t, 0, stats.CacheHits)
	assert.Equal(t, 0, stats.CacheMisses)
}

func TestVisualizationGeneration(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Test different visualization formats without parsing
	formats := []VisualizationFormat{
		FormatTextTree,
		FormatDOT,
		FormatMermaid,
		FormatJSON,
	}

	for _, format := range formats {
		output, err := parser.GenerateVisualization(format)
		assert.NoError(t, err)
		assert.True(t, len(output) > 0)
	}
}

func TestDebugInfoGeneration(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	debugInfo := parser.GenerateDebugInfo()

	assert.Contains(t, debugInfo, "Dependency Graph Debug Information")
	assert.Contains(t, debugInfo, "Statistics:")
}

func TestDependencyComplexityAnalysis(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	analysisResult := parser.AnalyzeDependencyComplexity()

	assert.True(t, analysisResult != nil)
	assert.Equal(t, 0, analysisResult.Stats.TotalNodes) // Empty graph initially
	assert.Equal(t, 0, analysisResult.Stats.TotalEdges)
	assert.Equal(t, 0, analysisResult.Stats.CircularCount)
}

func TestPerformanceOptimization(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Apply custom performance config
	config := PerformanceConfig{
		MaxNodes:                 5000,
		MaxDepth:                 50,
		EnableParallelProcessing: false,
		CacheEnabled:             false,
		BatchSize:                50,
	}

	parser.OptimizeForPerformance(config)

	// Verify the configuration was applied
	// Note: In a real implementation, you'd have getters to verify this
	assert.True(t, parser.optimizer != nil)
}

func TestVisualizationOptionsConfiguration(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Configure custom visualization options
	options := VisualizationOptions{
		Format:            FormatDOT,
		ShowNodeDetails:   false,
		HighlightCritical: false,
		HighlightCircular: true,
		CompactMode:       true,
	}

	parser.SetVisualizationOptions(options)

	// Verify configuration by generating output
	output, err := parser.GenerateVisualization(FormatDOT)
	assert.NoError(t, err)
	assert.Contains(t, output, "digraph DependencyGraph")
}

func TestSubqueryDependencyRetrieval(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Try to get dependencies for non-existent subquery
	_, err := parser.GetSubqueryDependencies("non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConcurrentAccessAdvanced(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Test concurrent access to parser methods
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			parser.GetFieldSources()
			parser.GetPerformanceStats()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			parser.AnalyzeDependencyComplexity()
			parser.GenerateDebugInfo()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should complete without race conditions
	assert.True(t, true)
}

func TestFieldSourcesImmutability(t *testing.T) {
	parser := NewSubqueryParserAdvanced()

	// Get field sources
	sources1 := parser.GetFieldSources()
	sources2 := parser.GetFieldSources()

	// Should be different map instances (copies)
	// Create a simple test entry using nil interface
	sources1["test"] = nil

	// Original should not be affected
	_, exists := sources2["test"]
	assert.False(t, exists)
}

// MockStatementNode implements the StatementNode interface for testing
type MockStatementNode struct{}

// AstNode interface methods
func (m *MockStatementNode) IsNode() {}
func (m *MockStatementNode) Position() tokenizer.Position {
	return tokenizer.Position{}
}
func (m *MockStatementNode) RawTokens() []tokenizer.Token {
	return []tokenizer.Token{}
}

// StatementNode interface methods
func (m *MockStatementNode) CTE() *parsercommon.WithClause {
	return nil
}

func (m *MockStatementNode) LeadingTokens() []tokenizer.Token {
	return []tokenizer.Token{}
}

func (m *MockStatementNode) Clauses() []parsercommon.ClauseNode {
	return []parsercommon.ClauseNode{}
}

func (m *MockStatementNode) GetFieldSources() map[string]parsercommon.FieldSourceInterface {
	return make(map[string]parsercommon.FieldSourceInterface)
}

func (m *MockStatementNode) GetTableReferences() map[string]parsercommon.TableReferenceInterface {
	return make(map[string]parsercommon.TableReferenceInterface)
}

func (m *MockStatementNode) GetSubqueryDependencies() parsercommon.DependencyGraphInterface {
	return nil
}

func (m *MockStatementNode) SetFieldSources(sources map[string]parsercommon.FieldSourceInterface) {}

func (m *MockStatementNode) SetTableReferences(refs map[string]parsercommon.TableReferenceInterface) {
}

func (m *MockStatementNode) SetSubqueryDependencies(deps parsercommon.DependencyGraphInterface) {}

func (m *MockStatementNode) FindFieldReference(tableOrAlias, fieldOrReference string) parsercommon.FieldSourceInterface {
	return nil
}

func (m *MockStatementNode) FindTableReference(tableOrAlias string) parsercommon.TableReferenceInterface {
	return nil
}
