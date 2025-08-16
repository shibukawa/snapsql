package parserstep6

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExecute(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	tests := []struct {
		name           string
		sql            string
		environment    map[string]any
		expectedErrors int // Number of expected errors
	}{
		{
			name: "Valid template with simple variable",
			sql: `
/*#
function_name: get_user_data
parameters:
  user:
    name: str
    id: int
*/
SELECT /*= user.name */default FROM users`,
			environment:    map[string]any{},
			expectedErrors: 0,
		},
		{
			name: "Invalid template with undefined variable",
			sql: `
/*#
function_name: get_user_data
parameters: {}
*/
SELECT /*= undefined_var */default FROM users`,
			environment:    map[string]any{},
			expectedErrors: 1,
		},
		{
			name: "Template with environment variable",
			sql: `
/*#
function_name: get_table_data
parameters: {}
*/
SELECT name FROM /*$ table_name */default_table`,
			environment: map[string]any{
				"table_name": "users",
			},
			expectedErrors: 0,
		},
		{
			name: "Template with LIMIT implicit condition",
			sql: `
/*#
function_name: get_users_with_limit
parameters:
  limit: int
*/
SELECT name FROM users LIMIT /*= limit */10`,
			environment:    map[string]any{},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)
			statement, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(statement)
			assert.NoError(t, err)
			err = parserstep4.Execute(statement)
			assert.NoError(t, err)
			err = parserstep5.Execute(statement, nil)
			assert.NoError(t, err)

			fd, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, ".", ".")
			assert.NoError(t, err)

			// Create namespaces
			paramNs, err := cmn.NewNamespaceFromDefinition(fd)
			assert.NoError(t, err)
			constNs, err := cmn.NewNamespaceFromConstants(tt.environment)
			assert.NoError(t, err)

			// Execute parserstep6 (which includes parserstep5 processing)
			parseErr := Execute(statement, paramNs, constNs)

			// Check expected error count
			if tt.expectedErrors == 0 {
				if parseErr != nil {
					t.Errorf("Did not expect an error but got: %v", parseErr)
				}
			} else {
				if parseErr == nil {
					t.Errorf("Expected %d errors, got 0 errors", tt.expectedErrors)
				} else {
					if len(parseErr.Errors) != tt.expectedErrors {
						t.Errorf("Expected %d errors, got %d errors", tt.expectedErrors, len(parseErr.Errors))
						t.Logf("Validation errors:")

						for i, err := range parseErr.Errors {
							t.Logf("  [%d] %v", i, err)
						}
					}
				}
			}
		})
	}
}

// TestExecuteWithFunctionDef tests DUMMY_LITERAL replacement functionality
func TestExecuteWithFunctionDef(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		functionDef cmn.FunctionDefinition
		expectError bool
	}{
		{
			name: "Replace DUMMY_LITERAL with int parameter",
			sql:  "SELECT /*= user_id */ FROM users",
			functionDef: cmn.FunctionDefinition{
				FunctionName: "test_query",
				Parameters: map[string]any{
					"user_id": map[string]any{
						"type": "int",
					},
				},
			},
			expectError: true, // 変数が見つからない場合はエラーが発生することを期待
		},
		{
			name: "Replace DUMMY_LITERAL with string parameter",
			sql:  "SELECT /*= user_name */ FROM users",
			functionDef: cmn.FunctionDefinition{
				FunctionName: "test_query",
				Parameters: map[string]any{
					"user_name": map[string]any{
						"type": "string",
					},
				},
			},
			expectError: true, // 変数が見つからない場合はエラーが発生することを期待
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize SQL
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Process through parserstep1 to insert DUMMY_LITERAL tokens
			processedTokens, err := parserstep1.Execute(tokens)
			assert.NoError(t, err)

			// Parse through steps 2-5
			stmt, err := parserstep2.Execute(processedTokens)
			assert.NoError(t, err)

			err = parserstep3.Execute(stmt)
			assert.NoError(t, err)

			parseErr := parserstep4.Execute(stmt)
			if parseErr != nil {
				t.Fatalf("parserstep4 failed: %v", parseErr)
			}

			parseErr = parserstep5.Execute(stmt, nil)
			if parseErr != nil {
				t.Fatalf("parserstep5 failed: %v", parseErr)
			}

			// Create namespaces
			paramNs, err := cmn.NewNamespaceFromDefinition(&tt.functionDef)
			if err != nil {
				t.Fatalf("Failed to create namespace from schema: %v", err)
			}

			constNs, err := cmn.NewNamespaceFromConstants(nil)
			if err != nil {
				t.Fatalf("Failed to create namespace from environment: %v", err)
			}

			// Execute parserstep6 with function definition
			parseErr6 := Execute(stmt, paramNs, constNs)

			if tt.expectError {
				assert.True(t, len(parseErr6.Errors) > 0, "Expected error but got none")
			} else {
				assert.True(t, len(parseErr6.Errors) == 0, "Did not expect an error but got: %v", parseErr6)

				dummyLiteralFound := false

				for _, clause := range stmt.Clauses() {
					tokens := clause.RawTokens()
					for _, token := range tokens {
						if token.Type == tokenizer.DUMMY_LITERAL {
							dummyLiteralFound = true

							t.Logf("Found unreplaced DUMMY_LITERAL token: %s", token.Value)
						}
					}
				}

				assert.False(t, dummyLiteralFound, "DUMMY_LITERAL tokens should have been replaced")
			}
		})
	}
}
