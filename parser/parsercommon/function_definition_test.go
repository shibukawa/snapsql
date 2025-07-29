package parsercommon

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/stretchr/testify/assert"
)

func TestFunctionDefinition_CommonTypes(t *testing.T) {
	// Set paths to test data directories
	projectRoot := filepath.Join("testdata", "commontype")
	basePath := filepath.Join(projectRoot, "api", "users")

	// Test cases
	tests := []struct {
		name            string
		yamlStr         string
		basePath        string
		projectRootPath string
		wantErr         bool
		check           func(*testing.T, *FunctionDefinition)
	}{
		{
			name: "Common type reference in same directory",
			yamlStr: `
name: GetUser
function_name: getUser
description: Get user information
parameters:
  user: User
  admin: .User
`,
			basePath:        basePath,
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				// Check that User field is expanded
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "int", user["id"], "user.id should be int")
				assert.Equal(t, "string", user["name"], "user.name should be string")
				assert.Equal(t, "string", user["email"], "user.email should be string")
				assert.Equal(t, "api/users/User", def.OriginalParameters["user"])

				// Check that .User is also expanded
				admin, ok := def.Parameters["admin"].(map[string]any)
				assert.True(t, ok, "admin parameter should be a map")
				assert.Equal(t, "int", admin["id"], "admin.id should be int")
				assert.Equal(t, "api/users/User", def.OriginalParameters["admin"])

				// Check that Department is recursively expanded
				dept, ok := user["department"].(string)
				assert.True(t, ok, "user.department should be a string")
				assert.Equal(t, "Department", dept, "user.department should be Department")
			},
		},
		{
			name: "Common type reference in different directory (relative path)",
			yamlStr: `
name: GetRoles
function_name: getRoles
description: Get role information
parameters:
  role: Role
`,
			basePath:        filepath.Join(projectRoot, "api", "roles"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				// Check that Role field is expanded
				role, ok := def.Parameters["role"].(map[string]any)
				assert.True(t, ok, "role parameter should be a map")
				assert.Equal(t, "int", role["id"], "role.id should be int")
				assert.Equal(t, "string", role["name"], "role.name should be string")
				assert.Equal(t, "string[]", role["permissions"], "role.permissions should be string[]")
			},
		},
		{
			name: "Common type reference in project root",
			yamlStr: `
name: GetGlobalType
function_name: getGlobalType
description: Get global type
parameters:
  global: /GlobalType
`,
			basePath:        basePath,
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				// Check that GlobalType field is expanded
				global, ok := def.Parameters["global"].(map[string]any)
				assert.True(t, ok, "global parameter should be a map")
				assert.Equal(t, "int", global["id"], "global.id should be int")
				assert.Equal(t, "string", global["name"], "global.name should be string")
				assert.Equal(t, "datetime", global["created_at"], "global.created_at should be datetime")
			},
		},
		{
			name: "Array type common type reference",
			yamlStr: `
name: GetUsers
function_name: getUsers
description: Get user list
parameters:
  users: User[]
  globals: /GlobalType[]
`,
			basePath:        basePath,
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				// Check that User[] is expanded as array
				users, ok := def.Parameters["users"].([]any)
				assert.True(t, ok, "users parameter should be an array")
				assert.Len(t, users, 1, "users array should have 1 element")

				// Check that array element is expanded User
				userMap, ok := users[0].(map[string]any)
				assert.True(t, ok, "users[0] should be a map")
				assert.Equal(t, "int", userMap["id"], "users[0].id should be int")
				assert.Equal(t, "string", userMap["name"], "users[0].name should be string")
				assert.Equal(t, "string", userMap["email"], "users[0].email should be string")

				// Check that GlobalType[] is also expanded as array
				globals, ok := def.Parameters["globals"].([]any)
				assert.True(t, ok, "globals parameter should be an array")
				assert.Len(t, globals, 1, "globals array should have 1 element")

				// Check that array element is expanded GlobalType
				globalMap, ok := globals[0].(map[string]any)
				assert.True(t, ok, "globals[0] should be a map")
				assert.Equal(t, "int", globalMap["id"], "globals[0].id should be int")
				assert.Equal(t, "string", globalMap["name"], "globals[0].name should be string")
				assert.Equal(t, "datetime", globalMap["created_at"], "globals[0].created_at should be datetime")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := parseFunctionDefinitionFromYAML(tt.yamlStr, tt.basePath, tt.projectRootPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, def)

			if tt.check != nil {
				tt.check(t, def)
			}
		})
	}
}

