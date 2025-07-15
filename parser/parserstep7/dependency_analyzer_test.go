package parserstep7

import (
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
)

// Helper function to create a test node
func createTestNode(id string, nodeType DependencyType) *DependencyNode {
	return &DependencyNode{
		ID:           id,
		Statement:    nil, // Mock statement for testing
		NodeType:     nodeType,
		Dependencies: []string{},
	}
}

func TestDependencyAnalyzer(t *testing.T) {
	graph := NewDependencyGraph()

	// Create test nodes
	graph.AddNode(createTestNode("main", DependencyMain))
	graph.AddNode(createTestNode("cte1", DependencyCTE))
	graph.AddNode(createTestNode("cte2", DependencyCTE))
	graph.AddNode(createTestNode("sub1", DependencySubquery))
	graph.AddNode(createTestNode("sub2", DependencySubquery))

	// Create dependencies: main -> cte1 -> cte2, main -> sub1 -> sub2
	graph.AddDependency("main", "cte1")
	graph.AddDependency("cte1", "cte2")
	graph.AddDependency("main", "sub1")
	graph.AddDependency("sub1", "sub2")

	analyzer := NewDependencyAnalyzer(graph)
	result := analyzer.AnalyzeComplexDependencies()

	// Test basic statistics
	assert.Equal(t, 5, result.Stats.TotalNodes)
	assert.Equal(t, 4, result.Stats.TotalEdges)
	assert.Equal(t, 2, result.Stats.MaxDepth)
	assert.Equal(t, 0, result.Stats.CircularCount)

	// Test dependency levels
	assert.Equal(t, 0, result.Levels["cte2"])
	assert.Equal(t, 1, result.Levels["cte1"])
	assert.Equal(t, 2, result.Levels["main"])

	// No circular paths should be found
	assert.Equal(t, 0, len(result.CircularPaths))

	// Critical nodes (nodes with high fan-in) - in this small graph, threshold is 2
	assert.Equal(t, 0, len(result.CriticalNodes))
}

func TestDependencyAnalyzerCircularDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	// Create test nodes with circular dependency
	graph.AddNode(createTestNode("a", DependencySubquery))
	graph.AddNode(createTestNode("b", DependencySubquery))
	graph.AddNode(createTestNode("c", DependencySubquery))

	// Create circular dependencies: a -> b -> c -> a
	graph.AddDependency("a", "b")
	graph.AddDependency("b", "c")
	graph.AddDependency("c", "a")

	analyzer := NewDependencyAnalyzer(graph)
	result := analyzer.AnalyzeComplexDependencies()

	// Should detect circular dependencies
	assert.True(t, result.Stats.CircularCount > 0)
	assert.True(t, len(result.CircularPaths) > 0)

	// Should have detected the circular path
	foundCircularPath := false
	for _, path := range result.CircularPaths {
		if len(path) == 3 {
			foundCircularPath = true
			break
		}
	}
	assert.True(t, foundCircularPath)
}

func TestDependencyAnalyzerCriticalNodes(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a star pattern with one central node
	graph.AddNode(createTestNode("center", DependencySubquery))
	for i := 0; i < 5; i++ {
		nodeID := fmt.Sprintf("node%d", i)
		graph.AddNode(createTestNode(nodeID, DependencySubquery))
		graph.AddDependency(nodeID, "center")
	}

	analyzer := NewDependencyAnalyzer(graph)
	result := analyzer.AnalyzeComplexDependencies()

	// The center node should be identified as critical
	assert.True(t, len(result.CriticalNodes) > 0)

	foundCenter := false
	for _, criticalNode := range result.CriticalNodes {
		if criticalNode == "center" {
			foundCenter = true
			break
		}
	}
	assert.True(t, foundCenter)
}

func TestDependencyAnalyzerLongestChains(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a linear chain: a -> b -> c -> d -> e
	nodes := []string{"a", "b", "c", "d", "e"}
	for _, node := range nodes {
		graph.AddNode(createTestNode(node, DependencySubquery))
	}

	for i := 0; i < len(nodes)-1; i++ {
		graph.AddDependency(nodes[i], nodes[i+1])
	}

	analyzer := NewDependencyAnalyzer(graph)
	result := analyzer.AnalyzeComplexDependencies()

	// Should find the longest chain
	assert.True(t, len(result.DependencyChains) > 0)

	// The longest chain should have 5 nodes
	maxChainLength := 0
	for _, chain := range result.DependencyChains {
		if len(chain) > maxChainLength {
			maxChainLength = len(chain)
		}
	}
	assert.Equal(t, 5, maxChainLength)
}

func TestDependencyAnalysisResultString(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencyCTE))
	graph.AddDependency("a", "b")

	analyzer := NewDependencyAnalyzer(graph)
	result := analyzer.AnalyzeComplexDependencies()

	output := result.String()

	// Should contain basic statistics
	assert.Contains(t, output, "Total Nodes: 2")
	assert.Contains(t, output, "Total Edges: 1")
	assert.Contains(t, output, "Max Depth:")
	assert.Contains(t, output, "Average Fan-out:")
	assert.Contains(t, output, "Average Fan-in:")
}

func TestCalculateDependencyLevels(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a tree structure
	graph.AddNode(createTestNode("root", DependencyMain))
	graph.AddNode(createTestNode("level1a", DependencyCTE))
	graph.AddNode(createTestNode("level1b", DependencyCTE))
	graph.AddNode(createTestNode("level2", DependencySubquery))

	graph.AddDependency("root", "level1a")
	graph.AddDependency("root", "level1b")
	graph.AddDependency("level1a", "level2")

	analyzer := NewDependencyAnalyzer(graph)
	levels := analyzer.calculateDependencyLevels()

	// The algorithm calculates levels based on maximum child depth + 1
	// level2 has no children, so level = 0
	// level1a has level2 as child, so level = 0 + 1 = 1
	// level1b has no children, so level = 0
	// root has level1a and level1b as children, so level = max(1, 0) + 1 = 2
	assert.Equal(t, 2, levels["root"])
	assert.Equal(t, 1, levels["level1a"])
	assert.Equal(t, 0, levels["level1b"])
	assert.Equal(t, 0, levels["level2"])
}
