package parserstep7

import (
	"fmt"
	"sort"
	"strings"
)

// VisualizationFormat represents different output formats for dependency visualization
type VisualizationFormat int

const (
	FormatTextTree VisualizationFormat = iota
	FormatDOT                          // Graphviz DOT format
	FormatMermaid                      // Mermaid diagram format
	FormatJSON                         // JSON format for programmatic use
)

// String returns the string representation of VisualizationFormat
func (vf VisualizationFormat) String() string {
	switch vf {
	case FormatTextTree:
		return "TextTree"
	case FormatDOT:
		return "DOT"
	case FormatMermaid:
		return "Mermaid"
	case FormatJSON:
		return "JSON"
	default:
		return "Unknown"
	}
}

// VisualizationOptions controls visualization output
type VisualizationOptions struct {
	Format            VisualizationFormat
	ShowNodeDetails   bool     // Include node metadata
	ShowFieldSources  bool     // Include field source information
	HighlightCritical bool     // Highlight critical nodes
	HighlightCircular bool     // Highlight circular dependencies
	FilterNodeTypes   []string // Only show specific node types
	MaxDepth          int      // Maximum depth to visualize (0 = unlimited)
	CompactMode       bool     // Use compact representation
}

// DefaultVisualizationOptions returns default visualization options
func DefaultVisualizationOptions() VisualizationOptions {
	return VisualizationOptions{
		Format:            FormatTextTree,
		ShowNodeDetails:   true,
		ShowFieldSources:  false,
		HighlightCritical: true,
		HighlightCircular: true,
		FilterNodeTypes:   nil,
		MaxDepth:          0,
		CompactMode:       false,
	}
}

// DependencyVisualizer generates visual representations of dependency graphs
type DependencyVisualizer struct {
	graph    *DependencyGraph
	analyzer *DependencyAnalyzer
	options  VisualizationOptions
}

// NewDependencyVisualizer creates a new dependency visualizer
func NewDependencyVisualizer(graph *DependencyGraph, options VisualizationOptions) *DependencyVisualizer {
	return &DependencyVisualizer{
		graph:    graph,
		analyzer: NewDependencyAnalyzer(graph),
		options:  options,
	}
}

// nodeTypeString returns string representation of DependencyType
func (dv *DependencyVisualizer) nodeTypeString(nodeType DependencyType) string {
	switch nodeType {
	case DependencyCTE:
		return "CTE"
	case DependencySubquery:
		return "Subquery"
	case DependencyMain:
		return "Main"
	default:
		return "Unknown"
	}
}

// Generate creates a visual representation of the dependency graph
func (dv *DependencyVisualizer) Generate() (string, error) {
	switch dv.options.Format {
	case FormatTextTree:
		return dv.generateTextTree(), nil
	case FormatDOT:
		return dv.generateDOT(), nil
	case FormatMermaid:
		return dv.generateMermaid(), nil
	case FormatJSON:
		return dv.generateJSON(), nil
	default:
		return "", fmt.Errorf("unsupported visualization format: %v", dv.options.Format)
	}
}

// generateTextTree creates a text-based tree representation
func (dv *DependencyVisualizer) generateTextTree() string {
	var sb strings.Builder

	// Get analysis results for highlighting
	analysis := dv.analyzer.AnalyzeComplexDependencies()
	criticalNodes := make(map[string]bool)
	for _, node := range analysis.CriticalNodes {
		criticalNodes[node] = true
	}

	circularNodes := make(map[string]bool)
	for _, path := range analysis.CircularPaths {
		for _, node := range path {
			circularNodes[node] = true
		}
	}

	// Find root nodes (nodes with no dependencies)
	rootNodes := dv.findRootNodes()
	if len(rootNodes) == 0 {
		sb.WriteString("No root nodes found (possible circular dependencies)\n")
		// Show all nodes when no clear roots exist
		for nodeID := range dv.graph.nodes {
			rootNodes = append(rootNodes, nodeID)
		}
	}

	// Sort root nodes for consistent output
	sort.Strings(rootNodes)

	visited := make(map[string]bool)
	for _, rootID := range rootNodes {
		if !visited[rootID] {
			dv.renderNodeTree(rootID, "", true, visited, criticalNodes, circularNodes, &sb, 0)
		}
	}

	// Add summary
	if dv.options.ShowNodeDetails {
		sb.WriteString("\n--- Summary ---\n")
		sb.WriteString(fmt.Sprintf("Total Nodes: %d\n", analysis.Stats.TotalNodes))
		sb.WriteString(fmt.Sprintf("Total Edges: %d\n", analysis.Stats.TotalEdges))
		sb.WriteString(fmt.Sprintf("Max Depth: %d\n", analysis.Stats.MaxDepth))

		if len(analysis.CircularPaths) > 0 {
			sb.WriteString(fmt.Sprintf("Circular Dependencies: %d\n", len(analysis.CircularPaths)))
		}

		if len(analysis.CriticalNodes) > 0 {
			sb.WriteString(fmt.Sprintf("Critical Nodes: %s\n", strings.Join(analysis.CriticalNodes, ", ")))
		}
	}

	return sb.String()
}

