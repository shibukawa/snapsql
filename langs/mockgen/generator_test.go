package mockgen_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/langs/mockgen"
)

func TestGeneratorWritesMockJSON(t *testing.T) {
	intermediateDir := t.TempDir()
	intermediatePath := filepath.Join(intermediateDir, "board_list.json")

	format := &intermediate.IntermediateFormat{
		FormatVersion: "1",
		FunctionName:  "board_list",
		MockTestCases: []snapsql.MockTestCase{
			{
				Name:       "happy path",
				Parameters: map[string]any{"id": 42},
				Responses: []snapsql.MockResponse{
					{Expected: []map[string]any{{"id": 42, "name": "Board 42"}}},
				},
			},
		},
	}

	bytes, err := format.MarshalJSON()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(intermediatePath, bytes, 0o644))

	outputDir := filepath.Join(intermediateDir, "mock")
	gen := mockgen.Generator{OutputDir: outputDir}

	generatedFile, err := gen.Generate(intermediatePath)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(outputDir, "board_list.json"), generatedFile)

	data, err := os.ReadFile(generatedFile)
	require.NoError(t, err)

	var payload []map[string]any
	require.NoError(t, json.Unmarshal(data, &payload))
	require.Len(t, payload, 1)
	require.Equal(t, "happy path", payload[0]["name"])

	params, ok := payload[0]["parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(42), params["id"])

	responses, ok := payload[0]["responses"].([]any)
	require.True(t, ok)
	require.Len(t, responses, 1)
	respMap, ok := responses[0].(map[string]any)
	require.True(t, ok)

	expected, ok := respMap["expected"].([]any)
	require.True(t, ok)
	require.Len(t, expected, 1)
	row, ok := expected[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Board 42", row["name"])
}

func TestGeneratorProducesEmptyArrayWhenNoCases(t *testing.T) {
	intermediateDir := t.TempDir()
	intermediatePath := filepath.Join(intermediateDir, "empty.json")

	format := &intermediate.IntermediateFormat{
		FormatVersion: "1",
		FunctionName:  "empty_case",
	}

	bytes, err := format.MarshalJSON()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(intermediatePath, bytes, 0o644))

	outputDir := filepath.Join(intermediateDir, "mock")
	gen := mockgen.Generator{OutputDir: outputDir}

	generatedFile, err := gen.Generate(intermediatePath)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(outputDir, "empty_case.json"), generatedFile)

	data, err := os.ReadFile(generatedFile)
	require.NoError(t, err)
	require.Equal(t, "[]\n", string(data))
}
