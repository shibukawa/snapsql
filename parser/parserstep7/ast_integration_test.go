package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestASTIntegrator_NewASTIntegrator(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	assert.NotZero(t, integrator)
	assert.Equal(t, parser, integrator.parser)
	assert.NotZero(t, integrator.errorHandler)
}

func TestASTIntegrator_ExtractSubqueries_NoCTE(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Create a mock statement without CTE
	stmt := &mockStatementNode{
		cte: nil,
	}

	err := integrator.ExtractSubqueries(stmt)
	assert.NoError(t, err)

	// Should have no nodes in dependency graph
	nodes := integrator.GetDependencyGraph().GetAllNodes()
	assert.Equal(t, 0, len(nodes))
}

func TestASTIntegrator_ExtractSubqueries_WithCTE(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Create a mock statement with CTE
	cte := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{Name: "test_cte"},
		},
	}
	stmt := &mockStatementNode{
		cte: cte,
	}

	err := integrator.ExtractSubqueries(stmt)
	assert.NoError(t, err)

	// Should have main node and CTE node
	nodes := integrator.GetDependencyGraph().GetAllNodes()
	assert.Equal(t, 2, len(nodes))

	// Check that we have main and CTE nodes
	var hasMain, hasCTE bool
	for _, node := range nodes {
		if node.NodeType == DependencyMain {
			hasMain = true
		}
		if node.NodeType == DependencyCTE {
			hasCTE = true
		}
	}
	assert.True(t, hasMain)
	assert.True(t, hasCTE)
}

func TestASTIntegrator_ExtractSubqueries_MultipleCTEs(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Create a mock statement with multiple CTEs
	cte := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{Name: "cte1"},
			{Name: "cte2"},
			{Name: "cte3"},
		},
	}
	stmt := &mockStatementNode{
		cte: cte,
	}

	err := integrator.ExtractSubqueries(stmt)
	assert.NoError(t, err)

	// Should have main node and 3 CTE nodes
	nodes := integrator.GetDependencyGraph().GetAllNodes()
	assert.Equal(t, 4, len(nodes))

	// Count node types
	var mainCount, cteCount int
	for _, node := range nodes {
		if node.NodeType == DependencyMain {
			mainCount++
		}
		if node.NodeType == DependencyCTE {
			cteCount++
		}
	}
	assert.Equal(t, 1, mainCount)
	assert.Equal(t, 3, cteCount)
}

func TestASTIntegrator_BuildFieldSources(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Add a test node to the dependency graph
	stmt := &cmn.SelectStatement{}
	node := &DependencyNode{
		ID:        "test",
		Statement: stmt,
		NodeType:  DependencyMain,
	}
	integrator.parser.dependencies.AddNode(node)

	err := integrator.BuildFieldSources()
	assert.NoError(t, err)
}

func TestASTIntegrator_GetDependencyGraph(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	graph := integrator.GetDependencyGraph()
	assert.NotZero(t, graph)
	assert.Equal(t, parser.dependencies, graph)
}

func TestASTIntegrator_GetErrors(t *testing.T) {
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	errors := integrator.GetErrors()
	assert.Equal(t, 0, len(errors))
}

// Mock statement node for testing
type mockStatementNode struct {
	cte                  *cmn.WithClause
	fieldSources         map[string]cmn.FieldSourceInterface
	tableReferences      map[string]cmn.TableReferenceInterface
	subqueryDependencies cmn.DependencyGraphInterface
}

func (m *mockStatementNode) CTE() *cmn.WithClause {
	return m.cte
}

func (m *mockStatementNode) LeadingTokens() []tokenizer.Token {
	return nil
}

func (m *mockStatementNode) Clauses() []cmn.ClauseNode {
	return nil
}

func (m *mockStatementNode) GetFieldSources() map[string]cmn.FieldSourceInterface {
	if m.fieldSources == nil {
		m.fieldSources = make(map[string]cmn.FieldSourceInterface)
	}
	return m.fieldSources
}

func (m *mockStatementNode) GetTableReferences() map[string]cmn.TableReferenceInterface {
	if m.tableReferences == nil {
		m.tableReferences = make(map[string]cmn.TableReferenceInterface)
	}
	return m.tableReferences
}

func (m *mockStatementNode) GetSubqueryDependencies() cmn.DependencyGraphInterface {
	return m.subqueryDependencies
}

func (m *mockStatementNode) SetFieldSources(fs map[string]cmn.FieldSourceInterface) {
	m.fieldSources = fs
}

func (m *mockStatementNode) SetTableReferences(tr map[string]cmn.TableReferenceInterface) {
	m.tableReferences = tr
}

func (m *mockStatementNode) SetSubqueryDependencies(dg cmn.DependencyGraphInterface) {
	m.subqueryDependencies = dg
}

func (m *mockStatementNode) FindFieldReference(tableOrAlias, fieldOrReference string) cmn.FieldSourceInterface {
	return nil
}

func (m *mockStatementNode) FindTableReference(tableOrAlias string) cmn.TableReferenceInterface {
	return nil
}

func (m *mockStatementNode) Position() tokenizer.Position {
	return tokenizer.Position{}
}

func (m *mockStatementNode) RawTokens() []tokenizer.Token {
	return nil
}

func (m *mockStatementNode) String() string {
	return "mock_statement"
}

func (m *mockStatementNode) Type() cmn.NodeType {
	return cmn.SELECT_STATEMENT
}
