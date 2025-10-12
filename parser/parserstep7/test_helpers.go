package parserstep7

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/require"
)

// parseFullPipeline executes the complete parsing pipeline (step1-7) from SQL string
// This is similar to parserstep5's parseFullPipeline but includes all steps up to parserstep7
func parseFullPipeline(t *testing.T, sql string) cmn.StatementNode {
	t.Helper()

	// Step 1: Tokenize
	tokens, err := tokenizer.Tokenize(sql)
	require.NoError(t, err, "tokenization failed")

	// Step 2: Basic parsing
	step2Result, err := parserstep2.Execute(tokens)
	require.NoError(t, err, "parserstep2 failed")

	// Step 3: Clause parsing
	err = parserstep3.Execute(step2Result)
	require.NoError(t, err, "parserstep3 failed")

	// Step 4: Table reference validation
	err = parserstep4.Execute(step2Result)
	require.NoError(t, err, "parserstep4 failed")

	// Step 5: Directive processing
	err = parserstep5.Execute(step2Result, nil)
	require.NoError(t, err, "parserstep5 failed")

	// Step 6: (currently no-op, but included for future compatibility)
	// No parserstep6 implementation yet

	// Step 7: Subquery and CTE analysis
	subqueryParser := NewSubqueryParserIntegrated()
	err = subqueryParser.ParseStatement(step2Result, nil)
	require.NoError(t, err, "parserstep7 failed")

	return step2Result
}

// assertHasTableReference verifies that at least one table reference matches the predicate
func assertHasTableReference(t *testing.T, stmt cmn.StatementNode, predicate func(*cmn.SQTableReference) bool) {
	t.Helper()

	tableRefs := stmt.GetTableReferences()
	for _, ref := range tableRefs {
		if predicate(ref) {
			return // Found a match
		}
	}

	require.Fail(t, "table reference not found", "no table reference matched the predicate")
}

// assertDependencyGraphNode verifies that a specific node exists in the dependency graph
func assertDependencyGraphNode(t *testing.T, stmt cmn.StatementNode,
	nodeID string, expectedNodeType cmn.SQDependencyType) {
	t.Helper()

	graph := stmt.GetSubqueryDependencies()
	require.NotNil(t, graph, "dependency graph should not be nil")

	node := graph.GetNode(nodeID)
	require.NotNil(t, node, "node '%s' should exist in dependency graph", nodeID)
	require.Equal(t, expectedNodeType, node.NodeType,
		"node type mismatch for '%s'", nodeID)
}

// assertNoDependencyErrors verifies that there are no dependency-related errors
func assertNoDependencyErrors(t *testing.T, stmt cmn.StatementNode) {
	t.Helper()

	analysis := stmt.GetSubqueryAnalysis()
	require.NotNil(t, analysis, "subquery analysis should not be nil")
	require.False(t, analysis.HasErrors, "should not have errors")
	require.Empty(t, analysis.ValidationErrors, "should have no validation errors")
}