func TestParseFunctionDefinitionFromSnapSQLDocument(t *testing.T) {
	// Create a sample markdown content
	markdownContent := `---
name: GetUserByID
function_name: getUserById
generators:
  go:
    package: "user"
    output: "./generated/user.go"
---

# Get User By ID

## Overview

This function retrieves user information by ID.

## Parameters

` + "```yaml" + `
id: int
includeDetails: bool
` + "```" + `

## SQL

` + "```sql" + `
SELECT * FROM users WHERE id = :id
` + "```" + `
`

	// Parse the markdown content
	reader := strings.NewReader(markdownContent)
	doc, err := markdownparser.Parse(reader)
	assert.NoError(t, err)

	// Create a FunctionDefinition from the SnapSQLDocument
	projectRoot := filepath.Join("testdata", "commontype")
	basePath := filepath.Join(projectRoot, "api", "users")

	def, err := ParseFunctionDefinitionFromSnapSQLDocument(doc, basePath, projectRoot)
	assert.NoError(t, err)
	assert.NotNil(t, def)

	// Check that the metadata was correctly extracted
	assert.Equal(t, "getUserById", def.FunctionName)

	// Check that the generators were correctly extracted
	assert.NotNil(t, def.Generators)
	assert.Contains(t, def.Generators, "go")
	assert.Equal(t, "user", def.Generators["go"]["package"])
	assert.Equal(t, "./generated/user.go", def.Generators["go"]["output"])

	// Check that the parameters were correctly extracted and ordered
	assert.Len(t, def.Parameters, 2)
	assert.Equal(t, "int", def.Parameters["id"])
	assert.Equal(t, "bool", def.Parameters["includeDetails"])
	assert.Equal(t, []string{"id", "includeDetails"}, def.ParameterOrder)
}

func TestParseFunctionDefinitionFromSnapSQLDocumentWithCommonTypes(t *testing.T) {
	// Create a sample markdown content with common type references
	markdownContent := `---
name: GetUserWithDepartment
function_name: getUserWithDepartment
generators:
  go:
    package: "user"
    output: "./generated/user_with_department.go"
---

# Get User With Department

## Overview

This function retrieves user information with department details.

## Parameters

` + "```yaml" + `
user: User
department: Department
includeDetails: bool
` + "```" + `

## SQL

` + "```sql" + `
SELECT u.*, d.* 
FROM users u
JOIN departments d ON u.department_id = d.id
WHERE u.id = :user.id
` + "```" + `
`

	// Parse the markdown content
	reader := strings.NewReader(markdownContent)
	doc, err := markdownparser.Parse(reader)
	assert.NoError(t, err)

	// Create a FunctionDefinition from the SnapSQLDocument
	projectRoot := filepath.Join("testdata", "commontype")
	basePath := filepath.Join(projectRoot, "api", "users")

	def, err := ParseFunctionDefinitionFromSnapSQLDocument(doc, basePath, projectRoot)
	assert.NoError(t, err)
	assert.NotNil(t, def)

	// Check that the metadata was correctly extracted
	assert.Equal(t, "getUserWithDepartment", def.FunctionName)

	// Check that the generators were correctly extracted
	assert.NotNil(t, def.Generators)
	assert.Contains(t, def.Generators, "go")
	assert.Equal(t, "user", def.Generators["go"]["package"])
	assert.Equal(t, "./generated/user_with_department.go", def.Generators["go"]["output"])

	// Check that the parameters were correctly extracted and ordered
	assert.Len(t, def.Parameters, 3)
	assert.Contains(t, def.Parameters, "user")
	assert.Contains(t, def.Parameters, "department")
	assert.Equal(t, "bool", def.Parameters["includeDetails"])

	// Check parameter order
	assert.Equal(t, []string{"user", "department", "includeDetails"}, def.ParameterOrder)

	// Check that common types were resolved
	user, ok := def.Parameters["user"].(map[string]any)
	assert.True(t, ok, "user parameter should be a map")
	assert.Contains(t, user, "id")
	assert.Contains(t, user, "name")
	assert.Contains(t, user, "email")

	department, ok := def.Parameters["department"].(map[string]any)
	assert.True(t, ok, "department parameter should be a map")
	assert.Contains(t, department, "id")
	assert.Contains(t, department, "name")
	assert.Contains(t, department, "manager_id")
}
