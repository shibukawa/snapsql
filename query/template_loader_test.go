package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	. "github.com/shibukawa/snapsql" // For TableInfo and ColumnInfo
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
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
	}

	// Verify basic properties
	assert.Equal(t, "get_user_simple", format.FunctionName)
	assert.Equal(t, "Get user information by ID (simple)", format.Description)

	if len(format.Parameters) == 0 {
		t.Error("Parameters should not be empty")
	}
}

func TestLoadIntermediateFormat_MarkdownWithParameters(t *testing.T) {
	// Create a temporary markdown file with parameters
	markdownContent := `---
function_name: get_user_by_id
description: Get user information by ID
---

# Get User By ID

## Description

Get user information by ID with optional profile data.

## Parameters

` + "```yaml" + `
user_id: int
include_profile: bool
limit: int
` + "```" + `

## SQL

` + "```sql" + `
SELECT u.id, u.name, u.email
/*# if include_profile */
    , p.bio, p.avatar_url
/*# end */
FROM users u
/*# if include_profile */
    LEFT JOIN profiles p ON u.id = p.user_id
/*# end */
WHERE u.id = /*= user_id */1
LIMIT /*= limit != 0 ? limit : 10 */10
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
	assert.True(t, format != nil)

	// Check function name and description
	assert.Equal(t, "get_user_by_id", format.FunctionName)
	assert.Equal(t, "Get user information by ID", format.Description)

	// Check parameters
	assert.Equal(t, 3, len(format.Parameters))

	// Create a map for easier testing
	paramMap := make(map[string]intermediate.Parameter)
	for _, param := range format.Parameters {
		paramMap[param.Name] = param
	}

	// Test user_id parameter
	userIdParam, exists := paramMap["user_id"]
	assert.True(t, exists)
	assert.Equal(t, "user_id", userIdParam.Name)
	assert.Equal(t, "int", userIdParam.Type)
	assert.Equal(t, "", userIdParam.Description) // No description in simple format
	assert.False(t, userIdParam.Optional)

	// Test include_profile parameter
	includeProfileParam, exists := paramMap["include_profile"]
	assert.True(t, exists)
	assert.Equal(t, "include_profile", includeProfileParam.Name)
	assert.Equal(t, "bool", includeProfileParam.Type)
	assert.Equal(t, "", includeProfileParam.Description) // No description in simple format
	assert.False(t, includeProfileParam.Optional)        // Default is false

	// Test limit parameter
	limitParam, limitExists := paramMap["limit"]
	assert.True(t, limitExists)
	assert.Equal(t, "limit", limitParam.Name)
	assert.Equal(t, "int", limitParam.Type)
	assert.Equal(t, "", limitParam.Description) // No description in simple format
	assert.False(t, limitParam.Optional)        // Default is false
}

func TestLoadIntermediateFormat_MarkdownWithJSONParameters(t *testing.T) {
	// Create a temporary markdown file with JSON parameters
	markdownContent := `# Get User List

## Description

Get paginated user list.

## Parameters

` + "```json" + `
{
  "page": {
    "type": "int",
    "description": "Page number",
    "optional": true
  },
  "page_size": {
    "type": "int", 
    "description": "Number of items per page"
  }
}
` + "```" + `

## SQL

` + "```sql" + `
SELECT id, name, email FROM users
LIMIT /*= page_size != 0 ? page_size : 10 */10
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */0
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
	assert.True(t, format != nil)

	// Check parameters
	assert.Equal(t, 2, len(format.Parameters))

	// Create a map for easier testing
	paramMap := make(map[string]intermediate.Parameter)
	for _, param := range format.Parameters {
		paramMap[param.Name] = param
	}

	// Test page parameter
	pageParam, exists := paramMap["page"]
	assert.True(t, exists)
	assert.Equal(t, "page", pageParam.Name)
	assert.Equal(t, "int", pageParam.Type)
	assert.Equal(t, "Page number", pageParam.Description)
	assert.True(t, pageParam.Optional)

	// Test page_size parameter
	pageSizeParam, exists := paramMap["page_size"]
	assert.True(t, exists)
	assert.Equal(t, "page_size", pageSizeParam.Name)
	assert.Equal(t, "int", pageSizeParam.Type)
	assert.Equal(t, "Number of items per page", pageSizeParam.Description)
	assert.False(t, pageSizeParam.Optional)
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
	assert.True(t, format != nil)

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
func TestLoadIntermediateFormat_MarkdownWithCELLimit(t *testing.T) {
	markdownContent := `# Get user information by ID

## Description

Get user information by ID

## Parameters

` + "```yaml" + `
user_id: int
include_profile: bool
limit: int
` + "```" + `

## SQL

` + "```sql" + `
SELECT 
    u.id, 
    u.name,
    /*# if include_profile */
        p.bio,
        p.avatar_url,
    /*# end */
FROM users u
/*# if include_profile */
    LEFT JOIN profiles p ON u.id = p.user_id
/*# end */
WHERE u.id = /*= user_id */
LIMIT /*= limit != 0 ? limit : 10 */10
` + "```" + `
`

	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.snap.md")
	err := os.WriteFile(tmpFile, []byte(markdownContent), 0644)
	assert.NoError(t, err)

	// Create table info using the correct types from schema.go
	tableInfo := map[string]*TableInfo{
		"users": {
			Name: "users",
			Columns: map[string]*ColumnInfo{
				"id": {
					Name:         "id",
					DataType:     "int",
					IsPrimaryKey: true,
				},
				"name": {
					Name:     "name",
					DataType: "string",
				},
			},
		},
		"profiles": {
			Name: "profiles",
			Columns: map[string]*ColumnInfo{
				"user_id": {
					Name:     "user_id",
					DataType: "int",
				},
				"bio": {
					Name:     "bio",
					DataType: "string",
				},
				"avatar_url": {
					Name:     "avatar_url",
					DataType: "string",
				},
			},
		},
	}

	// Create config
	config := &Config{
		Dialect: "postgres",
	}

	// Parse markdown document first
	file, err := os.Open(tmpFile)
	assert.NoError(t, err)

	defer file.Close()

	doc, err := markdownparser.Parse(file)
	assert.NoError(t, err)

	// Generate intermediate format using GenerateFromMarkdown with table info
	format, err := intermediate.GenerateFromMarkdown(doc, tmpFile, tmpDir, nil, tableInfo, config)
	assert.NoError(t, err)
	assert.True(t, format != nil)

	// Check function name and description
	assert.Equal(t, "get_user_information_by_id", format.FunctionName)
	assert.Equal(t, "Get user information by ID", format.Description)

	// Check parameters
	assert.Equal(t, 3, len(format.Parameters))

	// Check that CEL expressions are properly extracted
	assert.True(t, len(format.CELExpressions) > 0, "Should have CEL expressions")

	// Look for the LIMIT CEL expression
	var limitExprFound bool

	for _, expr := range format.CELExpressions {
		if expr.Expression == "limit != 0 ? limit : 10" {
			limitExprFound = true
			break
		}
	}

	assert.True(t, limitExprFound, "Should find LIMIT CEL expression")

	// Check instructions for proper LIMIT handling
	var limitInstructionFound bool

	for _, instr := range format.Instructions {
		if instr.Op == "EMIT_EVAL" && instr.ExprIndex != nil && *instr.ExprIndex >= 0 {
			// Find the corresponding expression
			if *instr.ExprIndex < len(format.CELExpressions) {
				expr := format.CELExpressions[*instr.ExprIndex]
				if expr.Expression == "limit != 0 ? limit : 10" {
					limitInstructionFound = true
					break
				}
			}
		}
	}

	assert.True(t, limitInstructionFound, "Should find LIMIT instruction with proper expression index")
}
