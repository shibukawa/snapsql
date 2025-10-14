package parsercommon

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

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
  admin: ./User
`,
			basePath:        basePath,
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// Check that User field is expanded
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "int", user["id"], "user.id should be int")
				assert.Equal(t, "string", user["name"], "user.name should be string")
				assert.Equal(t, "string", user["email"], "user.email should be string")
				assert.Equal(t, "api/users/User", def.OriginalParameters["user"])

				// Check that ./User is also expanded
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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

// TestFunctionDefinition_AncestorDirectorySearch tests the new ancestor directory search functionality
func TestFunctionDefinition_AncestorDirectorySearch(t *testing.T) {
	projectRoot := filepath.Join("testdata", "commontype")

	tests := []struct {
		name            string
		yamlStr         string
		basePath        string
		projectRootPath string
		wantErr         bool
		check           func(*testing.T, *FunctionDefinition)
	}{
		{
			name: "Find User type from nested directory (profiles/settings)",
			yamlStr: `
name: GetUserFromDeepNested
function_name: getUserFromDeepNested
description: Get user from deeply nested directory
parameters:
  user: User
  profile: Profile
  settings: Settings
`,
			basePath:        filepath.Join(projectRoot, "api", "users", "profiles", "settings"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// User should be found from api/users/profiles/_common.yaml (closest ancestor with User definition)
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "int", user["id"], "user.id should be int")
				assert.Equal(t, "string", user["name"], "user.name should be string")
				assert.Equal(t, "string", user["email"], "user.email should be string")
				assert.Equal(t, "int", user["profile_id"], "user.profile_id should be int")
				assert.Equal(t, "api/users/profiles/User", def.OriginalParameters["user"])

				// Profile should be found from api/users/profiles/_common.yaml (current directory's parent)
				profile, ok := def.Parameters["profile"].(map[string]any)
				assert.True(t, ok, "profile parameter should be a map")
				assert.Equal(t, "int", profile["id"], "profile.id should be int")
				assert.Equal(t, "string", profile["bio"], "profile.bio should be string")
				assert.Equal(t, "string", profile["avatar_url"], "profile.avatar_url should be string")
				assert.Equal(t, "api/users/profiles/Profile", def.OriginalParameters["profile"])

				// Settings should be found from current directory
				settings, ok := def.Parameters["settings"].(map[string]any)
				assert.True(t, ok, "settings parameter should be a map")
				assert.Equal(t, "int", settings["id"], "settings.id should be int")
				assert.Equal(t, "string", settings["theme"], "settings.theme should be string")
				assert.Equal(t, "string", settings["language"], "settings.language should be string")
				assert.Equal(t, "api/users/profiles/settings/Settings", def.OriginalParameters["settings"])
			},
		},
		{
			name: "Find GlobalType from nested directory",
			yamlStr: `
name: GetGlobalFromNested
function_name: getGlobalFromNested
description: Get global type from nested directory
parameters:
  global: GlobalType
  user: User
`,
			basePath:        filepath.Join(projectRoot, "api", "users", "profiles"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// GlobalType should be found from project root
				global, ok := def.Parameters["global"].(map[string]any)
				assert.True(t, ok, "global parameter should be a map")
				assert.Equal(t, "int", global["id"], "global.id should be int")
				assert.Equal(t, "string", global["name"], "global.name should be string")
				assert.Equal(t, "datetime", global["created_at"], "global.created_at should be datetime")
				assert.Equal(t, "GlobalType", def.OriginalParameters["global"])

				// User should be found from current directory (api/users/profiles)
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "int", user["id"], "user.id should be int")
				assert.Equal(t, "string", user["name"], "user.name should be string")
				assert.Equal(t, "api/users/profiles/User", def.OriginalParameters["user"])
			},
		},
		{
			name: "Array types with ancestor search",
			yamlStr: `
name: GetArraysFromNested
function_name: getArraysFromNested
description: Get array types from nested directory
parameters:
  users: User[]
  profiles: Profile[]
  globals: GlobalType[]
