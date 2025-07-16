package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
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
		if node.NodeType == cmn.SQDependencyMain {
			hasMain = true
		}
		if node.NodeType == cmn.SQDependencyCTE {
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
		if node.NodeType == cmn.SQDependencyMain {
			mainCount++
		}
		if node.NodeType == cmn.SQDependencyCTE {
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
	node := &cmn.SQDependencyNode{
		ID:        "test",
		Statement: stmt,
		NodeType:  cmn.SQDependencyMain,
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

// mockStatementNode is now defined in test_helpers.go
