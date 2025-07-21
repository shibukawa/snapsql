package intermediate

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// extractFunctionDef extracts function definition from SQL comments
func extractFunctionDef(sql string) *parser.FunctionDefinition {
	// Simple regex to extract function definition from SQL comments
	re := regexp.MustCompile(`/\*#\s*name:\s*([^\n]*)\s*function_name:\s*([^\n]*)\s*parameters:\s*([\s\S]*?)\*/`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) < 4 {
		return nil
	}

	name := strings.TrimSpace(matches[1])
	functionName := strings.TrimSpace(matches[2])
	paramsText := matches[3]

	// Parse parameters
	params := make(map[string]interface{})
	paramLines := strings.Split(paramsText, "\n")
	for _, line := range paramLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		paramName := strings.TrimSpace(parts[0])
		paramType := strings.TrimSpace(parts[1])

		params[paramName] = paramType
	}

	return &parser.FunctionDefinition{
		Name:         name,
		FunctionName: functionName,
		Parameters:   params,
	}
}

func TestCELExtractor(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedExpressions []string
		expectedEnvs        [][]EnvVar
	}{
		{
			name: "SimpleVariableSubstitution",
			sql: `/*#
name: getUserById
function_name: getUserById
parameters:
  user_id: int
*/
SELECT id, name, email FROM users WHERE id = /*= user_id */123`,
			expectedExpressions: []string{"user_id"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfDirective",
			sql: `/*#
name: getFilteredUsers
function_name: getFilteredUsers
parameters:
  filters:
    active: bool
*/
SELECT id, name, email FROM users 
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */`,
			expectedExpressions: []string{"filters.active"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfElseDirective",
			sql: `/*#
name: getUserStatus
function_name: getUserStatus
parameters:
  user_id: int
  include_details: bool
*/
SELECT id, status, last_login,
/*# if include_details */
  created_at
/*# else */
  created_date
/*# end */
FROM users WHERE id = /*= user_id */123`,
			expectedExpressions: []string{"include_details", "user_id"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfElseIfDirective",
			sql: `/*#
name: getUserType
function_name: getUserType
parameters:
  user_id: int
  user_type: string
*/
SELECT id, name FROM users 
WHERE id = /*= user_id */123
/*# if user_type == "admin" */
AND role = 'admin'
/*# elseif user_type == "manager" */
AND role = 'manager'
/*# else */
AND role = 'user'
/*# end */`,
			expectedExpressions: []string{`user_id`, `user_type == "admin"`, `user_type == "manager"`},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "ForDirective",
			sql: `/*#
name: getUsersWithFields
function_name: getUsersWithFields
parameters:
  additional_fields: []
*/
SELECT 
  id,
  name
  /*# for field : additional_fields */
  , /*= field */
  /*# end */
FROM users`,
			expectedExpressions: []string{"additional_fields", "field"},
			expectedEnvs: [][]EnvVar{
				{{Name: "field", Type: "any"}}, // Loop environment
			},
		},
		{
			name: "NestedForDirective",
			sql: `/*#
name: getNestedData
function_name: getNestedData
parameters:
  departments: array
*/
SELECT id, name FROM (
  /*# for dept : departments */
  SELECT 
    /*= dept.id */ as dept_id,
    /*= dept.name */ as dept_name,
    (
      /*# for emp : dept.employees */
      SELECT /*= emp.id */, /*= emp.name */
      /*# if !for.last */
      /*# end */
      /*# end */
    ) as employees
  /*# if !for.last */
  /*# end */
  /*# end */
)`,
			expectedExpressions: []string{
				"departments",
				"dept.id",
				"dept.name",
				"dept.employees",
				"emp.id",
				"emp.name",
				"!for.last",
			},
			expectedEnvs: [][]EnvVar{
				{{Name: "dept", Type: "any"}}, // First loop environment
				{{Name: "emp", Type: "any"}},  // Nested loop environment
			},
		},
		{
			name: "ComplexConditions",
			sql: `/*#
name: getFilteredData
function_name: getFilteredData
parameters:
  min_age: int
  max_age: int
  departments: array
  active: bool
*/
SELECT id, name, age, department FROM users
WHERE 1=1
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */
/*# if max_age > 0 */
AND age <= /*= max_age */65
/*# end */
/*# if departments != null && departments.size() > 0 */
AND department IN (/*= departments */('HR', 'Engineering'))
/*# end */
/*# if active */
AND status = 'active'
/*# end */`,
			expectedExpressions: []string{
				"min_age > 0",
				"max_age > 0",
				"departments != null && departments.size() > 0",
			},
			expectedEnvs: [][]EnvVar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Extract function definition from SQL comments
			funcDef := extractFunctionDef(tt.sql)
			assert.NotEqual(t, nil, funcDef, "Function definition should be extracted from SQL")

			// Parse tokens into a statement with function definition
			stmt, err := parser.Parse(tokens, funcDef, nil)
			assert.NoError(t, err, "Statement should be parsed without errors")

			// Extract CEL expressions and environments using the new function
			expressions, envs := ExtractFromStatement(stmt)

			// Debug output
			t.Logf("Extracted expressions (%d):", len(expressions))
			for i, expr := range expressions {
				t.Logf("  %d: %s", i, expr)
			}

			t.Logf("Expected expressions (%d):", len(tt.expectedExpressions))
			for i, expr := range tt.expectedExpressions {
				t.Logf("  %d: %s", i, expr)
			}

			// Verify expressions - check that all expected expressions are present
			for _, expected := range tt.expectedExpressions {
				assert.True(t, slices.Contains(expressions, expected), "Expected expression %s not found", expected)
			}

			// Verify environments
			assert.Equal(t, len(tt.expectedEnvs), len(envs), "Number of environment levels should match")
			for i, expectedLevel := range tt.expectedEnvs {
				if i < len(envs) {
					assert.Equal(t, len(expectedLevel), len(envs[i]), "Number of variables in environment level should match")
					for j, expectedVar := range expectedLevel {
						if j < len(envs[i]) {
							assert.Equal(t, expectedVar.Name, envs[i][j].Name, "Variable name should match")
							assert.Equal(t, expectedVar.Type, envs[i][j].Type, "Variable type should match")
						}
					}
				}
			}
		})
	}
}