`,
			basePath:        filepath.Join(projectRoot, "api", "users", "profiles"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// User[] should be found from current directory (api/users/profiles)
				users, ok := def.Parameters["users"].([]any)
				assert.True(t, ok, "users parameter should be an array")
				assert.Len(t, users, 1, "users array should have 1 element")
				userMap, ok := users[0].(map[string]any)
				assert.True(t, ok, "users[0] should be a map")
				assert.Equal(t, "int", userMap["id"], "users[0].id should be int")
				assert.Equal(t, "api/users/profiles/User[]", def.OriginalParameters["users"])

				// Profile[] should be found from current directory
				profiles, ok := def.Parameters["profiles"].([]any)
				assert.True(t, ok, "profiles parameter should be an array")
				assert.Len(t, profiles, 1, "profiles array should have 1 element")
				profileMap, ok := profiles[0].(map[string]any)
				assert.True(t, ok, "profiles[0] should be a map")
				assert.Equal(t, "int", profileMap["id"], "profiles[0].id should be int")
				assert.Equal(t, "api/users/profiles/Profile[]", def.OriginalParameters["profiles"])

				// GlobalType[] should be found from project root
				globals, ok := def.Parameters["globals"].([]any)
				assert.True(t, ok, "globals parameter should be an array")
				assert.Len(t, globals, 1, "globals array should have 1 element")
				globalMap, ok := globals[0].(map[string]any)
				assert.True(t, ok, "globals[0] should be a map")
				assert.Equal(t, "int", globalMap["id"], "globals[0].id should be int")
				assert.Equal(t, "GlobalType[]", def.OriginalParameters["globals"])
			},
		},
		{
			name: "Type not found in any ancestor directory",
			yamlStr: `
name: GetNonExistentType
function_name: getNonExistentType
description: Try to get non-existent type
parameters:
  nonexistent: NonExistentType
`,
			basePath:        filepath.Join(projectRoot, "api", "users", "profiles"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// NonExistentType should remain as string since it's not found
				nonexistent, ok := def.Parameters["nonexistent"].(string)
				assert.True(t, ok, "nonexistent parameter should remain as string")
				assert.Equal(t, "nonexistenttype", nonexistent, "nonexistent should be normalized to lowercase")
				assert.Equal(t, "nonexistenttype", def.OriginalParameters["nonexistent"])
			},
		},
		{
			name: "Mixed path specifications and ancestor search",
			yamlStr: `
name: GetMixedTypes
function_name: getMixedTypes
description: Mix of explicit paths and ancestor search
parameters:
  user: User                    # Should find from ancestor (closest one)
  explicit_user: ./User         # Should find from current directory (if exists) or explicit path
  global_explicit: /GlobalType  # Should find from project root
  profile: Profile              # Should find from ancestor (closest one)
