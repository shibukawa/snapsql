package parsercommon

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
	"gopkg.in/yaml.v3"
)

func TestInterfaceSchemaParser(t *testing.T) {
	// Test that the new functions work
	def, err := NewFunctionDefinitionFromFrontMatter("")
	assert.NoError(t, err)
	assert.True(t, def != nil)
}

func TestFrontmatterParsing(t *testing.T) {
	markdownContent := `---
name: user_search
function_name: searchUsers
description: Search users with filters
version: "1.0.0"
parameters:
  search_query: str
  filters:
    active: bool
    departments: list[str]
  pagination:
    page: int
    size: int
tags:
  - user
  - search
---

# User Search Query

This query searches for users based on various criteria.

SELECT id, name, email FROM users
WHERE name LIKE /*= search_query + '%' */'test%'
 AND active = /*= filters.active */true
 /*# if filters.departments */
 AND department IN (/*= filters.departments */)
 /*# end */
LIMIT /*= pagination.size */10
OFFSET /*= pagination.page * pagination.size */0;`

	def, err := NewFunctionDefinitionFromFrontMatter(markdownContent)

	assert.NoError(t, err)
	assert.Equal(t, "user_search", def.Name)
	assert.Equal(t, "searchUsers", def.FunctionName)
	assert.Equal(t, "Search users with filters", def.Description)

	// Check parameters
	assert.True(t, def.Parameters != nil)
	assert.Equal(t, "str", def.Parameters["search_query"])

	filters, ok := def.Parameters["filters"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bool", filters["active"])
	assert.Equal(t, "list[str]", filters["departments"]) // GetParameterType returns old format for compatibility
}

func TestCommentBlockParsing(t *testing.T) {
	sqlContent := `/*@
name: comment_query
function_name: commentQuery
parameters:
 user_id: int
 include_email: bool
*/

SELECT id, name FROM users WHERE id = /*= user_id */;`

	// Tokenize SQL first
	tokens, err := tokenizer.Tokenize(sqlContent)
	assert.NoError(t, err)

	def, err := NewFunctionDefinitionFromSQL(tokens)

	assert.NoError(t, err)
	assert.Equal(t, "comment_query", def.Name)
	assert.Equal(t, "commentQuery", def.FunctionName)

	// Check parameters
	assert.Equal(t, "int", def.Parameters["user_id"])
	assert.Equal(t, "bool", def.Parameters["include_email"])
}

func TestSchemaCommentFiltering(t *testing.T) {
	// Test that regular comments are ignored, only /*@ */ comments are processed
	sqlContent := `
/* This is a regular comment with parameters: and name: that should be ignored */
/*@
name: def_query
parameters:
 user_id: int
*/
/* Another regular comment */
SELECT id, name FROM users WHERE id = /*= user_id */;`

	// Tokenize SQL first
	tokens, err := tokenizer.Tokenize(sqlContent)
	assert.NoError(t, err)

	def, err := NewFunctionDefinitionFromSQL(tokens)

	assert.NoError(t, err)
	assert.Equal(t, "def_query", def.Name)
	assert.Equal(t, "int", def.Parameters["user_id"])
}

func TestInterfaceSchema(t *testing.T) {
	def := &FunctionDefinition{
		Name:         "test_query",
		FunctionName: "testQuery",
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"active":      "bool",
				"departments": []any{"str"},
				"score_range": map[string]any{
					"min": "float",
					"max": "float",
				},
			},
		},
	}

	// Process def to generate flattened parameters and CEL variables
	def.ProcessDefinition()

	// Check hierarchical parameters (instead of flattened)
	userIdType, exists := def.GetParameterType("user_id")
	assert.True(t, exists)
	assert.Equal(t, "int", userIdType)

	activeType, exists := def.GetParameterType("filters.active")
	assert.True(t, exists)
	assert.Equal(t, "bool", activeType)

	deptType, exists := def.GetParameterType("filters.departments")
	assert.True(t, exists)
	assert.Equal(t, "list[str]", deptType) // GetParameterType returns old format for compatibility

	minType, exists := def.GetParameterType("filters.score_range.min")
	assert.True(t, exists)
	assert.Equal(t, "float", minType)

	maxType, exists := def.GetParameterType("filters.score_range.max")
	assert.True(t, exists)
	assert.Equal(t, "float", maxType)

	// Check hierarchical structure preservation
	assert.True(t, len(def.Parameters) >= 2, "Should have hierarchical parameters")

	// Check parameter types using hierarchical lookup
	userIdType, userIdExists := def.GetParameterType("user_id")
	assert.True(t, userIdExists)
	assert.Equal(t, "int", userIdType)

	activeType, activeExists := def.GetParameterType("filters.active")
	assert.True(t, activeExists)
	assert.Equal(t, "bool", activeType)

	depsType, depsExists := def.GetParameterType("filters.departments")
	assert.True(t, depsExists)
	assert.Equal(t, "list[str]", depsType) // GetParameterType returns old format for compatibility
}

func TestInterfaceParameterValidation(t *testing.T) {
	def := &FunctionDefinition{
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"active": "bool",
			},
		},
	}

	// Process def to generate flattened parameters
	def.ProcessDefinition()

	// Valid parameters
	err := def.ValidateParameterReference("user_id")
	assert.NoError(t, err)

	err = def.ValidateParameterReference("filters.active")
	assert.NoError(t, err)

	// Invalid parameter
	err = def.ValidateParameterReference("nonexistent")
	assert.Error(t, err)

	err = def.ValidateParameterReference("filters.nonexistent")
	assert.Error(t, err)
}

func TestGetParameterType(t *testing.T) {
	def := &FunctionDefinition{
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"departments": []any{"str"},
			},
		},
	}

	// Process def to generate flattened parameters
	def.ProcessDefinition()

	// Existing parameters
	paramType, exists := def.GetParameterType("user_id")
	assert.True(t, exists)
	assert.Equal(t, "int", paramType)

	paramType, exists = def.GetParameterType("filters.departments")
	assert.True(t, exists)
	assert.Equal(t, "list[str]", paramType) // GetParameterType returns old format for compatibility

	// Non-existing parameter
	_, exists = def.GetParameterType("nonexistent")
	assert.False(t, exists)
}

func TestGetFunctionMetadata(t *testing.T) {
	def := &FunctionDefinition{
		Name:         "user_query",
		FunctionName: "getUserData",
		Description:  "Get user data",
	}

	metadata := def.GetFunctionMetadata()

	assert.Equal(t, "user_query", metadata["name"])
	assert.Equal(t, "getUserData", metadata["function_name"])
	assert.Equal(t, "Get user data", metadata["description"])
}

func TestGetTags(t *testing.T) {
	def := &FunctionDefinition{}

	tags := def.GetTags()

	assert.Equal(t, []string{}, tags) // GetTags now returns empty slice
}

func TestEmptySchema(t *testing.T) {
	// Test with empty frontmatter YAML
	def, err := NewFunctionDefinitionFromFrontMatter("")

	assert.NoError(t, err)
	assert.True(t, def != nil)
	assert.True(t, def.Parameters != nil)
	assert.Equal(t, 0, len(def.Parameters))
}

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
