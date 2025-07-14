package parserstep6

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestValidateVariables tests the main ValidateVariables function
func TestValidateVariables(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		paramSchema    map[string]any
		environment    map[string]any
		expectedErrors int
	}{
		{
			name: "Valid template with simple variable",
			sql:  "SELECT /*= user.name */'default' FROM users",
			paramSchema: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			environment:    map[string]any{},
			expectedErrors: 0,
		},
		{
			name:        "Template with environment variable",
			sql:         "SELECT * FROM /*$ table */default_table",
			paramSchema: map[string]any{},
			environment: map[string]any{
				"table": "users",
			},
			expectedErrors: 0,
		},
		{
			name: "Template with LIMIT implicit condition",
			sql:  "SELECT * FROM users LIMIT /*= limit */10",
			paramSchema: map[string]any{
				"limit": 5,
			},
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
			schema := &cmn.FunctionDefinition{
				Parameters: tt.paramSchema,
			}
			namespace := cmn.NewNamespace(schema, tt.environment, nil)

			// Validate variables
			perr := &cmn.ParseError{}
			ValidateVariables(parsed, namespace, perr)

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
		paramSchema    map[string]any
		expectedErrors int
	}{
		{
			name: "Valid variable directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= user.name */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramSchema: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			expectedErrors: 0,
		},
		{
			name: "Invalid variable expression",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= undefined.var */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramSchema:    map[string]any{},
			expectedErrors: 1,
		},
		{
			name: "Malformed directive",
			token: tokenizer.Token{
				Type:     tokenizer.BLOCK_COMMENT,
				Value:    "/*= */",
				Position: tokenizer.Position{Line: 1, Column: 1},
			},
			paramSchema:    map[string]any{},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create namespace
			schema := &cmn.FunctionDefinition{
				Parameters: tt.paramSchema,
			}
			namespace := cmn.NewNamespace(schema, map[string]any{}, nil)

			// Validate
			perr := &cmn.ParseError{}
			validateVariableDirective(tt.token, namespace, perr)

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
			// Create namespace
			schema := &cmn.FunctionDefinition{
				Parameters: map[string]any{},
			}
			namespace := cmn.NewNamespace(schema, tt.environment, nil)

			// Validate
			perr := &cmn.ParseError{}
			validateConstDirective(tt.token, namespace, perr)

			if len(perr.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d errors", tt.expectedErrors, len(perr.Errors))
				for i, err := range perr.Errors {
					t.Errorf("  [%d] %v", i, err)
				}
			}
		})
	}
}
