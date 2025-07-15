package parserstep7

import (
	"fmt"
	"sync"
	"time"
)

// Performance-related errors
var (
	ErrTooManyNodes = fmt.Errorf("too many nodes in dependency graph")
	ErrTimeout      = fmt.Errorf("processing timeout exceeded")
)

// PerformanceConfig contains configuration for performance optimization
type PerformanceConfig struct {
	MaxNodes                 int           // Maximum number of nodes to process
	MaxDepth                 int           // Maximum recursion depth
	TimeoutDuration          time.Duration // Processing timeout
	EnableParallelProcessing bool          // Enable parallel processing
	CacheEnabled             bool          // Enable result caching
	BatchSize                int           // Batch size for processing
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		MaxNodes:                 10000,
		MaxDepth:                 100,
		TimeoutDuration:          30 * time.Second,
		EnableParallelProcessing: true,
		CacheEnabled:             true,
		BatchSize:                100,
	}
}

// PerformanceOptimizer provides performance optimization features
type PerformanceOptimizer struct {
	config PerformanceConfig
	cache  *ResultCache
	stats  *PerformanceStats
	mu     sync.RWMutex
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config PerformanceConfig) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		config: config,
		cache:  NewResultCache(config.CacheEnabled),
		stats:  NewPerformanceStats(),
	}
}

// OptimizedDependencyGraph is a performance-optimized version of DependencyGraph
type OptimizedDependencyGraph struct {
	*DependencyGraph
	optimizer *PerformanceOptimizer
	indexed   bool
	levelMap  map[string]int // Pre-computed level mapping
	fanOutMap map[string]int // Pre-computed fan-out mapping
	fanInMap  map[string]int // Pre-computed fan-in mapping
}

// NewOptimizedDependencyGraph creates an optimized dependency graph
func NewOptimizedDependencyGraph(optimizer *PerformanceOptimizer) *OptimizedDependencyGraph {
	return &OptimizedDependencyGraph{
		DependencyGraph: NewDependencyGraph(),
		optimizer:       optimizer,
		indexed:         false,
		levelMap:        make(map[string]int),
		fanOutMap:       make(map[string]int),
		fanInMap:        make(map[string]int),
	}
}

// BuildIndexes pre-computes indexes for faster access
func (odg *OptimizedDependencyGraph) BuildIndexes() {
	if odg.indexed {
		return
	}

	odg.optimizer.mu.Lock()
	defer odg.optimizer.mu.Unlock()

	start := time.Now()

	// Build level map
	odg.buildLevelIndex()

	// Build fan-out/fan-in maps
	odg.buildFanMaps()

	odg.indexed = true
	odg.optimizer.stats.RecordIndexBuildTime(time.Since(start))
}

// buildLevelIndex pre-computes dependency levels
func (odg *OptimizedDependencyGraph) buildLevelIndex() {
	analyzer := NewDependencyAnalyzer(odg.DependencyGraph)
	odg.levelMap = analyzer.calculateDependencyLevels()
}

// buildFanMaps pre-computes fan-out and fan-in values
func (odg *OptimizedDependencyGraph) buildFanMaps() {
	// Calculate fan-out (number of dependencies)
	for nodeID, deps := range odg.edges {
		odg.fanOutMap[nodeID] = len(deps)
	}

	// Calculate fan-in (number of dependents)
	for nodeID := range odg.nodes {
		odg.fanInMap[nodeID] = 0
	}
	for _, deps := range odg.edges {
		for _, dep := range deps {
			odg.fanInMap[dep]++
		}
	}
}

// GetLevel returns the pre-computed level for a node
func (odg *OptimizedDependencyGraph) GetLevel(nodeID string) (int, bool) {
	if !odg.indexed {
		odg.BuildIndexes()
	}
	level, exists := odg.levelMap[nodeID]
	return level, exists
}

// GetFanOut returns the pre-computed fan-out for a node
func (odg *OptimizedDependencyGraph) GetFanOut(nodeID string) (int, bool) {
	if !odg.indexed {
		odg.BuildIndexes()
	}
	fanOut, exists := odg.fanOutMap[nodeID]
	return fanOut, exists
}

