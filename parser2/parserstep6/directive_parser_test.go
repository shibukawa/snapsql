package parserstep6

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExtractDirectiveExpressions(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedExpressions map[string]string // directive type -> expected expression
	}{
		{
			name: "IF directive with simple condition",
			sql:  "SELECT * FROM users /*# if user_id != null */ WHERE id = /*= user_id */1 /*# end */",
			expectedExpressions: map[string]string{
				"if":       "user_id != null",
				"variable": "user_id",
			},
		},
		{
			name: "FOR directive with list expression",
			sql:  "SELECT * FROM /*# for table in tables *//*$ table */users/*# end */",
			expectedExpressions: map[string]string{
				"for":   "table in tables",
				"const": "table",
			},
		},
		{
			name: "Complex IF-ELSEIF-ELSE structure",
			sql:  "SELECT * FROM users /*# if status == 'active' */ WHERE status = 'active' /*# elseif status == 'inactive' */ WHERE status = 'inactive' /*# else */ /*# end */",
			expectedExpressions: map[string]string{
				"if":     "status == 'active'",
				"elseif": "status == 'inactive'",
			},
		},
		{
			name: "Variable and const directives",
			sql:  "SELECT /*= field_name */name, /*$ env.table_prefix */user_table FROM users LIMIT /*= limit */10",
			expectedExpressions: map[string]string{
				"variable": "limit", // Last variable directive found
				"const":    "env.table_prefix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Parse statement
			statement, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)

			// Extract directive expressions
			extractor := NewDirectiveExpressionExtractor()
			err = extractor.ExtractDirectiveExpressions(statement)
			assert.NoError(t, err)

			// Collect actual expressions from processed tokens
			actualExpressions := make(map[string]string)
			for _, clause := range statement.Clauses() {
				for _, token := range clause.RawTokens() {
					if token.Directive != nil && token.Directive.Condition != "" {
						actualExpressions[token.Directive.Type] = token.Directive.Condition
					}
				}
			}

			// Verify expected expressions
			for expectedType, expectedExpr := range tt.expectedExpressions {
				actualExpr, found := actualExpressions[expectedType]
				assert.True(t, found, "Expected directive type %s not found", expectedType)
				assert.Equal(t, expectedExpr, actualExpr, "Expression mismatch for directive type %s", expectedType)
			}
		})
	}
}

func TestParseIfExpression(t *testing.T) {
	tests := []struct {
		name        string
		tokenValue  string
		expected    string
		expectError bool
	}{
		{
			name:       "Simple if condition",
			tokenValue: "/*# if user_id != null */",
			expected:   "user_id != null",
		},
		{
			name:       "Complex if condition",
			tokenValue: "/*# if status == 'active' && age > 18 */",
			expected:   "status == 'active' && age > 18",
		},
		{
			name:       "Elseif condition",
			tokenValue: "/*# elseif count > 0 */",
			expected:   "count > 0",
		},
		{
			name:        "Invalid format - missing if keyword",
			tokenValue:  "/*# user_id != null */",
			expectError: true,
		},
		{
			name:        "Invalid format - wrong comment style",
			tokenValue:  "/* if user_id != null */",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewDirectiveExpressionExtractor()
			result, err := extractor.parseIfExpression(tt.tokenValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseForExpression(t *testing.T) {
	tests := []struct {
		name        string
		tokenValue  string
		expected    string
		expectError bool
	}{
		{
			name:       "Simple for expression",
			tokenValue: "/*# for item in items */",
			expected:   "item in items",
		},
		{
			name:       "For with complex list expression",
			tokenValue: "/*# for user in users.filter(u => u.active) */",
			expected:   "user in users.filter(u => u.active)",
		},
		{
			name:        "Invalid format - missing in keyword",
			tokenValue:  "/*# for item items */",
			expectError: true,
		},
		{
			name:        "Invalid format - wrong comment style",
			tokenValue:  "/* for item in items */",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewDirectiveExpressionExtractor()
			result, err := extractor.parseForExpression(tt.tokenValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseVariableExpression(t *testing.T) {
	tests := []struct {
		name        string
		tokenValue  string
		expected    string
		expectError bool
	}{
		{
			name:       "Simple variable",
			tokenValue: "/*= user_id */",
			expected:   "user_id",
		},
		{
			name:       "Complex variable expression",
			tokenValue: "/*= user.profile.name */",
			expected:   "user.profile.name",
		},
		{
			name:       "Variable with function call",
			tokenValue: "/*= users.size() */",
			expected:   "users.size()",
		},
		{
			name:        "Invalid format - wrong marker",
			tokenValue:  "/*$ user_id */",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewDirectiveExpressionExtractor()
			result, err := extractor.parseVariableExpression(tt.tokenValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseConstExpression(t *testing.T) {
	tests := []struct {
		name        string
		tokenValue  string
		expected    string
		expectError bool
	}{
		{
			name:       "Simple const",
			tokenValue: "/*$ table_name */",
			expected:   "table_name",
		},
		{
			name:       "Nested const expression",
			tokenValue: "/*$ env.database.table */",
			expected:   "env.database.table",
		},
		{
			name:        "Invalid format - wrong marker",
			tokenValue:  "/*= table_name */",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewDirectiveExpressionExtractor()
			result, err := extractor.parseConstExpression(tt.tokenValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseForExpressionDetailed(t *testing.T) {
	tests := []struct {
		name          string
		forExpression string
		expectedVar   string
		expectedExpr  string
		expectError   bool
	}{
		{
			name:          "Simple for expression",
			forExpression: "item in items",
			expectedVar:   "item",
			expectedExpr:  "items",
		},
		{
			name:          "Complex for expression",
			forExpression: "user in users.filter(u => u.active)",
			expectedVar:   "user",
			expectedExpr:  "users.filter(u => u.active)",
		},
		{
			name:          "Invalid format - missing in",
			forExpression: "item items",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewDirectiveExpressionExtractor()
			variable, listExpr, err := extractor.ParseForExpression(tt.forExpression)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVar, variable)
				assert.Equal(t, tt.expectedExpr, listExpr)
			}
		})
	}
}
