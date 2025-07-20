package parsercommon

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFunctionDefinition_CommonTypes(t *testing.T) {
	// Set paths to test data directories
	projectRoot := filepath.Join("testdata", "commontype")
	basePath := filepath.Join(projectRoot, "api", "users")
	
	// Test cases
	tests := []struct {
		name           string
		yamlStr        string
		basePath       string
		projectRootPath string
		wantErr        bool
		check          func(*testing.T, *FunctionDefinition)
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
			basePath:       basePath,
			projectRootPath: projectRoot,
			wantErr:        false,
			check: func(t *testing.T, def *FunctionDefinition) {
				// Check that User field is expanded
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "int", user["id"], "user.id should be int")
				assert.Equal(t, "string", user["name"], "user.name should be string")
				assert.Equal(t, "string", user["email"], "user.email should be string")
				
				// Check that .User is also expanded
				admin, ok := def.Parameters["admin"].(map[string]any)
				assert.True(t, ok, "admin parameter should be a map")
				assert.Equal(t, "int", admin["id"], "admin.id should be int")
				
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
			basePath:       filepath.Join(projectRoot, "api", "roles"),
			projectRootPath: projectRoot,
			wantErr:        false,
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
  global: GlobalType
`,
			basePath:       basePath,
			projectRootPath: projectRoot,
			wantErr:        false,
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
  globals: GlobalType[]
`,
			basePath:       basePath,
			projectRootPath: projectRoot,
			wantErr:        false,
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
			def, err := ParseFunctionDefinitionFromYAML(tt.yamlStr, tt.basePath, tt.projectRootPath)
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
