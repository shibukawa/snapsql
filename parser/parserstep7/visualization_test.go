package parserstep7

import (
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestVisualizationTextTree(t *testing.T) {
	graph := NewDependencyGraph()

	// Create test graph: main -> cte1 -> cte2
	graph.AddNode(createTestNode("main", DependencyMain))
	graph.AddNode(createTestNode("cte1", DependencyCTE))
	graph.AddNode(createTestNode("cte2", DependencyCTE))

	graph.AddDependency("main", "cte1")
	graph.AddDependency("cte1", "cte2")

	options := DefaultVisualizationOptions()
	visualizer := NewDependencyVisualizer(graph, options)

	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should contain tree structure
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "cte1")
	assert.Contains(t, output, "cte2")
	assert.Contains(t, output, "└──") // Tree structure
	assert.Contains(t, output, "Summary")
}

func TestVisualizationDOT(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencyCTE))
	graph.AddDependency("a", "b")

	options := VisualizationOptions{
		Format:            FormatDOT,
		ShowNodeDetails:   true,
		HighlightCritical: false,
		HighlightCircular: false,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should contain DOT format elements
	assert.Contains(t, output, "digraph DependencyGraph")
	assert.Contains(t, output, "rankdir=TB")
	assert.Contains(t, output, "\"a\" -> \"b\"")
	assert.Contains(t, output, "Main") // Node type in label
}

