package parserstep7

import (
	"fmt"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// DependencyAnalyzer provides advanced dependency analysis features
type DependencyAnalyzer struct {
	graph *cmn.SQDependencyGraph
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(graph *cmn.SQDependencyGraph) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		graph: graph,
	}
}

// AnalyzeComplexDependencies analyzes complex dependency patterns
func (da *DependencyAnalyzer) AnalyzeComplexDependencies() *DependencyAnalysisResult {
	result := &DependencyAnalysisResult{
		Levels:           da.calculateDependencyLevels(),
		CircularPaths:    da.findAllCircularPaths(),
		CriticalNodes:    da.findCriticalNodes(),
		DependencyChains: da.findLongestDependencyChains(),
		Stats:            da.calculateStatistics(),
	}
	return result
}

// DependencyAnalysisResult contains the results of dependency analysis
type DependencyAnalysisResult struct {
	Levels           map[string]int  // Node ID to dependency level
	CircularPaths    [][]string      // All circular dependency paths
	CriticalNodes    []string        // Nodes that many others depend on
	DependencyChains [][]string      // Longest dependency chains
	Stats            DependencyStats // Overall statistics
}

// DependencyStats contains dependency statistics
type DependencyStats struct {
	TotalNodes    int     // Total number of nodes
	TotalEdges    int     // Total number of edges
	MaxDepth      int     // Maximum dependency depth
	AverageFanOut float64 // Average number of dependencies per node
	AverageFanIn  float64 // Average number of dependents per node
	CircularCount int     // Number of circular dependencies
}

// calculateDependencyLevels calculates the dependency level for each node
func (da *DependencyAnalyzer) calculateDependencyLevels() map[string]int {
	levels := make(map[string]int)
	visited := make(map[string]bool)

	// Initialize all nodes to level 0
	for nodeID := range da.graph.GetAllNodes() {
		levels[nodeID] = 0
	}

	// Calculate levels using DFS
	for nodeID := range da.graph.GetAllNodes() {
		if !visited[nodeID] {
			da.calculateLevelDFS(nodeID, levels, visited, make(map[string]bool))
		}
	}

	return levels
}

// calculateLevelDFS performs DFS to calculate dependency levels
func (da *DependencyAnalyzer) calculateLevelDFS(nodeID string, levels map[string]int, visited, inStack map[string]bool) {
	if visited[nodeID] {
		return
	}

	visited[nodeID] = true
	inStack[nodeID] = true

	maxChildLevel := -1
	nodes := da.graph.GetAllNodes()
	node := nodes[nodeID]
	if node != nil {
		for _, childID := range node.Dependencies {
			if inStack[childID] {
				// Circular dependency detected, skip to avoid infinite recursion
				continue
			}

			da.calculateLevelDFS(childID, levels, visited, inStack)
			if levels[childID] > maxChildLevel {
				maxChildLevel = levels[childID]
			}
		}
	}

	if maxChildLevel >= 0 {
		levels[nodeID] = maxChildLevel + 1
	}

	inStack[nodeID] = false
}

