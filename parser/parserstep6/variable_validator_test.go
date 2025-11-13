package parserstep6

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestValidateVariables tests the main ValidateVariables function
func TestValidateVariables(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		environment    map[string]any
		expectedErrors int
	}{
		{
			name: "Valid template with simple variable",
			sql: `
/*#
name: findUsers
function_name: find_users
parameters:
  user:
    name: string
*/
SELECT /*= user.name */'default' FROM users`,
			environment:    map[string]any{},
			expectedErrors: 0, // 変数が見つからない場合はエラーが発生することを期待
		},
		{
			name: "Template with environment variable",
			sql: `
/*#
name: findUsers
function_name: find_users
*/
SELECT * FROM /*$ table */default_table`,
			environment: map[string]any{
				"table": "users",
			},
			expectedErrors: 0,
		},
		{
			name: "Template with LIMIT implicit condition",
			sql: `
/*#
name: findUsers
function_name: find_users
parameters:
  limit: number
*/
SELECT id, name FROM users LIMIT /*= limit */10`,
			environment:    map[string]any{},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the SQL
			tokens, err := tokenizer.Tokenize(tt.sql)
			if err != nil {
				t.Fatalf("Failed to tokenize SQL: %v", err)
			}

			parsed, err := parserstep2.Execute(tokens)
			if err != nil {
				t.Fatalf("Failed to parse SQL: %v", err)
			}

			// Create namespace
			schema, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, ".", ".")
			assert.NoError(t, err, "Failed to parse function definition from SQL comment")
			// Create namespaces
			paramNs, err := cmn.NewNamespaceFromDefinition(schema)
			if err != nil {
				t.Fatalf("Failed to create namespace from schema: %v", err)
			}

			constNs, err := cmn.NewNamespaceFromConstants(tt.environment)
			if err != nil {
				t.Fatalf("Failed to create namespace from environment: %v", err)
			}

			// Validate variables
			perr := &cmn.ParseError{}
			typeInfo := make(map[string]any)
			validateVariables(parsed, paramNs, constNs, perr, typeInfo)

			if len(perr.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d errors", tt.expectedErrors, len(perr.Errors))

				for i, err := range perr.Errors {
					t.Errorf("  [%d] %v", i, err)
				}
			}
		})
	}
}

// TestExtractExpressionFromDirective tests the expression extraction utility
func TestExtractExpressionFromDirective(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		prefix   string
		suffix   string
		expected string
	}{
		{
			name:     "Variable directive",
			content:  "/*= user.name */",
			prefix:   "/*=",
			suffix:   "*/",
			expected: "user.name",
		},
		{
			name:     "Const directive",
			content:  "/*$ env.table */",
			prefix:   "/*$",
			suffix:   "*/",
			expected: "env.table",
		},
		{
			name:     "Directive with spaces",
			content:  "/*=  user.age  */",
			prefix:   "/*=",
			suffix:   "*/",
			expected: "user.age",
		},
		{
			name:     "Invalid prefix",
			content:  "/*! user.name */",
			prefix:   "/*=",
			suffix:   "*/",
			expected: "",
		},
		{
			name:     "Empty expression",
			content:  "/*= */",
			prefix:   "/*=",
			suffix:   "*/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExpressionFromDirective(tt.content, tt.prefix, tt.suffix)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestValidateVariableDirective tests variable directive validation
func TestValidateVariableDirective(t *testing.T) {
	tests := []struct {
		name           string
		token          tokenizer.Token
		paramYAML      string
		expectedErrors int
	}{
		{
			name: "Valid variable directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= user.name */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramYAML:      "  user:\n    name: string\n",
			expectedErrors: 0,
		},
		{
			name: "Invalid variable expression",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= undefined.var */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramYAML:      "",
			expectedErrors: 1,
		},
		{
			name: "Malformed directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramYAML:      "",
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramNs := newNamespaceForTest(t, tt.paramYAML)
			perr := &cmn.ParseError{}
			validateVariableDirective(tt.token, paramNs, perr)

			if len(perr.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d errors", tt.expectedErrors, len(perr.Errors))

				for i, err := range perr.Errors {
					t.Errorf("  [%d] %v", i, err)
				}
			}
		})
	}
}

// TestValidateConstDirective tests const directive validation
func TestValidateConstDirective(t *testing.T) {
	tests := []struct {
		name           string
		token          tokenizer.Token
		environment    map[string]any
		expectedErrors int
	}{
		{
			name: "Valid const directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*$ table */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			environment: map[string]any{
				"table": "users",
			},
			expectedErrors: 0,
		},
		{
			name: "Undefined environment variable",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*$ undefined */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			environment:    map[string]any{},
			expectedErrors: 1,
		},
		{
			name: "Malformed directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*$ */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			environment:    map[string]any{},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create namespaces
			constNs, err := cmn.NewNamespaceFromConstants(tt.environment)
			if err != nil {
				t.Fatalf("Failed to create namespace from environment: %v", err)
			}
			// Validate
			perr := &cmn.ParseError{}
			validateConstDirective(tt.token, constNs, perr)

			if len(perr.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d errors", tt.expectedErrors, len(perr.Errors))

				for i, err := range perr.Errors {
					t.Errorf("  [%d] %v", i, err)
				}
			}
		})
	}
}

func newNamespaceForTest(t *testing.T, paramYAML string) *cmn.Namespace {
	t.Helper()

	var b strings.Builder
	b.WriteString("/*#\nname: test\nfunction_name: test\n")

	if paramYAML != "" {
		b.WriteString("parameters:\n")
		b.WriteString(paramYAML)
	}

	b.WriteString("*/\nSELECT 1")

	tokens, err := tokenizer.Tokenize(b.String())
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	fd, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, ".", ".")
	if err != nil {
		t.Fatalf("parse function definition failed: %v", err)
	}

	paramNs, err := cmn.NewNamespaceFromDefinition(fd)
	if err != nil {
		t.Fatalf("Failed to create namespace from schema: %v", err)
	}

	return paramNs
}
