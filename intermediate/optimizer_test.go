package intermediate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimizeInstructions_SystemLimitFallback(t *testing.T) {
	instructions := []Instruction{
		{Op: OpEmitStatic, Value: "SELECT * FROM demo ORDER BY id ASC LIMIT "},
		{Op: OpIfSystemLimit},
		{Op: OpEmitSystemLimit},
		{Op: OpElse},
		{Op: OpEmitStatic, Value: "1"},
		{Op: OpEnd},
		{Op: OpEmitStatic, Value: " -- tail"},
	}

	optimized, err := OptimizeInstructions(instructions, "")
	require.NoError(t, err)

	var emitted []string

	for _, inst := range optimized {
		switch inst.Op {
		case "EMIT_STATIC":
			emitted = append(emitted, inst.Value)
		case "ADD_PARAM", "ADD_SYSTEM_PARAM":
			// keep order but not needed for this test
		default:
			assert.NotEqual(t, "IF", inst.Op)
			assert.NotEqual(t, "ELSE", inst.Op)
			assert.NotEqual(t, "END", inst.Op)
		}
	}

	joined := strings.Join(emitted, "")
	assert.Contains(t, joined, "LIMIT 1")
	assert.Contains(t, joined, "-- tail")
	assert.NotContains(t, joined, "LIMIT  ")
}

func TestOptimizeInstructions_SystemLimitAuto(t *testing.T) {
	instructions := []Instruction{
		{Op: OpEmitStatic, Value: "SELECT * FROM demo"},
		{Op: OpIfSystemLimit},
		{Op: OpEmitStatic, Value: " LIMIT "},
		{Op: OpEmitSystemLimit},
		{Op: OpEnd},
		{Op: OpEmitStatic, Value: " ORDER BY id"},
	}

	optimized, err := OptimizeInstructions(instructions, "")
	require.NoError(t, err)

	var joined strings.Builder

	for _, inst := range optimized {
		if inst.Op == "EMIT_STATIC" {
			joined.WriteString(inst.Value)
		}

		assert.NotEqual(t, "IF", inst.Op)
		assert.NotEqual(t, "ELSE", inst.Op)
		assert.NotEqual(t, "END", inst.Op)
	}

	sql := joined.String()
	assert.Equal(t, -1, strings.Index(sql, "LIMIT"))
	assert.Contains(t, sql, "ORDER BY id")
}