// GetFanIn returns the pre-computed fan-in for a node
func (odg *OptimizedDependencyGraph) GetFanIn(nodeID string) (int, bool) {
	if !odg.indexed {
		odg.BuildIndexes()
	}
	fanIn, exists := odg.fanInMap[nodeID]
	return fanIn, exists
}

// GetProcessingOrderOptimized returns an optimized processing order
func (odg *OptimizedDependencyGraph) GetProcessingOrderOptimized() ([]string, error) {
	// Check cache first
	if order, found := odg.optimizer.cache.GetProcessingOrder(odg.GetCacheKey()); found {
		odg.optimizer.stats.RecordCacheHit()
		return order, nil
	}

	odg.optimizer.stats.RecordCacheMiss()

	start := time.Now()
	defer func() {
		odg.optimizer.stats.RecordProcessingTime(time.Since(start))
	}()

	// Check size limits
	if len(odg.nodes) > odg.optimizer.config.MaxNodes {
		return nil, ErrTooManyNodes
	}

	// Use optimized algorithm based on graph size
	var order []string
	var err error

	if len(odg.nodes) < odg.optimizer.config.BatchSize {
		// Small graph: use standard algorithm
		order, err = odg.DependencyGraph.GetProcessingOrder()
	} else {
		// Large graph: use optimized batch processing
		order, err = odg.getBatchProcessingOrder()
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	odg.optimizer.cache.SetProcessingOrder(odg.GetCacheKey(), order)

	return order, nil
}

// getBatchProcessingOrder implements batch-based processing for large graphs
func (odg *OptimizedDependencyGraph) getBatchProcessingOrder() ([]string, error) {
	if !odg.indexed {
		odg.BuildIndexes()
	}

	// Group nodes by level for batch processing
	levelGroups := make(map[int][]string)
	maxLevel := 0

	for nodeID, level := range odg.levelMap {
		if level > maxLevel {
			maxLevel = level
		}
		levelGroups[level] = append(levelGroups[level], nodeID)
	}

	var result []string

	// Process level by level
	for level := 0; level <= maxLevel; level++ {
		nodes := levelGroups[level]

		if odg.optimizer.config.EnableParallelProcessing && len(nodes) > 10 {
			// Process large levels in parallel
			sortedNodes := odg.parallelSortNodes(nodes)
			result = append(result, sortedNodes...)
		} else {
			// Process small levels sequentially
			result = append(result, nodes...)
		}
	}

	return result, nil
}

// parallelSortNodes sorts nodes in parallel for large batches
func (odg *OptimizedDependencyGraph) parallelSortNodes(nodes []string) []string {
	// This is a simplified parallel sorting
	// In a real implementation, you might use worker pools

	// For now, just return the nodes as-is
	// TODO: Implement actual parallel sorting based on priority criteria
	return nodes
}

// GetCacheKey generates a cache key for the current graph state
func (odg *OptimizedDependencyGraph) GetCacheKey() string {
	// Simple hash based on node count and edge count
	// In a real implementation, you'd use a proper hash function
	nodeCount := len(odg.nodes)
	edgeCount := 0
	for _, deps := range odg.edges {
		edgeCount += len(deps)
	}
	return fmt.Sprintf("graph_%d_%d", nodeCount, edgeCount)
}

// ResultCache provides caching for computation results
type ResultCache struct {
	enabled          bool
	processingOrders map[string][]string
	analysisResults  map[string]*DependencyAnalysisResult
	mu               sync.RWMutex
	maxSize          int
	accessCount      map[string]int
}

// NewResultCache creates a new result cache
func NewResultCache(enabled bool) *ResultCache {
	return &ResultCache{
		enabled:          enabled,
		processingOrders: make(map[string][]string),
		analysisResults:  make(map[string]*DependencyAnalysisResult),
		maxSize:          1000,
		accessCount:      make(map[string]int),
	}
}

// GetProcessingOrder retrieves cached processing order
func (rc *ResultCache) GetProcessingOrder(key string) ([]string, bool) {
	if !rc.enabled {
		return nil, false
	}

	rc.mu.RLock()
	defer rc.mu.RUnlock()

	order, exists := rc.processingOrders[key]
	if exists {
		rc.accessCount[key]++
	}
	return order, exists
}

// SetProcessingOrder caches a processing order
func (rc *ResultCache) SetProcessingOrder(key string, order []string) {
	if !rc.enabled {
		return
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Check cache size and evict if necessary
	if len(rc.processingOrders) >= rc.maxSize {
		rc.evictLeastUsed()
	}

	rc.processingOrders[key] = order
	rc.accessCount[key] = 1
}

// evictLeastUsed removes the least used cache entry
func (rc *ResultCache) evictLeastUsed() {
	minAccess := int(^uint(0) >> 1) // Max int
	var keyToEvict string

	for key, count := range rc.accessCount {
		if count < minAccess {
			minAccess = count
			keyToEvict = key
		}
	}

	if keyToEvict != "" {
		delete(rc.processingOrders, keyToEvict)
		delete(rc.analysisResults, keyToEvict)
		delete(rc.accessCount, keyToEvict)
	}
}

// PerformanceStats tracks performance metrics
type PerformanceStats struct {
	mu                  sync.RWMutex
	IndexBuildTime      time.Duration
	TotalProcessingTime time.Duration
	CacheHits           int
	CacheMisses         int
	ProcessingCount     int
}

// NewPerformanceStats creates new performance statistics
func NewPerformanceStats() *PerformanceStats {
	return &PerformanceStats{}
}

// RecordIndexBuildTime records time spent building indexes
func (ps *PerformanceStats) RecordIndexBuildTime(duration time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.IndexBuildTime += duration
}

// RecordProcessingTime records processing time
func (ps *PerformanceStats) RecordProcessingTime(duration time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.TotalProcessingTime += duration
	ps.ProcessingCount++
}

// RecordCacheHit records a cache hit
func (ps *PerformanceStats) RecordCacheHit() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.CacheHits++
}

// RecordCacheMiss records a cache miss
func (ps *PerformanceStats) RecordCacheMiss() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.CacheMisses++
}

