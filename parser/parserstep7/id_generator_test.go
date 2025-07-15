package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestIDGenerator_Generate(t *testing.T) {
	ig := NewIDGenerator()

	id1 := ig.Generate("test")
	id2 := ig.Generate("test")
	id3 := ig.Generate("other")

	assert.Equal(t, "test_1", id1)
	assert.Equal(t, "test_2", id2)
	assert.Equal(t, "other_1", id3)
}

func TestIDGenerator_Reset(t *testing.T) {
	ig := NewIDGenerator()

	ig.Generate("test")
	ig.Generate("test")
	ig.Reset("test")

	id := ig.Generate("test")
	assert.Equal(t, "test_1", id)
}

func TestIDGenerator_ResetAll(t *testing.T) {
	ig := NewIDGenerator()

	ig.Generate("test1")
	ig.Generate("test2")
	ig.ResetAll()

	id1 := ig.Generate("test1")
	id2 := ig.Generate("test2")

	assert.Equal(t, "test1_1", id1)
	assert.Equal(t, "test2_1", id2)
}
