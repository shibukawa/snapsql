package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestSubqueryParserIntegrated_NewSubqueryParserIntegrated(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	assert.NotZero(t, parser)
	assert.NotZero(t, parser.parser)
	assert.NotZero(t, parser.integrator)
	assert.NotZero(t, parser.errorHandler)
}

func TestSubqueryParserIntegrated_ParseStatement_NilStatement(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	result, err := parser.ParseStatement(nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatement, err)
	assert.Zero(t, result)
}

func TestSubqueryParserIntegrated_ParseStatement_NoCTE(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Create a mock statement without CTE
	stmt := &mockStatementNodeIntegrated{
		cte: nil,
	}

	result, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)
	assert.NotZero(t, result)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, len(result.Errors))
	assert.Equal(t, 0, len(result.DependencyGraph.GetAllNodes()))
	assert.Equal(t, 0, len(result.ProcessingOrder))
}

func TestSubqueryParserIntegrated_ParseStatement_WithCTE(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Create a mock statement with CTE
	cte := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{Name: "test_cte"},
		},
	}
	stmt := &mockStatementNodeIntegrated{
		cte: cte,
	}

	result, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)
	assert.NotZero(t, result)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, len(result.Errors))
	assert.Equal(t, 2, len(result.DependencyGraph.GetAllNodes())) // main + CTE
	assert.Equal(t, 2, len(result.ProcessingOrder))
}

func TestSubqueryParserIntegrated_GetDependencyVisualization_Empty(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	visualization := parser.GetDependencyVisualization()
	assert.Equal(t, "No dependencies found", visualization)
}

func TestSubqueryParserIntegrated_GetDependencyVisualization_WithNodes(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Add some nodes to the dependency graph
	stmt := &mockStatementNodeIntegrated{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{
				{Name: "test_cte"},
			},
		},
	}

	_, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)

	visualization := parser.GetDependencyVisualization()
	assert.Contains(t, visualization, "Dependency Graph:")
	assert.Contains(t, visualization, "Main")
	assert.Contains(t, visualization, "CTE")
}

func TestSubqueryParserIntegrated_Reset(t *testing.T) {
	parser := NewSubqueryParserIntegrated()

	// Parse a statement to add some state
	stmt := &mockStatementNodeIntegrated{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{
				{Name: "test_cte"},
			},
		},
	}

	_, err := parser.ParseStatement(stmt)
	assert.NoError(t, err)

	// Verify state exists
	assert.Equal(t, 2, len(parser.integrator.GetDependencyGraph().GetAllNodes()))

	// Reset and verify state is cleared
	parser.Reset()
	assert.Equal(t, 0, len(parser.integrator.GetDependencyGraph().GetAllNodes()))
}

func TestParseResult_String(t *testing.T) {
	// Test with errors
	result := &ParseResult{
		HasErrors: true,
	}
	assert.Equal(t, "ParseResult: Has errors", result.String())

	// Test without errors
	graph := NewDependencyGraph()
	node := &DependencyNode{
		ID:       "test",
		NodeType: DependencyMain,
	}
	graph.AddNode(node)

	result = &ParseResult{
		DependencyGraph: graph,
		ProcessingOrder: []string{"test"},
		HasErrors:       false,
	}
	assert.Equal(t, "ParseResult: 1 nodes, 1 processing steps", result.String())
}

// Mock statement node for integrated testing
type mockStatementNodeIntegrated struct {
	cte                  *cmn.WithClause
	fieldSources         map[string]cmn.FieldSourceInterface
	tableReferences      map[string]cmn.TableReferenceInterface
	subqueryDependencies cmn.DependencyGraphInterface
}

func (m *mockStatementNodeIntegrated) CTE() *cmn.WithClause {
	return m.cte
}

func (m *mockStatementNodeIntegrated) LeadingTokens() []tokenizer.Token {
	return nil
}

func (m *mockStatementNodeIntegrated) Clauses() []cmn.ClauseNode {
	return nil
}

func (m *mockStatementNodeIntegrated) GetFieldSources() map[string]cmn.FieldSourceInterface {
	if m.fieldSources == nil {
		m.fieldSources = make(map[string]cmn.FieldSourceInterface)
	}
	return m.fieldSources
}

func (m *mockStatementNodeIntegrated) GetTableReferences() map[string]cmn.TableReferenceInterface {
	if m.tableReferences == nil {
		m.tableReferences = make(map[string]cmn.TableReferenceInterface)
	}
	return m.tableReferences
}

func (m *mockStatementNodeIntegrated) GetSubqueryDependencies() cmn.DependencyGraphInterface {
	return m.subqueryDependencies
}

func (m *mockStatementNodeIntegrated) SetFieldSources(fs map[string]cmn.FieldSourceInterface) {
	m.fieldSources = fs
}

func (m *mockStatementNodeIntegrated) SetTableReferences(tr map[string]cmn.TableReferenceInterface) {
	m.tableReferences = tr
}

func (m *mockStatementNodeIntegrated) SetSubqueryDependencies(dg cmn.DependencyGraphInterface) {
	m.subqueryDependencies = dg
}

func (m *mockStatementNodeIntegrated) FindFieldReference(tableOrAlias, fieldOrReference string) cmn.FieldSourceInterface {
	return nil
}

func (m *mockStatementNodeIntegrated) FindTableReference(tableOrAlias string) cmn.TableReferenceInterface {
	return nil
}

func (m *mockStatementNodeIntegrated) Position() tokenizer.Position {
	return tokenizer.Position{}
}

func (m *mockStatementNodeIntegrated) RawTokens() []tokenizer.Token {
	return nil
}

func (m *mockStatementNodeIntegrated) String() string {
	return "mock_statement_integrated"
}

func (m *mockStatementNodeIntegrated) Type() cmn.NodeType {
	return cmn.SELECT_STATEMENT
}

// Implement new SubqueryAnalysisInfo methods
func (m *mockStatementNodeIntegrated) GetSubqueryAnalysis() *cmn.SubqueryAnalysisInfo {
	return nil
}

func (m *mockStatementNodeIntegrated) SetSubqueryAnalysis(info *cmn.SubqueryAnalysisInfo) {
	// Mock implementation does nothing
}

func (m *mockStatementNodeIntegrated) HasSubqueryAnalysis() bool {
	return false
}
