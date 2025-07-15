package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestDependencyGraph_AddNode(t *testing.T) {
	dg := NewDependencyGraph()

	node := &DependencyNode{
		ID:       "test_1",
		NodeType: DependencyMain,
	}

	err := dg.AddNode(node)
	assert.NoError(t, err)
	assert.Equal(t, node, dg.GetNode("test_1"))
}

func TestDependencyGraph_AddDependency(t *testing.T) {
	dg := NewDependencyGraph()

	node1 := &DependencyNode{ID: "node1", NodeType: DependencyMain}
	node2 := &DependencyNode{ID: "node2", NodeType: DependencyCTE}

	dg.AddNode(node1)
	dg.AddNode(node2)

	err := dg.AddDependency("node1", "node2")
	assert.NoError(t, err)
}

func TestDependencyGraph_GetProcessingOrder_NoDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	node1 := &DependencyNode{ID: "node1", NodeType: DependencyMain}
	node2 := &DependencyNode{ID: "node2", NodeType: DependencyCTE}

	dg.AddNode(node1)
	dg.AddNode(node2)

	order, err := dg.GetProcessingOrder()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(order))

	// Check that both nodes are in the order
	found1, found2 := false, false
	for _, nodeID := range order {
		if nodeID == "node1" {
			found1 = true
		}
		if nodeID == "node2" {
			found2 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)
}

func TestDependencyGraph_GetProcessingOrder_WithDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	node1 := &DependencyNode{ID: "main", NodeType: DependencyMain}
	node2 := &DependencyNode{ID: "cte", NodeType: DependencyCTE}

	dg.AddNode(node1)
	dg.AddNode(node2)
	dg.AddDependency("cte", "main") // cte points to main (main depends on cte)

	order, err := dg.GetProcessingOrder()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(order))

	// cte should come before main (dependencies processed first)
	cteIndex := -1
	mainIndex := -1
	for i, nodeID := range order {
		if nodeID == "cte" {
			cteIndex = i
		}
		if nodeID == "main" {
			mainIndex = i
		}
	}

	assert.True(t, cteIndex >= 0, "cte should be in processing order")
	assert.True(t, mainIndex >= 0, "main should be in processing order")
	assert.True(t, cteIndex < mainIndex, "cte should be processed before main")
}

func TestDependencyGraph_CircularDependency(t *testing.T) {
	dg := NewDependencyGraph()

	node1 := &DependencyNode{ID: "node1", NodeType: DependencyMain}
	node2 := &DependencyNode{ID: "node2", NodeType: DependencyCTE}

	dg.AddNode(node1)
	dg.AddNode(node2)
	dg.AddDependency("node1", "node2")
	dg.AddDependency("node2", "node1") // Creates circular dependency

	assert.True(t, dg.HasCircularDependency())

	_, err := dg.GetProcessingOrder()
	assert.Error(t, err)
	assert.Equal(t, ErrCircularDependency, err)
}
