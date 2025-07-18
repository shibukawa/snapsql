package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestCELExtractor(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedExpressions []string
		expectedSimpleVars  []string
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
			expectedExpressions: []string{},
			expectedSimpleVars:  []string{"user_id"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfDirective",
			sql: `/*#
name: getFilteredUsers
function_name: getFilteredUsers
parameters:
  filters: map[string]any
*/
SELECT id, name, email FROM users 
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */`,
			expectedExpressions: []string{"filters.active"},
			expectedSimpleVars:  []string{},
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
SELECT id, status
/*# if include_details */
, last_login, created_at
/*# else */
, last_login
/*# end */
FROM users WHERE id = /*= user_id */123`,
			expectedExpressions: []string{},
			expectedSimpleVars:  []string{"include_details", "user_id"},
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
			expectedExpressions: []string{`user_type == "admin"`, `user_type == "manager"`},
			expectedSimpleVars:  []string{"user_id"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "ForDirective",
			sql: `/*#
name: getUsersWithFields
function_name: getUsersWithFields
parameters:
  additional_fields: array
*/
SELECT 
  id,
  name
  /*# for field : additional_fields */
  , /*= field */
  /*# end */
FROM users`,
			expectedExpressions: []string{},
			expectedSimpleVars:  []string{"additional_fields", "field"},
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
SELECT * FROM (
  /*# for dept : departments */
  SELECT 
    /*= dept.id */ as dept_id,
    /*= dept.name */ as dept_name,
    (
      /*# for emp : dept.employees */
      SELECT /*= emp.id */, /*= emp.name */
      /*# if !for.last */
      UNION ALL
      /*# end */
      /*# end */
    ) as employees
  /*# if !for.last */
  UNION ALL
  /*# end */
  /*# end */
)`,
			expectedExpressions: []string{
				"dept.id", 
				"dept.name", 
				"dept.employees", 
				"emp.id", 
				"emp.name", 
				"!for.last",
			},
			expectedSimpleVars: []string{"departments", "dept", "emp"},
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
			expectedSimpleVars: []string{"min_age", "max_age", "departments", "active"},
			expectedEnvs:       [][]EnvVar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Extract CEL expressions and environments
			extractor := NewCELExtractor()
			extractor.ExtractFromTokens(tokens)

			expressions := extractor.GetExpressions()
			simpleVars := extractor.GetSimpleVars()
			envs := extractor.GetEnvs()

			// Debug output
			t.Logf("Extracted complex expressions (%d):", len(expressions))
			for i, expr := range expressions {
				t.Logf("  %d: %s", i, expr)
			}

			t.Logf("Extracted simple variables (%d):", len(simpleVars))
			for i, v := range simpleVars {
				t.Logf("  %d: %s", i, v)
			}

			t.Logf("Expected complex expressions (%d):", len(tt.expectedExpressions))
			for i, expr := range tt.expectedExpressions {
				t.Logf("  %d: %s", i, expr)
			}

			t.Logf("Expected simple variables (%d):", len(tt.expectedSimpleVars))
			for i, v := range tt.expectedSimpleVars {
				t.Logf("  %d: %s", i, v)
			}

			// Debug token information
			t.Logf("Tokens with directives:")
			for _, token := range tokens {
				if token.Directive != nil {
					t.Logf("  %s: %s", token.Type, token.Value)
					t.Logf("    Directive: %s, Condition: %s", token.Directive.Type, token.Directive.Condition)
				}
			}

			// Verify expressions
			assert.Equal(t, len(tt.expectedExpressions), len(expressions), "Number of complex expressions should match")
			
			// Check that all expected expressions are present
			for _, expected := range tt.expectedExpressions {
				found := false
				for _, actual := range expressions {
					if expected == actual {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected complex expression %s not found", expected)
			}

			// Verify simple variables
			assert.Equal(t, len(tt.expectedSimpleVars), len(simpleVars), "Number of simple variables should match")
			
			// Check that all expected simple variables are present
			for _, expected := range tt.expectedSimpleVars {
				found := false
				for _, actual := range simpleVars {
					if expected == actual {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected simple variable %s not found", expected)
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
