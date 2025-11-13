package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestLoadIntermediateFormat_MarkdownSimple(t *testing.T) {
	// Create a simple markdown template without CEL expressions
	markdownContent := `---
function_name: get_user_simple
description: Get user information by ID (simple)
---

# Get User By ID (Simple)

## Description

Get user information by ID (simple version without CEL expressions).

## Parameters

` + "```yaml" + `
user_id:
  type: int
  description: The user ID to search for
` + "```" + `

## SQL

` + "```sql" + `
SELECT u.id, u.name, u.email
FROM users u
WHERE u.id = /*= user_id */
LIMIT 10
` + "```" + `
`

	// Create temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_simple_*.snap.md")
	assert.NoError(t, err)

	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(markdownContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// Test loading
	format, err := LoadIntermediateFormat(tmpFile.Name())
	assert.NoError(t, err)

	if format == nil {
		t.Fatal("format should not be nil")
		return
	}

	// Verify basic properties
	assert.Equal(t, "get_user_simple", format.FunctionName)
	assert.Equal(t, "Get user information by ID (simple)", format.Description)

	if len(format.Parameters) == 0 {
		t.Error("Parameters should not be empty")
	}
}

func TestLoadIntermediateFormat_MarkdownWithSimpleParameters(t *testing.T) {
	// Create a temporary markdown file with simple parameter format
	markdownContent := `# Simple Query

## Description

Simple user query with basic filters.

## Parameters

` + "```yaml" + `
name: string
age: int
active: bool
` + "```" + `

## SQL

` + "```sql" + `
SELECT id, name, age, active FROM users 
WHERE name = /*= name */'dummy'
  AND age = /*= age */25
  AND active = /*= active */true
` + "```" + `
`

	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.snap.md")
	err := os.WriteFile(tmpFile, []byte(markdownContent), 0644)
	assert.NoError(t, err)

	// Load intermediate format
	format, err := LoadIntermediateFormat(tmpFile)
	assert.NoError(t, err)

	if format == nil {
		t.Fatalf("format is nil")
		return
	}

	// Check parameters
	assert.Equal(t, 3, len(format.Parameters))

	// Create a map for easier testing
	paramMap := make(map[string]intermediate.Parameter)
	for _, param := range format.Parameters {
		paramMap[param.Name] = param
	}

	// Test name parameter
	nameParam, exists := paramMap["name"]
	assert.True(t, exists)
	assert.Equal(t, "name", nameParam.Name)
	assert.Equal(t, "string", nameParam.Type)
	assert.Equal(t, "", nameParam.Description) // No description in simple format
	assert.False(t, nameParam.Optional)

	// Test age parameter
	ageParam, exists := paramMap["age"]
	assert.True(t, exists)
	assert.Equal(t, "age", ageParam.Name)
	assert.Equal(t, "int", ageParam.Type)

	// Test active parameter
	activeParam, exists := paramMap["active"]
	assert.True(t, exists)
	assert.Equal(t, "active", activeParam.Name)
	assert.Equal(t, "bool", activeParam.Type)
}

/*
func TestConvertParametersToIntermediate(t *testing.T) {
	testCases := []struct {
		name           string
		markdownParams map[string]any
		expectedParams []intermediate.Parameter
		expectError    bool
	}{
		{
			name: "simple string parameters",
			markdownParams: map[string]any{
				"name": "string",
				"age":  "int",
			},
			expectedParams: []intermediate.Parameter{
				{Name: "name", Type: "string"},
				{Name: "age", Type: "int"},
			},
			expectError: false,
		},
		{
			name: "complex parameters with description",
			markdownParams: map[string]any{
				"user_id": map[string]any{
					"type":        "int",
					"description": "User ID",
					"optional":    false,
				},
				"include_profile": map[string]any{
					"type":        "bool",
					"description": "Include profile data",
					"optional":    true,
				},
			},
			expectedParams: []intermediate.Parameter{
				{Name: "user_id", Type: "int", Description: "User ID", Optional: false},
				{Name: "include_profile", Type: "bool", Description: "Include profile data", Optional: true},
			},
			expectError: false,
		},
		{
			name: "invalid parameter definition",
			markdownParams: map[string]any{
				"invalid": 123, // Invalid type
			},
			expectedParams: nil,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := convertParametersToIntermediate(tc.markdownParams)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.expectedParams), len(params))

				// Convert to map for easier comparison
				paramMap := make(map[string]intermediate.Parameter)
				for _, param := range params {
					paramMap[param.Name] = param
				}

				for _, expected := range tc.expectedParams {
					actual, exists := paramMap[expected.Name]
					assert.True(t, exists, "Parameter %s should exist", expected.Name)
					assert.Equal(t, expected.Type, actual.Type)
					assert.Equal(t, expected.Description, actual.Description)
					assert.Equal(t, expected.Optional, actual.Optional)
				}
			}
		})
	}
}
*/

func TestLoadIntermediateFormat_FileTypes(t *testing.T) {
	testCases := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "JSON file (non-existent)",
			filename:    "test.json",
			expectError: true,
			errorMsg:    "template file not found",
		},
		{
			name:        "SQL file (non-existent)",
			filename:    "test.snap.sql",
			expectError: true,
			errorMsg:    "template file not found",
		},
		{
			name:        "Markdown file (non-existent)",
			filename:    "test.snap.md",
			expectError: true,
			errorMsg:    "template file not found",
		},
		{
			name:        "Unsupported file type",
			filename:    "test.txt",
			expectError: true,
			errorMsg:    "template file not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadIntermediateFormat(tc.filename)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
