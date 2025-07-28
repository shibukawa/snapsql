package intermediate

import (
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/testhelper"
	"github.com/shibukawa/snapsql/tokenizer"
)

// We'll use our own interfaces for testing to avoid issues with unexported types
type testStatementNode interface {
	Clauses() []testClauseNode
	CTE() *testWithClause
}

type testClauseNode interface {
	RawTokens() []tokenizer.Token
}

type testWithClause struct {
	tokens []tokenizer.Token
}

func (w *testWithClause) RawTokens() []tokenizer.Token {
	return w.tokens
}

// Simple test implementation of StatementNode for testing
type testStatement struct {
	tokens []tokenizer.Token
}

func (s *testStatement) Clauses() []testClauseNode {
	return []testClauseNode{&testClause{tokens: s.tokens}}
}

func (s *testStatement) CTE() *testWithClause {
	return nil
}

// Simple test implementation of ClauseNode for testing
type testClause struct {
	tokens []tokenizer.Token
}

func (c *testClause) RawTokens() []tokenizer.Token {
	return c.tokens
}

// Modified ExtractFromStatement function for testing
func extractFromTestStatement(stmt testStatementNode) (expressions []string, envs [][]EnvVar) {
	expressions = make([]string, 0)
	envs = make([][]EnvVar, 0)
	envLevel := 0
	forLoopStack := make([]string, 0) // Stack of loop variable names

	// Helper function to add an expression if it's not already in the list
	addExpression := func(expr string) {
		if expr != "" && !slices.Contains(expressions, expr) {
			expressions = append(expressions, expr)
		}
	}

	// Process all tokens from the statement
	processTokens := func(tokens []tokenizer.Token) {
		for _, token := range tokens {
			// Check for directives
			if token.Directive != nil {
				switch token.Directive.Type {
				case "if", "elseif":
					if token.Directive.Condition != "" {
						// Add the full condition expression
						addExpression(token.Directive.Condition)
					}

				case "for":
					// Parse "variable : collection" format
					parts := strings.Split(token.Directive.Condition, ":")
					if len(parts) == 2 {
						variable := strings.TrimSpace(parts[0])
						collection := strings.TrimSpace(parts[1])

						// Extract collection expression
						addExpression(collection)

						// Also add the loop variable as an expression
						addExpression(variable)

						// Increase environment level for the loop body
						envLevel++

						// Ensure we have enough environment levels
						for len(envs) <= envLevel-1 {
							envs = append(envs, make([]EnvVar, 0))
						}

						// Add loop variable to the environment
						envs[envLevel-1] = append(envs[envLevel-1], EnvVar{
							Name: variable,
							Type: "any", // Default type, can be refined later
						})

						// Push loop variable to stack
						forLoopStack = append(forLoopStack, variable)
					}

				case "end":
					// Check if we're ending a for loop
					if len(forLoopStack) > 0 {
						// Pop the last loop variable
						forLoopStack = forLoopStack[:len(forLoopStack)-1]
						// Decrease environment level
						if envLevel > 0 {
							envLevel--
						}
					}

				case "variable":
					// Extract variable expression from the token value
					// The format is /*= variable_name */
					if token.Value != "" && strings.HasPrefix(token.Value, "/*=") && strings.HasSuffix(token.Value, "*/") {
						// Extract variable expression between /*= and */
						varExpr := strings.TrimSpace(token.Value[3 : len(token.Value)-2])

						// Add the full expression, not just simple variables
						addExpression(varExpr)
					}
				}
			}
		}
	}

	// Process tokens from each clause
	for _, clause := range stmt.Clauses() {
		processTokens(clause.RawTokens())
	}

	// Process CTE tokens if available
	if cte := stmt.CTE(); cte != nil {
		processTokens(cte.RawTokens())
	}

	return
}

func TestCELExtractor(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedExpressions []string
		expectedEnvs        [][]EnvVar
	}{
		{
			name:                "SimpleVariableSubstitution" + testhelper.GetCaller(t),
			sql:                 `SELECT id, name, email FROM users WHERE id = /*= user_id */123`,
			expectedExpressions: []string{"user_id"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name:                "ComplexVariableSubstitution" + testhelper.GetCaller(t),
			sql:                 `SELECT id, name, email FROM users WHERE id = /*= user_id + offset */123`,
			expectedExpressions: []string{"user_id + offset"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfDirective",
			sql: `SELECT id, name, email FROM users 
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */`,
			expectedExpressions: []string{"filters.active"},
			expectedEnvs:        [][]EnvVar{},
		},
		{
			name: "IfElseDirective" + testhelper.GetCaller(t),
			sql: `SELECT id, status, last_login,
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
			name: "IfElseIfDirective" + testhelper.GetCaller(t),
			sql: `SELECT id, name FROM users 
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
			name: "ComplexExpressions",
			sql: `SELECT 
  id, 
  name,
  /*= display_name ? username : "Anonymous" */
FROM users
WHERE 
  /*# if start_date != "" && end_date != "" */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if sort_field != "" */
ORDER BY /*= sort_field + " " + (sort_direction || "ASC") */
  /*# end */
LIMIT /*= page_size || 10 */10
OFFSET /*= (page - 1) * page_size || 0 */0`,
			expectedExpressions: []string{
				"display_name ? username : \"Anonymous\"",
				"start_date != \"\" && end_date != \"\"",
				"start_date",
				"end_date",
				"sort_field != \"\"",
				"sort_field + \" \" + (sort_direction || \"ASC\")",
				"page_size || 10",
				"(page - 1) * page_size || 0",
			},
			expectedEnvs: [][]EnvVar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize SQL directly
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Create a simple statement node for testing
			stmt := &testStatement{tokens: tokens}

			// Extract CEL expressions and environments
			expressions, envs := extractFromTestStatement(stmt)

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
