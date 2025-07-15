package parserstep7

import (
	"fmt"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestPerformanceOptimizer(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)

	// Basic checks
	assert.True(t, optimizer != nil)
	assert.True(t, optimizer.cache != nil)
	assert.True(t, optimizer.stats != nil)
}

func TestOptimizedDependencyGraph(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	// Add test nodes
	graph.AddNode(createTestNode("main", DependencyMain))
	graph.AddNode(createTestNode("cte1", DependencyCTE))
	graph.AddNode(createTestNode("sub1", DependencySubquery))

	graph.AddDependency("main", "cte1")
	graph.AddDependency("main", "sub1")

	// Build indexes
	graph.BuildIndexes()
	assert.True(t, graph.indexed)

	// Test level access
	level, exists := graph.GetLevel("main")
	assert.True(t, exists)
	assert.Equal(t, 1, level)

	// Test fan-out access
	fanOut, exists := graph.GetFanOut("main")
	assert.True(t, exists)
	assert.Equal(t, 2, fanOut)

	// Test fan-in access
	fanIn, exists := graph.GetFanIn("cte1")
	assert.True(t, exists)
	assert.Equal(t, 1, fanIn)
}

func TestOptimizedProcessingOrder(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	// Create small graph
	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencySubquery))
	graph.AddDependency("a", "b")

	order, err := graph.GetProcessingOrderOptimized()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(order))

	// Test caching - second call should hit cache
	order2, err := graph.GetProcessingOrderOptimized()
	assert.NoError(t, err)
	assert.Equal(t, order, order2)

	// Verify cache hit was recorded
	stats := optimizer.stats.GetStats()
	assert.True(t, stats.CacheHits > 0)
}

func TestPerformanceConfigMaxNodes(t *testing.T) {
	config := PerformanceConfig{
		MaxNodes: 2, // Set very low limit
	}
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	// Add more nodes than the limit
	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencySubquery))
	graph.AddNode(createTestNode("c", DependencySubquery))

	_, err := graph.GetProcessingOrderOptimized()
	assert.Error(t, err)
	assert.Equal(t, ErrTooManyNodes, err)
}

func TestResultCache(t *testing.T) {
	cache := NewResultCache(true)

	// Test cache miss
	_, found := cache.GetProcessingOrder("key1")
	assert.False(t, found)

	// Test cache set and hit
	order := []string{"a", "b", "c"}
	cache.SetProcessingOrder("key1", order)

	retrieved, found := cache.GetProcessingOrder("key1")
	assert.True(t, found)
	assert.Equal(t, order, retrieved)
}

func TestResultCacheDisabled(t *testing.T) {
	cache := NewResultCache(false) // Disabled cache

	order := []string{"a", "b", "c"}
	cache.SetProcessingOrder("key1", order)

	// Should not find anything when cache is disabled
	_, found := cache.GetProcessingOrder("key1")
	assert.False(t, found)
}

func TestResultCacheEviction(t *testing.T) {
	cache := NewResultCache(true)
	cache.maxSize = 2 // Set small limit for testing

	// Fill cache beyond limit
	cache.SetProcessingOrder("key1", []string{"a"})
	cache.SetProcessingOrder("key2", []string{"b"})
	cache.SetProcessingOrder("key3", []string{"c"}) // Should trigger eviction

	// key1 should be evicted (least used)
	_, found := cache.GetProcessingOrder("key1")
	assert.False(t, found)

	// key2 and key3 should still be there
	_, found = cache.GetProcessingOrder("key2")
	assert.True(t, found)
	_, found = cache.GetProcessingOrder("key3")
	assert.True(t, found)
}