func TestVisualizationMermaid(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode(createTestNode("node1", DependencySubquery))
	graph.AddNode(createTestNode("node2", DependencySubquery))
	graph.AddDependency("node1", "node2")

	options := VisualizationOptions{
		Format:            FormatMermaid,
		ShowNodeDetails:   false,
		HighlightCritical: false,
		HighlightCircular: false,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should contain Mermaid format elements
	assert.Contains(t, output, "graph TD")
	assert.Contains(t, output, "node1 --> node2")
	assert.Contains(t, output, "node1[\"node1\"]")
}

func TestVisualizationJSON(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode(createTestNode("x", DependencyMain))
	graph.AddNode(createTestNode("y", DependencyCTE))
	graph.AddDependency("x", "y")

	options := VisualizationOptions{
		Format: FormatJSON,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should contain JSON structure
	assert.Contains(t, output, "\"nodes\":")
	assert.Contains(t, output, "\"edges\":")
	assert.Contains(t, output, "\"id\": \"x\"")
	assert.Contains(t, output, "\"from\": \"x\", \"to\": \"y\"")
}

func TestVisualizationHighlighting(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a star pattern to generate critical nodes
	graph.AddNode(createTestNode("center", DependencySubquery))
	for i := 0; i < 5; i++ {
		nodeID := fmt.Sprintf("node%d", i)
		graph.AddNode(createTestNode(nodeID, DependencySubquery))
		graph.AddDependency(nodeID, "center")
	}

	options := VisualizationOptions{
		Format:            FormatTextTree,
		HighlightCritical: true,
		HighlightCircular: false,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should highlight critical nodes
	assert.Contains(t, output, "[CRITICAL]")
	assert.Contains(t, output, "center")
}

func TestVisualizationCircularHighlighting(t *testing.T) {
	graph := NewDependencyGraph()

	// Create circular dependency
	graph.AddNode(createTestNode("a", DependencySubquery))
	graph.AddNode(createTestNode("b", DependencySubquery))
	graph.AddNode(createTestNode("c", DependencySubquery))

	graph.AddDependency("a", "b")
	graph.AddDependency("b", "c")
	graph.AddDependency("c", "a")

	options := VisualizationOptions{
		Format:            FormatTextTree,
		HighlightCritical: false,
		HighlightCircular: true,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should highlight circular nodes
	assert.Contains(t, output, "[CIRCULAR]")
}

func TestVisualizationMaxDepth(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a deep chain
	nodes := []string{"a", "b", "c", "d", "e"}
	for _, node := range nodes {
		graph.AddNode(createTestNode(node, DependencySubquery))
	}

	for i := 0; i < len(nodes)-1; i++ {
		graph.AddDependency(nodes[i], nodes[i+1])
	}

	options := VisualizationOptions{
		Format:   FormatTextTree,
		MaxDepth: 2, // Limit depth
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// Should respect depth limit
	assert.Contains(t, output, "a")
	assert.Contains(t, output, "b")
	// Deeper nodes might not be shown due to depth limit
}

func TestVisualizationDebugInfo(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode(createTestNode("main", DependencyMain))
	graph.AddNode(createTestNode("sub", DependencySubquery))
	graph.AddDependency("main", "sub")

	options := DefaultVisualizationOptions()
	visualizer := NewDependencyVisualizer(graph, options)

	debugInfo := visualizer.GenerateDebugInfo()

	// Should contain comprehensive debug information
	assert.Contains(t, debugInfo, "Dependency Graph Debug Information")
	assert.Contains(t, debugInfo, "Statistics:")
	assert.Contains(t, debugInfo, "Total Nodes: 2")
	assert.Contains(t, debugInfo, "Total Edges: 1")
	assert.Contains(t, debugInfo, "Nodes:")
	assert.Contains(t, debugInfo, "Dependencies:")
	assert.Contains(t, debugInfo, "main -> [sub]")
}

func TestVisualizationFormatString(t *testing.T) {
	tests := []struct {
		format   VisualizationFormat
		expected string
	}{
		{FormatTextTree, "TextTree"},
		{FormatDOT, "DOT"},
		{FormatMermaid, "Mermaid"},
		{FormatJSON, "JSON"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.format.String())
	}
}

func TestVisualizationUnsupportedFormat(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode(createTestNode("test", DependencyMain))

	options := VisualizationOptions{
		Format: VisualizationFormat(999), // Invalid format
	}

	visualizer := NewDependencyVisualizer(graph, options)
	_, err := visualizer.Generate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported visualization format")
}

func TestNodeTypeString(t *testing.T) {
	graph := NewDependencyGraph()
	options := DefaultVisualizationOptions()
	visualizer := NewDependencyVisualizer(graph, options)

	tests := []struct {
		nodeType DependencyType
		expected string
	}{
		{DependencyCTE, "CTE"},
		{DependencySubquery, "Subquery"},
		{DependencyMain, "Main"},
		{DependencyType(999), "Unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, visualizer.nodeTypeString(test.nodeType))
	}
}

func TestFindRootNodes(t *testing.T) {
	graph := NewDependencyGraph()

	// Create graph: root1 -> child1, root2 -> child2 -> grandchild
	graph.AddNode(createTestNode("root1", DependencyMain))
	graph.AddNode(createTestNode("root2", DependencyMain))
	graph.AddNode(createTestNode("child1", DependencySubquery))
	graph.AddNode(createTestNode("child2", DependencySubquery))
	graph.AddNode(createTestNode("grandchild", DependencySubquery))

	graph.AddDependency("root1", "child1")
	graph.AddDependency("root2", "child2")
	graph.AddDependency("child2", "grandchild")

	options := DefaultVisualizationOptions()
	visualizer := NewDependencyVisualizer(graph, options)

	rootNodes := visualizer.findRootNodes()

	// Should find the two root nodes
	assert.Equal(t, 2, len(rootNodes))

	// Check if both root nodes are present
	foundRoot1 := false
	foundRoot2 := false
	for _, root := range rootNodes {
		if root == "root1" {
			foundRoot1 = true
		}
		if root == "root2" {
			foundRoot2 = true
		}
	}
	assert.True(t, foundRoot1)
	assert.True(t, foundRoot2)
}

func TestVisualizationCompactMode(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencyCTE))
	graph.AddDependency("a", "b")

	options := VisualizationOptions{
		Format:          FormatTextTree,
		ShowNodeDetails: false,
		CompactMode:     true,
	}

	visualizer := NewDependencyVisualizer(graph, options)
	output, err := visualizer.Generate()
	assert.NoError(t, err)

	// In compact mode, should not show node type details
	assert.NotContains(t, output, "(Main)")
	assert.NotContains(t, output, "(CTE)")
}

func TestDefaultVisualizationOptions(t *testing.T) {
	options := DefaultVisualizationOptions()

	assert.Equal(t, FormatTextTree, options.Format)
	assert.True(t, options.ShowNodeDetails)
	assert.False(t, options.ShowFieldSources)
	assert.True(t, options.HighlightCritical)
	assert.True(t, options.HighlightCircular)
	assert.Equal(t, 0, len(options.FilterNodeTypes)) // Should be empty slice
	assert.Equal(t, 0, options.MaxDepth)
	assert.False(t, options.CompactMode)
}
