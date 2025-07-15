package parserstep6

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
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
