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

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		schema         *cmn.FunctionDefinition
		environment    map[string]any
		expectedErrors int // Number of expected errors
	}{
		{
			name: "Valid template with simple variable",
			sql:  "SELECT /*= user.name */default FROM users",
			schema: &cmn.FunctionDefinition{
				Name: "getUserData",
				Parameters: map[string]any{
					"user": map[string]any{
						"name": "str",
						"id":   "int",
					},
				},
			},
			environment:    map[string]any{},
			expectedErrors: 0,
		},
		{
			name: "Invalid template with undefined variable",
			sql:  "SELECT /*= undefined_var */default FROM users",
			schema: &cmn.FunctionDefinition{
				Name:       "getUserData",
				Parameters: map[string]any{},
			},
			environment:    map[string]any{},
			expectedErrors: 1,
		},
		{
			name: "Template with environment variable",
			sql:  "SELECT name FROM /*$ table_name */default_table",
			schema: &cmn.FunctionDefinition{
				Name:       "getTableData",
				Parameters: map[string]any{},
			},
			environment: map[string]any{
				"table_name": "users",
			},
			expectedErrors: 0,
		},
		{
			name: "Template with LIMIT implicit condition",
			sql:  "SELECT name FROM users LIMIT /*= limit */10",
			schema: &cmn.FunctionDefinition{
				Name: "getUsersWithLimit",
				Parameters: map[string]any{
					"limit": "int",
				},
			},
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
			err = parserstep5.Execute(statement)
			assert.NoError(t, err)

			// Create namespace
			namespace := cmn.NewNamespace(tt.schema, tt.environment, nil)

			// Execute parserstep6 (which includes parserstep5 processing)
			parseErr := Execute(statement, namespace)

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
				Name: "test_query",
				Parameters: map[string]any{
					"user_id": map[string]any{
						"type": "int",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Replace DUMMY_LITERAL with string parameter",
			sql:  "SELECT /*= user_name */ FROM users",
			functionDef: cmn.FunctionDefinition{
				Name: "test_query",
				Parameters: map[string]any{
					"user_name": map[string]any{
						"type": "string",
					},
				},
			},
			expectError: false,
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

			parseErr = parserstep5.Execute(stmt)
			if parseErr != nil {
				t.Fatalf("parserstep5 failed: %v", parseErr)
			}

			// Create namespace
			namespace := cmn.NewNamespace(&tt.functionDef, nil, nil)

			// Execute parserstep6 with function definition
			parseErr = ExecuteWithFunctionDef(stmt, namespace, tt.functionDef)

			if tt.expectError {
				assert.True(t, parseErr != nil, "Expected error but got none")
			} else {
				// Check if DUMMY_LITERAL tokens were replaced correctly
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