// renderNodeTree recursively renders a node and its dependencies as a tree
func (dv *DependencyVisualizer) renderNodeTree(nodeID string, prefix string, isLast bool, visited map[string]bool, criticalNodes, circularNodes map[string]bool, sb *strings.Builder, depth int) {
	// Check depth limit
	if dv.options.MaxDepth > 0 && depth >= dv.options.MaxDepth {
		return
	}

	// Prevent infinite recursion in case of cycles
	if visited[nodeID] {
		sb.WriteString(fmt.Sprintf("%s↻ %s (already shown)\n", prefix, nodeID))
		return
	}
	visited[nodeID] = true
	defer func() { visited[nodeID] = false }()

	// Create node representation
	nodeSymbol := "├── "
	if isLast {
		nodeSymbol = "└── "
	}

	nodeRepr := nodeID

	// Add highlighting markers
	if dv.options.HighlightCritical && criticalNodes[nodeID] {
		nodeRepr = fmt.Sprintf("[CRITICAL] %s", nodeRepr)
	}
	if dv.options.HighlightCircular && circularNodes[nodeID] {
		nodeRepr = fmt.Sprintf("[CIRCULAR] %s", nodeRepr)
	} // Add node details if requested
	if dv.options.ShowNodeDetails {
		if node, exists := dv.graph.nodes[nodeID]; exists {
			nodeRepr = fmt.Sprintf("%s (%s)", nodeRepr, dv.nodeTypeString(node.NodeType))
		}
	}

	sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, nodeSymbol, nodeRepr))

	// Render children
	children := dv.graph.edges[nodeID]
	if len(children) == 0 {
		return
	}

	// Sort children for consistent output
	sortedChildren := make([]string, len(children))
	copy(sortedChildren, children)
	sort.Strings(sortedChildren)

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, childID := range sortedChildren {
		isLastChild := i == len(sortedChildren)-1
		dv.renderNodeTree(childID, childPrefix, isLastChild, visited, criticalNodes, circularNodes, sb, depth+1)
	}
}

// generateDOT creates a Graphviz DOT representation
func (dv *DependencyVisualizer) generateDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph DependencyGraph {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n")

	// Get analysis results for styling
	analysis := dv.analyzer.AnalyzeComplexDependencies()
	criticalNodes := make(map[string]bool)
	for _, node := range analysis.CriticalNodes {
		criticalNodes[node] = true
	}

	circularNodes := make(map[string]bool)
	for _, path := range analysis.CircularPaths {
		for _, node := range path {
			circularNodes[node] = true
		}
	}

	// Add nodes with styling
	for nodeID, node := range dv.graph.nodes {
		style := ""
		if dv.options.HighlightCritical && criticalNodes[nodeID] {
			style += "fillcolor=lightcoral, style=\"filled,rounded\", "
		}
		if dv.options.HighlightCircular && circularNodes[nodeID] {
			style += "color=red, penwidth=2, "
		}

		label := nodeID
		if dv.options.ShowNodeDetails {
			label = fmt.Sprintf("%s\\n(%s)", nodeID, dv.nodeTypeString(node.NodeType))
		}

		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", %s];\n", nodeID, label, style))
	}

	// Add edges
	for fromID, toIDs := range dv.graph.edges {
		for _, toID := range toIDs {
			edgeStyle := ""
			// Highlight circular edges
			if dv.options.HighlightCircular && circularNodes[fromID] && circularNodes[toID] {
				edgeStyle = " [color=red, penwidth=2]"
			}
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\"%s;\n", fromID, toID, edgeStyle))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// generateMermaid creates a Mermaid diagram representation
func (dv *DependencyVisualizer) generateMermaid() string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// Get analysis results for styling
	analysis := dv.analyzer.AnalyzeComplexDependencies()
	criticalNodes := make(map[string]bool)
	for _, node := range analysis.CriticalNodes {
		criticalNodes[node] = true
	}

	circularNodes := make(map[string]bool)
	for _, path := range analysis.CircularPaths {
		for _, node := range path {
			circularNodes[node] = true
		}
	}

	// Add nodes with styling
	for nodeID, node := range dv.graph.nodes {
		nodeLabel := nodeID
		if dv.options.ShowNodeDetails {
			nodeLabel = fmt.Sprintf("%s<br/>(%s)", nodeID, dv.nodeTypeString(node.NodeType))
		}

		// Mermaid uses different syntax for node shapes and styling
		shape := fmt.Sprintf("%s[\"%s\"]", nodeID, nodeLabel)
		sb.WriteString(fmt.Sprintf("  %s\n", shape))

		// Add styling classes
		if dv.options.HighlightCritical && criticalNodes[nodeID] {
			sb.WriteString(fmt.Sprintf("  class %s critical\n", nodeID))
		}
		if dv.options.HighlightCircular && circularNodes[nodeID] {
			sb.WriteString(fmt.Sprintf("  class %s circular\n", nodeID))
		}
	}

	// Add edges
	for fromID, toIDs := range dv.graph.edges {
		for _, toID := range toIDs {
			sb.WriteString(fmt.Sprintf("  %s --> %s\n", fromID, toID))
		}
	}

	// Add styling definitions
	if dv.options.HighlightCritical || dv.options.HighlightCircular {
		sb.WriteString("\n  classDef critical fill:#ffcccc,stroke:#ff0000,stroke-width:2px\n")
		sb.WriteString("  classDef circular fill:#ccccff,stroke:#0000ff,stroke-width:2px\n")
	}

	return sb.String()
}