`,
			basePath:        filepath.Join(projectRoot, "api", "users", "profiles"),
			projectRootPath: projectRoot,
			wantErr:         false,
			check: func(t *testing.T, def *FunctionDefinition) {
				t.Helper()
				// User should be found from current directory (api/users/profiles) - closest ancestor
				user, ok := def.Parameters["user"].(map[string]any)
				assert.True(t, ok, "user parameter should be a map")
				assert.Equal(t, "api/users/profiles/User", def.OriginalParameters["user"])
				assert.Equal(t, "int", user["id"], "user.id should be int")

				// explicit_user should use explicit path resolution
				explicitUser, ok := def.Parameters["explicit_user"].(map[string]any)
				assert.True(t, ok, "explicit_user parameter should be a map")
				assert.Equal(t, "api/users/profiles/User", def.OriginalParameters["explicit_user"])
				assert.Equal(t, "int", explicitUser["id"], "explicit_user.id should be int")

				// global_explicit should use explicit path resolution
				globalExplicit, ok := def.Parameters["global_explicit"].(map[string]any)
				assert.True(t, ok, "global_explicit parameter should be a map")
				assert.Equal(t, "GlobalType", def.OriginalParameters["global_explicit"])
				assert.Equal(t, "int", globalExplicit["id"], "global_explicit.id should be int")

				// profile should be found from current directory
				profile, ok := def.Parameters["profile"].(map[string]any)
				assert.True(t, ok, "profile parameter should be a map")
				assert.Equal(t, "api/users/profiles/Profile", def.OriginalParameters["profile"])
				assert.Equal(t, "int", profile["id"], "profile.id should be int")
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

// TestSearchCommonTypeInAncestors tests the searchCommonTypeInAncestors function directly
func TestSearchCommonTypeInAncestors(t *testing.T) {
	projectRoot := filepath.Join("testdata", "commontype")

	tests := []struct {
		name        string
		typeName    string
		isArray     bool
		startDir    string
		wantFound   bool
		wantTypeKey string
		checkResult func(*testing.T, any)
	}{
		{
			name:        "Find User from deeply nested directory",
			typeName:    "User",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles", "settings"),
			wantFound:   true,
			wantTypeKey: "api/users/profiles/User", // Should find the closest ancestor with User definition
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				userMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", userMap["id"])
				assert.Equal(t, "string", userMap["name"])
				assert.Equal(t, "string", userMap["email"])
				assert.Equal(t, "int", userMap["profile_id"]) // This is specific to profiles/User
			},
		},
		{
			name:        "Find User from api/users directory (should find local User)",
			typeName:    "User",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users"),
			wantFound:   true,
			wantTypeKey: "api/users/User", // Should find User from api/users/_common.yaml
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				userMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", userMap["id"])
				assert.Equal(t, "string", userMap["name"])
				assert.Equal(t, "string", userMap["email"])
				// This User has department field as string reference, not expanded
				assert.Equal(t, "Department", userMap["department"])
			},
		},
		{
			name:        "Find Profile from settings directory",
			typeName:    "Profile",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles", "settings"),
			wantFound:   true,
			wantTypeKey: "api/users/profiles/Profile",
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				profileMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", profileMap["id"])
				assert.Equal(t, "string", profileMap["bio"])
				assert.Equal(t, "string", profileMap["avatar_url"])
			},
		},
		{
			name:        "Find Settings from current directory",
			typeName:    "Settings",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles", "settings"),
			wantFound:   true,
			wantTypeKey: "api/users/profiles/settings/Settings",
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				settingsMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", settingsMap["id"])
				assert.Equal(t, "string", settingsMap["theme"])
				assert.Equal(t, "string", settingsMap["language"])
			},
		},
		{
			name:        "Find GlobalType from nested directory",
			typeName:    "GlobalType",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles"),
			wantFound:   true,
			wantTypeKey: "GlobalType",
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				globalMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", globalMap["id"])
				assert.Equal(t, "string", globalMap["name"])
				assert.Equal(t, "datetime", globalMap["created_at"])
			},
		},
		{
			name:        "Find User array from nested directory",
			typeName:    "User",
			isArray:     true,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles"),
			wantFound:   true,
			wantTypeKey: "api/users/profiles/User[]", // Should find the closest ancestor with User definition
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				userArray, ok := result.([]any)
				assert.True(t, ok, "result should be an array")
				assert.Len(t, userArray, 1)
				userMap, ok := userArray[0].(map[string]any)
				assert.True(t, ok, "array element should be a map")
				assert.Equal(t, "int", userMap["id"])
				assert.Equal(t, "string", userMap["name"])
				assert.Equal(t, "int", userMap["profile_id"]) // This is specific to profiles/User
			},
		},
		{
			name:        "Type not found in any ancestor",
			typeName:    "NonExistentType",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles"),
			wantFound:   false,
			wantTypeKey: "",
			checkResult: func(t *testing.T, result any) {
				t.Helper()
				assert.Nil(t, result, "result should be nil when type not found")
			},
		},
		{
			name:        "Find Department from nested directory (should find from api/users)",
			typeName:    "Department",
			isArray:     false,
			startDir:    filepath.Join(projectRoot, "api", "users", "profiles", "settings"),
			wantFound:   true,
			wantTypeKey: "api/users/Department", // Department is only defined in api/users/_common.yaml
			checkResult: func(t *testing.T, result any) {
				t.Helper()

				deptMap, ok := result.(map[string]any)
				assert.True(t, ok, "result should be a map")
				assert.Equal(t, "int", deptMap["id"])
				assert.Equal(t, "string", deptMap["name"])
				assert.Equal(t, "int", deptMap["manager_id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a FunctionDefinition instance for testing
			def := &FunctionDefinition{
				commonTypes:     make(map[string]map[string]map[string]any),
				projectRootPath: projectRoot,
			}

			result, typeKey := def.searchCommonTypeInAncestors(tt.typeName, tt.isArray, tt.startDir)

			if tt.wantFound {
				assert.NotNil(t, result, "result should not be nil when type is found")
				assert.Equal(t, tt.wantTypeKey, typeKey, "type key should match expected")

				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			} else {
				assert.Nil(t, result, "result should be nil when type is not found")
				assert.Equal(t, tt.wantTypeKey, typeKey, "type key should be empty when not found")

				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
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

func TestFunctionDefinition_PerformanceThresholdFromYAML(t *testing.T) {
	def, err := parseFunctionDefinitionFromYAML(`
function_name: sample
parameters:
  id: int
performance:
  slow_query_threshold: 750ms
`, "", "")
	assert.NoError(t, err)
	assert.Equal(t, 750*time.Millisecond, def.SlowQueryThreshold)
}

func TestFunctionDefinition_PerformanceThresholdFromDocument(t *testing.T) {
	doc := &markdownparser.SnapSQLDocument{
		Metadata: map[string]any{
			"function_name": "from_doc",
		},
		Performance: markdownparser.PerformanceSettings{
			SlowQueryThreshold: 2 * time.Second,
		},
	}

	def, err := ParseFunctionDefinitionFromSnapSQLDocument(doc, "", "")
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Second, def.SlowQueryThreshold)
}