func TestPerformanceStats(t *testing.T) {
	stats := NewPerformanceStats()

	// Record various metrics
	stats.RecordIndexBuildTime(100 * time.Millisecond)
	stats.RecordProcessingTime(50 * time.Millisecond)
	stats.RecordProcessingTime(150 * time.Millisecond)
	stats.RecordCacheHit()
	stats.RecordCacheHit()
	stats.RecordCacheMiss()

	metrics := stats.GetStats()

	assert.Equal(t, 100*time.Millisecond, metrics.IndexBuildTime)
	assert.Equal(t, 200*time.Millisecond, metrics.TotalProcessingTime)
	assert.Equal(t, 100*time.Millisecond, metrics.AverageProcessingTime)
	assert.Equal(t, 2, metrics.ProcessingCount)
	assert.Equal(t, 2, metrics.CacheHits)
	assert.Equal(t, 1, metrics.CacheMisses)
	// Check cache hit rate (2/3 * 100 = 66.67%)
	assert.True(t, metrics.CacheHitRate > 66.0 && metrics.CacheHitRate < 67.0)
}

func TestPerformanceStatsString(t *testing.T) {
	stats := NewPerformanceStats()
	stats.RecordProcessingTime(100 * time.Millisecond)
	stats.RecordCacheHit()

	metrics := stats.GetStats()
	output := metrics.String()

	assert.Contains(t, output, "Performance Metrics:")
	assert.Contains(t, output, "Index Build Time:")
	assert.Contains(t, output, "Total Processing Time:")
	assert.Contains(t, output, "Average Processing Time:")
	assert.Contains(t, output, "Processing Count:")
	assert.Contains(t, output, "Cache Hits:")
	assert.Contains(t, output, "Cache Misses:")
	assert.Contains(t, output, "Cache Hit Rate:")
}

func TestDefaultPerformanceConfig(t *testing.T) {
	config := DefaultPerformanceConfig()

	assert.Equal(t, 10000, config.MaxNodes)
	assert.Equal(t, 100, config.MaxDepth)
	assert.Equal(t, 30*time.Second, config.TimeoutDuration)
	assert.True(t, config.EnableParallelProcessing)
	assert.True(t, config.CacheEnabled)
	assert.Equal(t, 100, config.BatchSize)
}

func TestBatchProcessingOrder(t *testing.T) {
	config := PerformanceConfig{
		MaxNodes:                 20, // Increase limit to accommodate test
		BatchSize:                10,
		EnableParallelProcessing: false, // Disable parallel for predictable testing
	}
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	// Create large enough graph to trigger batch processing
	for i := 0; i < 15; i++ {
		nodeID := fmt.Sprintf("node%d", i)
		graph.AddNode(createTestNode(nodeID, DependencySubquery))
	}

	// Add some dependencies to create levels
	graph.AddDependency("node0", "node1")
	graph.AddDependency("node1", "node2")

	order, err := graph.GetProcessingOrderOptimized()
	assert.NoError(t, err)
	assert.Equal(t, 15, len(order))
}

func TestCacheKeyGeneration(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	graph.AddNode(createTestNode("a", DependencyMain))
	graph.AddNode(createTestNode("b", DependencySubquery))
	graph.AddDependency("a", "b")

	key1 := graph.GetCacheKey()

	// Add another node and edge
	graph.AddNode(createTestNode("c", DependencySubquery))
	graph.AddDependency("b", "c")

	key2 := graph.GetCacheKey()

	// Keys should be different
	assert.NotEqual(t, key1, key2)
}

func TestParallelSortNodes(t *testing.T) {
	config := DefaultPerformanceConfig()
	optimizer := NewPerformanceOptimizer(config)
	graph := NewOptimizedDependencyGraph(optimizer)

	nodes := []string{"a", "b", "c", "d"}
	sorted := graph.parallelSortNodes(nodes)

	// For now, the implementation just returns the same nodes
	// This is a placeholder for actual parallel sorting
	assert.Equal(t, nodes, sorted)
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent access to performance stats
	stats := NewPerformanceStats()

	// Use goroutines to test thread safety
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			stats.RecordProcessingTime(time.Millisecond)
			stats.RecordCacheHit()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			stats.RecordIndexBuildTime(time.Millisecond)
			stats.RecordCacheMiss()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	metrics := stats.GetStats()

	// Should have recorded all operations
	assert.Equal(t, 100, metrics.ProcessingCount)
	assert.Equal(t, 100, metrics.CacheHits)
	assert.Equal(t, 100, metrics.CacheMisses)
}
