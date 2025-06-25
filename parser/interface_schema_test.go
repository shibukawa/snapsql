package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestInterfaceSchemaParser(t *testing.T) {
	// Test that the new functions work
	schema, err := NewInterfaceSchemaFromFrontMatter("")
	assert.NoError(t, err)
	assert.True(t, schema != nil)
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
metadata:
  author: "dev-team"
  created: "2024-01-01"
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

	schema, err := NewInterfaceSchemaFromFrontMatter(markdownContent)

	assert.NoError(t, err)
	assert.Equal(t, "user_search", schema.Name)
	assert.Equal(t, "searchUsers", schema.FunctionName)
	assert.Equal(t, "Search users with filters", schema.Description)

	// Check parameters
	assert.True(t, schema.Parameters != nil)
	assert.Equal(t, "str", schema.Parameters["search_query"])

	filters, ok := schema.Parameters["filters"].(map[string]any)
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
	tok := tokenizer.NewSqlTokenizer(sqlContent, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	schema, err := NewInterfaceSchemaFromSQL(tokens)

	assert.NoError(t, err)
	assert.Equal(t, "comment_query", schema.Name)
	assert.Equal(t, "commentQuery", schema.FunctionName)

	// Check parameters
	assert.Equal(t, "int", schema.Parameters["user_id"])
	assert.Equal(t, "bool", schema.Parameters["include_email"])
}

func TestSchemaCommentFiltering(t *testing.T) {
	// Test that regular comments are ignored, only /*@ */ comments are processed
	sqlContent := `
/* This is a regular comment with parameters: and name: that should be ignored */
/*@
name: schema_query
parameters:
 user_id: int
*/
/* Another regular comment */
SELECT id, name FROM users WHERE id = /*= user_id */;`

	// Tokenize SQL first
	tok := tokenizer.NewSqlTokenizer(sqlContent, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	schema, err := NewInterfaceSchemaFromSQL(tokens)

	assert.NoError(t, err)
	assert.Equal(t, "schema_query", schema.Name)
	assert.Equal(t, "int", schema.Parameters["user_id"])
}

func TestInterfaceSchema(t *testing.T) {
	schema := &InterfaceSchema{
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

	// Process schema to generate flattened parameters and CEL variables
	schema.ProcessSchema()

	// Check hierarchical parameters (instead of flattened)
	userIdType, exists := schema.GetParameterType("user_id")
	assert.True(t, exists)
	assert.Equal(t, "int", userIdType)

	activeType, exists := schema.GetParameterType("filters.active")
	assert.True(t, exists)
	assert.Equal(t, "bool", activeType)

	deptType, exists := schema.GetParameterType("filters.departments")
	assert.True(t, exists)
	assert.Equal(t, "list[str]", deptType) // GetParameterType returns old format for compatibility

	minType, exists := schema.GetParameterType("filters.score_range.min")
	assert.True(t, exists)
	assert.Equal(t, "float", minType)

	maxType, exists := schema.GetParameterType("filters.score_range.max")
	assert.True(t, exists)
	assert.Equal(t, "float", maxType)

	// Check hierarchical structure preservation
	assert.True(t, len(schema.Parameters) >= 2, "Should have hierarchical parameters")

	// Check parameter types using hierarchical lookup
	userIdType, userIdExists := schema.GetParameterType("user_id")
	assert.True(t, userIdExists)
	assert.Equal(t, "int", userIdType)

	activeType, activeExists := schema.GetParameterType("filters.active")
	assert.True(t, activeExists)
	assert.Equal(t, "bool", activeType)

	depsType, depsExists := schema.GetParameterType("filters.departments")
	assert.True(t, depsExists)
	assert.Equal(t, "list[str]", depsType) // GetParameterType returns old format for compatibility
}

func TestInterfaceParameterValidation(t *testing.T) {
	schema := &InterfaceSchema{
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"active": "bool",
			},
		},
	}

	// Process schema to generate flattened parameters
	schema.ProcessSchema()

	// Valid parameters
	err := schema.ValidateParameterReference("user_id")
	assert.NoError(t, err)

	err = schema.ValidateParameterReference("filters.active")
	assert.NoError(t, err)

	// Invalid parameter
	err = schema.ValidateParameterReference("nonexistent")
	assert.Error(t, err)

	err = schema.ValidateParameterReference("filters.nonexistent")
	assert.Error(t, err)
}

func TestGetParameterType(t *testing.T) {
	schema := &InterfaceSchema{
		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"departments": []any{"str"},
			},
		},
	}

	// Process schema to generate flattened parameters
	schema.ProcessSchema()

	// Existing parameters
	paramType, exists := schema.GetParameterType("user_id")
	assert.True(t, exists)
	assert.Equal(t, "int", paramType)

	paramType, exists = schema.GetParameterType("filters.departments")
	assert.True(t, exists)
	assert.Equal(t, "list[str]", paramType) // GetParameterType returns old format for compatibility

	// Non-existing parameter
	_, exists = schema.GetParameterType("nonexistent")
	assert.False(t, exists)
}

func TestGetFunctionMetadata(t *testing.T) {
	schema := &InterfaceSchema{
		Name:         "user_query",
		FunctionName: "getUserData",
		Description:  "Get user data",
	}

	metadata := schema.GetFunctionMetadata()

	assert.Equal(t, "user_query", metadata["name"])
	assert.Equal(t, "getUserData", metadata["function_name"])
	assert.Equal(t, "Get user data", metadata["description"])
}

func TestGetTags(t *testing.T) {
	schema := &InterfaceSchema{}

	tags := schema.GetTags()

	assert.Equal(t, []string{}, tags) // GetTags now returns empty slice
}

func TestEmptySchema(t *testing.T) {
	// Test with empty frontmatter YAML
	schema, err := NewInterfaceSchemaFromFrontMatter("")

	assert.NoError(t, err)
	assert.True(t, schema != nil)
	assert.True(t, schema.Parameters != nil)
	assert.Equal(t, 0, len(schema.Parameters))
}
