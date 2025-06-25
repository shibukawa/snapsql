package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"gopkg.in/yaml.v3"
)

func TestOrderedParameters(t *testing.T) {
	ordered := NewOrderedParameters()

	// Add parameters (simplified - no path or order)
	ordered.Add("user_id", "int")
	ordered.Add("name", "string")
	ordered.Add("active", "bool")
	ordered.Add("departments", []any{"string"})

	// Test GetByName
	param, exists := ordered.GetByName("name")
	assert.True(t, exists)
	assert.Equal(t, "name", param.Name)
	assert.Equal(t, "string", param.Type)

	// Test GetInOrder
	inOrder := ordered.GetInOrder()
	assert.Equal(t, 4, len(inOrder))
	assert.Equal(t, "user_id", inOrder[0].Name)
	assert.Equal(t, "name", inOrder[1].Name)
	assert.Equal(t, "active", inOrder[2].Name)
	assert.Equal(t, "departments", inOrder[3].Name)

	// Test non-existent parameter
	_, exists = ordered.GetByName("nonexistent")
	assert.False(t, exists)
}

func TestParseParametersWithOrder(t *testing.T) {
	yamlContent := `
name: test_query
parameters:
  user_id: int
  filters:
    name: string
    active: bool
    departments:
      - string
`

	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &yamlNode)
	assert.NoError(t, err)

	ordered, err := parseParametersWithOrder(&yamlNode)
	assert.NoError(t, err)
	assert.NotZero(t, ordered)

	// Check parsed parameters
	inOrder := ordered.GetInOrder()
	assert.Equal(t, 2, len(inOrder)) // user_id and filters

	// Check user_id parameter
	userIdParam, exists := ordered.GetByName("user_id")
	assert.True(t, exists)
	assert.Equal(t, "user_id", userIdParam.Name)
	assert.Equal(t, "int", userIdParam.Type)

	// Check filters parameter (should be map[string]any)
	filtersParam, exists := ordered.GetByName("filters")
	assert.True(t, exists)
	assert.Equal(t, "filters", filtersParam.Name)

	// filters should be a map
	filtersMap, ok := filtersParam.Type.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", filtersMap["name"])
	assert.Equal(t, "bool", filtersMap["active"])

	// departments should be an array
	departments, ok := filtersMap["departments"].([]any)
	assert.True(t, ok)
	assert.Equal(t, 1, len(departments))
	assert.Equal(t, "string", departments[0])
}

func TestConvertYamlNodeToMap(t *testing.T) {
	yamlContent := `
name: string
active: bool
`
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &yamlNode)
	assert.NoError(t, err)

	// Get the mapping node
	mappingNode := yamlNode.Content[0]

	result, err := convertYamlNodeToMap(mappingNode)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "string", result["name"])
	assert.Equal(t, "bool", result["active"])
}

func TestConvertYamlNodeToArray(t *testing.T) {
	yamlContent := `
- string
- int
- bool
`
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &yamlNode)
	assert.NoError(t, err)

	// Get the sequence node
	sequenceNode := yamlNode.Content[0]

	result, err := convertYamlNodeToArray(sequenceNode)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "string", result[0])
	assert.Equal(t, "int", result[1])
	assert.Equal(t, "bool", result[2])
}

func TestConvertYamlNodeToAny(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected any
	}{
		{
			name:     "scalar string",
			yaml:     "test_value",
			expected: "test_value",
		},
		{
			name:     "scalar number",
			yaml:     "123",
			expected: "123", // YAML scalars are strings in our implementation
		},
		{
			name: "simple map",
			yaml: `
name: string
active: bool`,
			expected: map[string]any{
				"name":   "string",
				"active": "bool",
			},
		},
		{
			name: "simple array",
			yaml: `
- string
- int`,
			expected: []any{"string", "int"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &yamlNode)
			assert.NoError(t, err)

			// Get the content node
			contentNode := yamlNode.Content[0]

			result, err := convertYamlNodeToAny(contentNode)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
