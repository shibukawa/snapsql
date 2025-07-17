package parsercommon

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestParameterOrderFromYAML(t *testing.T) {
	yamlText := `parameters:
    user_id: int
    include_email: bool
    filters:
        active: bool
        departments: [str]`
	def, err := NewFunctionDefinitionFromYAML(yamlText)
	assert.NoError(t, err)
	assert.Equal(t, []string{"user_id", "include_email", "filters"}, def.ParameterOrder)
}

func TestInterfaceSchemaParser(t *testing.T) {
	// Test that the new functions work
	def, err := NewFunctionDefinitionFromYAML("")
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

	def, err := NewFunctionDefinitionFromYAML(markdownContent)

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
	sqlContent := `/*#
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

func TestNewFunctionDefinitionFromMarkdown(t *testing.T) {
	frontMatter := map[string]any{
		"name":          "user_query",
		"function_name": "getUserData",
		"description":   "Query user data with filters",
		"version":       "1.0.0",
	}

	parametersText := `user_id: int
include_email: bool
filters:
    active: bool
    departments: list[str]`

	description := "This query retrieves user data"

	def, err := NewFunctionDefinitionFromMarkdown(frontMatter, parametersText, description)

	assert.NoError(t, err)
	assert.Equal(t, "user_query", def.Name)
	assert.Equal(t, "getUserData", def.FunctionName)
	assert.Equal(t, "Query user data with filters", def.Description) // Front matter takes precedence

	// Check parameters
	assert.True(t, def.Parameters != nil)
	assert.Equal(t, "int", def.Parameters["user_id"])
	assert.Equal(t, "bool", def.Parameters["include_email"])

	filters, ok := def.Parameters["filters"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bool", filters["active"])
	assert.Equal(t, "list[str]", filters["departments"])

	// Check parameter order
	assert.Equal(t, []string{"user_id", "include_email", "filters"}, def.ParameterOrder)
}

func TestNewFunctionDefinitionFromMarkdownDescriptionFallback(t *testing.T) {
	frontMatter := map[string]any{
		"name":          "simple_query",
		"function_name": "getSimpleData",
	}

	parametersText := `page: int
size: int`

	description := "Fallback description from overview section"

	def, err := NewFunctionDefinitionFromMarkdown(frontMatter, parametersText, description)

	assert.NoError(t, err)
	assert.Equal(t, "simple_query", def.Name)
	assert.Equal(t, "getSimpleData", def.FunctionName)
	assert.Equal(t, "Fallback description from overview section", def.Description) // Description fallback

	// Check parameters
	assert.Equal(t, "int", def.Parameters["page"])
	assert.Equal(t, "int", def.Parameters["size"])
	assert.Equal(t, []string{"page", "size"}, def.ParameterOrder)
}

func TestNewFunctionDefinitionFromMarkdownEmptyParameters(t *testing.T) {
	frontMatter := map[string]any{
		"name":          "no_params_query",
		"function_name": "getNoParams",
		"description":   "Query without parameters",
	}

	def, err := NewFunctionDefinitionFromMarkdown(frontMatter, "", "")

	assert.NoError(t, err)
	assert.Equal(t, "no_params_query", def.Name)
	assert.Equal(t, "getNoParams", def.FunctionName)
	assert.Equal(t, "Query without parameters", def.Description)

	// Check empty parameters
	assert.True(t, def.Parameters != nil)
	assert.Equal(t, 0, len(def.Parameters))
	assert.Equal(t, []string{}, def.ParameterOrder)
}

func TestValidateParameterName(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		shouldError bool
		errorType   error
	}{
		// Valid names
		{"valid_underscore", "user_id", false, nil},
		{"valid_camelCase", "userId", false, nil},
		{"valid_single_char", "x", false, nil},
		{"valid_start_underscore", "_private", false, nil},
		{"valid_mixed", "user_Name123", false, nil},

		// Invalid names - empty
		{"empty_name", "", true, ErrInvalidParameterName},

		// Invalid names - naming convention
		{"start_with_digit", "123abc", true, ErrInvalidNamingConvention},
		{"contains_hyphen", "user-id", true, ErrInvalidNamingConvention},
		{"contains_space", "user id", true, ErrInvalidNamingConvention},
		{"contains_special", "user@id", true, ErrInvalidNamingConvention},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterName(tt.paramName)
			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.True(t, errors.Is(err, tt.errorType), "expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAllParameterNames(t *testing.T) {
	tests := []struct {
		name        string
		parameters  map[string]any
		shouldError bool
	}{
		{
			name: "valid_parameters",
			parameters: map[string]any{
				"user_id":   123,
				"user_name": "test",
				"is_active": true,
				"nested_obj": map[string]any{
					"inner_field": "value",
					"count_123":   456,
				},
			},
			shouldError: false,
		},
		{
			name: "invalid_naming_convention",
			parameters: map[string]any{
				"user_id":   123,
				"user_name": "test",
			},
			shouldError: false,
		},
		{
			name: "invalid_starts_with_digit",
			parameters: map[string]any{
				"user_id":    123,
				"123invalid": "test", // starts with digit
			},
			shouldError: true,
		},
		{
			name: "invalid_nested_parameter",
			parameters: map[string]any{
				"user_id": 123,
				"filters": map[string]any{
					"valid_field": "ok",
					"123invalid":  "invalid", // starts with digit in nested
				},
			},
			shouldError: true,
		},
		{
			name:        "empty_parameters",
			parameters:  map[string]any{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAllParameterNames(tt.parameters, "")
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewFunctionDefinitionFromYAMLWithValidation(t *testing.T) {
	tests := []struct {
		name        string
		yamlText    string
		shouldError bool
	}{
		{
			name: "valid_parameters",
			yamlText: `name: test_func
parameters:
  user_id: int
  user_name: string
  is_active: bool`,
			shouldError: false,
		},
		{
			name: "invalid_naming_convention",
			yamlText: `name: test_func
parameters:
  user_id: int
  123invalid: string`,
			shouldError: true,
		},
		{
			name: "invalid_nested_parameter",
			yamlText: `name: test_func
parameters:
  user_id: int
  filters:
    active: bool
    123invalid: string`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFunctionDefinitionFromYAML(tt.yamlText)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
