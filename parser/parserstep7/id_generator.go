package parserstep7

import (
	"fmt"
	"sync"
)

// IDGenerator generates unique IDs for subqueries and dependencies
type IDGenerator struct {
	counter map[string]int
	mutex   sync.Mutex
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		counter: make(map[string]int),
	}
}

// Generate generates a unique ID with the given prefix
func (ig *IDGenerator) Generate(prefix string) string {
	ig.mutex.Lock()
	defer ig.mutex.Unlock()

	ig.counter[prefix]++

	return fmt.Sprintf("%s_%d", prefix, ig.counter[prefix])
}

// Reset resets the counter for a specific prefix
func (ig *IDGenerator) Reset(prefix string) {
	ig.mutex.Lock()
	defer ig.mutex.Unlock()

	delete(ig.counter, prefix)
}

// ResetAll resets all counters
func (ig *IDGenerator) ResetAll() {
	ig.mutex.Lock()
	defer ig.mutex.Unlock()

	ig.counter = make(map[string]int)
}