// generateJSON creates a JSON representation
func (dv *DependencyVisualizer) generateJSON() string {
	// This is a simplified JSON representation
	// In a real implementation, you'd use proper JSON marshaling
	var sb strings.Builder

	sb.WriteString("{\n")
	sb.WriteString("  \"nodes\": [\n")

	nodeIDs := make([]string, 0, len(dv.graph.nodes))
	for nodeID := range dv.graph.nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	for i, nodeID := range nodeIDs {
		node := dv.graph.nodes[nodeID]
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("    {\"id\": \"%s\", \"type\": \"%s\"}", nodeID, dv.nodeTypeString(node.NodeType)))
	}

	sb.WriteString("\n  ],\n")
	sb.WriteString("  \"edges\": [\n")

	edgeCount := 0
	for fromID, toIDs := range dv.graph.edges {
		for _, toID := range toIDs {
			if edgeCount > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf("    {\"from\": \"%s\", \"to\": \"%s\"}", fromID, toID))
			edgeCount++
		}
	}

	sb.WriteString("\n  ]\n")
	sb.WriteString("}\n")

	return sb.String()
}

// findRootNodes finds nodes that have no incoming dependencies
func (dv *DependencyVisualizer) findRootNodes() []string {
	hasIncoming := make(map[string]bool)

	// Mark all nodes that have incoming edges
	for _, toIDs := range dv.graph.edges {
		for _, toID := range toIDs {
			hasIncoming[toID] = true
		}
	}

	// Find nodes without incoming edges
	var rootNodes []string
	for nodeID := range dv.graph.nodes {
		if !hasIncoming[nodeID] {
			rootNodes = append(rootNodes, nodeID)
		}
	}

	return rootNodes
}

// GenerateDebugInfo creates comprehensive debug information
func (dv *DependencyVisualizer) GenerateDebugInfo() string {
	var sb strings.Builder

	analysis := dv.analyzer.AnalyzeComplexDependencies()

	sb.WriteString("=== Dependency Graph Debug Information ===\n\n")

	// Basic statistics
	sb.WriteString("Statistics:\n")
	sb.WriteString(fmt.Sprintf("  Total Nodes: %d\n", analysis.Stats.TotalNodes))
	sb.WriteString(fmt.Sprintf("  Total Edges: %d\n", analysis.Stats.TotalEdges))
	sb.WriteString(fmt.Sprintf("  Max Depth: %d\n", analysis.Stats.MaxDepth))
	sb.WriteString(fmt.Sprintf("  Average Fan-out: %.2f\n", analysis.Stats.AverageFanOut))
	sb.WriteString(fmt.Sprintf("  Average Fan-in: %.2f\n", analysis.Stats.AverageFanIn))
	sb.WriteString(fmt.Sprintf("  Circular Dependencies: %d\n\n", analysis.Stats.CircularCount))

	// Node details
	sb.WriteString("Nodes:\n")
	for nodeID, node := range dv.graph.nodes {
		dependencies := len(dv.graph.edges[nodeID])
		level := analysis.Levels[nodeID]
		sb.WriteString(fmt.Sprintf("  %s: type=%s, dependencies=%d, level=%d\n", nodeID, dv.nodeTypeString(node.NodeType), dependencies, level))
	}
	sb.WriteString("\n")

	// Dependency details
	sb.WriteString("Dependencies:\n")
	for fromID, toIDs := range dv.graph.edges {
		sb.WriteString(fmt.Sprintf("  %s -> [%s]\n", fromID, strings.Join(toIDs, ", ")))
	}
	sb.WriteString("\n")

	// Circular paths
	if len(analysis.CircularPaths) > 0 {
		sb.WriteString("Circular Paths:\n")
		for i, path := range analysis.CircularPaths {
			sb.WriteString(fmt.Sprintf("  %d: %s\n", i+1, strings.Join(path, " -> ")))
		}
		sb.WriteString("\n")
	}

	// Critical nodes
	if len(analysis.CriticalNodes) > 0 {
		sb.WriteString("Critical Nodes:\n")
		for _, nodeID := range analysis.CriticalNodes {
			sb.WriteString(fmt.Sprintf("  %s\n", nodeID))
		}
		sb.WriteString("\n")
	}

	// Longest chains
	if len(analysis.DependencyChains) > 0 {
		sb.WriteString("Longest Dependency Chains:\n")
		for i, chain := range analysis.DependencyChains {
			sb.WriteString(fmt.Sprintf("  %d: %s\n", i+1, strings.Join(chain, " -> ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