// findAllCircularPaths finds all circular dependency paths
func (da *DependencyAnalyzer) findAllCircularPaths() [][]string {
	var circularPaths [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	for nodeID := range da.graph.GetAllNodes() {
		if !visited[nodeID] {
			path := []string{}
			da.findCircularPathsDFS(nodeID, visited, inStack, path, &circularPaths)
		}
	}

	return circularPaths
}

// findCircularPathsDFS performs DFS to find circular paths
func (da *DependencyAnalyzer) findCircularPathsDFS(nodeID string, visited, inStack map[string]bool, path []string, circularPaths *[][]string) {
	visited[nodeID] = true
	inStack[nodeID] = true
	path = append(path, nodeID)

	nodes := da.graph.GetAllNodes()
	node := nodes[nodeID]
	if node != nil {
		for _, childID := range node.Dependencies {
			if inStack[childID] {
				// Found a circular path
				circularStart := -1
				for i, id := range path {
					if id == childID {
						circularStart = i
						break
					}
				}
				if circularStart >= 0 {
					circularPath := make([]string, len(path)-circularStart)
					copy(circularPath, path[circularStart:])
					*circularPaths = append(*circularPaths, circularPath)
				}
			} else if !visited[childID] {
				da.findCircularPathsDFS(childID, visited, inStack, path, circularPaths)
			}
		}

		inStack[nodeID] = false
	}
}

// findCriticalNodes finds nodes that many others depend on
func (da *DependencyAnalyzer) findCriticalNodes() []string {
	inDegree := make(map[string]int)

	// Calculate in-degrees
	nodes := da.graph.GetAllNodes()
	for nodeID := range nodes {
		inDegree[nodeID] = 0
	}
	for _, node := range nodes {
		for _, dep := range node.Dependencies {
			inDegree[dep]++
		}
	}

	// Find nodes with high in-degree (many dependents)
	var criticalNodes []string
	threshold := len(nodes) / 4 // 25% threshold
	if threshold < 2 {
		threshold = 2
	}

	for nodeID, degree := range inDegree {
		if degree >= threshold {
			criticalNodes = append(criticalNodes, nodeID)
		}
	}

	return criticalNodes
}

// findLongestDependencyChains finds the longest dependency chains
func (da *DependencyAnalyzer) findLongestDependencyChains() [][]string {
	var longestChains [][]string
	maxLength := 0

	nodes := da.graph.GetAllNodes()
	for nodeID := range nodes {
		path := []string{nodeID}
		visited := make(map[string]bool)
		chains := da.findChainsFromNode(nodeID, path, visited)

		for _, chain := range chains {
			if len(chain) > maxLength {
				maxLength = len(chain)
				longestChains = [][]string{chain}
			} else if len(chain) == maxLength {
				longestChains = append(longestChains, chain)
			}
		}
	}

	return longestChains
}

// findChainsFromNode finds all dependency chains starting from a node
func (da *DependencyAnalyzer) findChainsFromNode(nodeID string, path []string, visited map[string]bool) [][]string {
	if visited[nodeID] {
		return [][]string{path}
	}

	visited[nodeID] = true
	defer func() { visited[nodeID] = false }()

	nodes := da.graph.GetAllNodes()
	node := nodes[nodeID]
	var children []string
	if node != nil {
		children = node.Dependencies
	}

	if len(children) == 0 {
		// Leaf node, return current path
		result := make([]string, len(path))
		copy(result, path)
		return [][]string{result}
	}

	var allChains [][]string
	for _, childID := range children {
		childPath := append(path, childID)
		childChains := da.findChainsFromNode(childID, childPath, visited)
		allChains = append(allChains, childChains...)
	}

	return allChains
}

// calculateStatistics calculates overall dependency statistics
func (da *DependencyAnalyzer) calculateStatistics() DependencyStats {
	nodes := da.graph.GetAllNodes()
	totalNodes := len(nodes)
	totalEdges := 0

	for _, node := range nodes {
		totalEdges += len(node.Dependencies)
	}

	levels := da.calculateDependencyLevels()
	maxDepth := 0
	for _, level := range levels {
		if level > maxDepth {
			maxDepth = level
		}
	}

	averageFanOut := 0.0
	if totalNodes > 0 {
		averageFanOut = float64(totalEdges) / float64(totalNodes)
	}

	inDegree := make(map[string]int)
	for nodeID := range nodes {
		inDegree[nodeID] = 0
	}
	for _, node := range nodes {
		for _, dep := range node.Dependencies {
			inDegree[dep]++
		}
	}

	averageFanIn := 0.0
	if totalNodes > 0 {
		totalInDegree := 0
		for _, degree := range inDegree {
			totalInDegree += degree
		}
		averageFanIn = float64(totalInDegree) / float64(totalNodes)
	}

	circularPaths := da.findAllCircularPaths()
	circularCount := len(circularPaths)

	return DependencyStats{
		TotalNodes:    totalNodes,
		TotalEdges:    totalEdges,
		MaxDepth:      maxDepth,
		AverageFanOut: averageFanOut,
		AverageFanIn:  averageFanIn,
		CircularCount: circularCount,
	}
}

// String returns a string representation of the analysis result
func (dar *DependencyAnalysisResult) String() string {
	var sb strings.Builder

	sb.WriteString("Dependency Analysis Result:\n")
	sb.WriteString(fmt.Sprintf("- Total Nodes: %d\n", dar.Stats.TotalNodes))
	sb.WriteString(fmt.Sprintf("- Total Edges: %d\n", dar.Stats.TotalEdges))
	sb.WriteString(fmt.Sprintf("- Max Depth: %d\n", dar.Stats.MaxDepth))
	sb.WriteString(fmt.Sprintf("- Average Fan-out: %.2f\n", dar.Stats.AverageFanOut))
	sb.WriteString(fmt.Sprintf("- Average Fan-in: %.2f\n", dar.Stats.AverageFanIn))
	sb.WriteString(fmt.Sprintf("- Circular Dependencies: %d\n", dar.Stats.CircularCount))

	if len(dar.CircularPaths) > 0 {
		sb.WriteString("\nCircular Paths:\n")
		for _, path := range dar.CircularPaths {
			sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(path, " -> ")))
		}
	}

	if len(dar.CriticalNodes) > 0 {
		sb.WriteString("\nCritical Nodes:\n")
		for _, node := range dar.CriticalNodes {
			sb.WriteString(fmt.Sprintf("  %s\n", node))
		}
	}

	if len(dar.DependencyChains) > 0 {
		sb.WriteString("\nLongest Dependency Chains:\n")
		for _, chain := range dar.DependencyChains {
			sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(chain, " -> ")))
		}
	}

	return sb.String()
}