// GetStats returns current performance statistics
func (ps *PerformanceStats) GetStats() PerformanceMetrics {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	avgProcessingTime := time.Duration(0)
	if ps.ProcessingCount > 0 {
		avgProcessingTime = ps.TotalProcessingTime / time.Duration(ps.ProcessingCount)
	}

	cacheHitRate := 0.0
	totalCacheAccess := ps.CacheHits + ps.CacheMisses
	if totalCacheAccess > 0 {
		cacheHitRate = float64(ps.CacheHits) / float64(totalCacheAccess) * 100
	}

	return PerformanceMetrics{
		IndexBuildTime:        ps.IndexBuildTime,
		TotalProcessingTime:   ps.TotalProcessingTime,
		AverageProcessingTime: avgProcessingTime,
		ProcessingCount:       ps.ProcessingCount,
		CacheHits:             ps.CacheHits,
		CacheMisses:           ps.CacheMisses,
		CacheHitRate:          cacheHitRate,
	}
}

// PerformanceMetrics contains performance metric values
type PerformanceMetrics struct {
	IndexBuildTime        time.Duration
	TotalProcessingTime   time.Duration
	AverageProcessingTime time.Duration
	ProcessingCount       int
	CacheHits             int
	CacheMisses           int
	CacheHitRate          float64 // Percentage
}

// String returns a string representation of performance metrics
func (pm PerformanceMetrics) String() string {
	return fmt.Sprintf(
		"Performance Metrics:\n"+
			"  Index Build Time: %v\n"+
			"  Total Processing Time: %v\n"+
			"  Average Processing Time: %v\n"+
			"  Processing Count: %d\n"+
			"  Cache Hits: %d\n"+
			"  Cache Misses: %d\n"+
			"  Cache Hit Rate: %.2f%%",
		pm.IndexBuildTime,
		pm.TotalProcessingTime,
		pm.AverageProcessingTime,
		pm.ProcessingCount,
		pm.CacheHits,
		pm.CacheMisses,
		pm.CacheHitRate,
	)
}
