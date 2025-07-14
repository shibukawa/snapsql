package parserstep5

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestApplyImplicitIfConditions(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		expectedLimit  string
		expectedOffset string
	}{
		{
			name:          "LIMIT with single variable should get implicit condition",
			sql:           "SELECT * FROM users LIMIT /*= limit */10",
			expectedLimit: "limit != null",
		},
		{
			name:           "OFFSET with single variable should get implicit condition",
			sql:            "SELECT * FROM users OFFSET /*= offset */5",
			expectedOffset: "offset != null",
		},
		{
			name:          "LIMIT with explicit if condition should not change",
			sql:           "SELECT * FROM users /*# if limit > 0 */LIMIT /*= limit */10/*# end */",
			expectedLimit: "limit > 0",
		},
		{
			name:          "LIMIT without variable should not get condition",
			sql:           "SELECT * FROM users LIMIT 10",
			expectedLimit: "",
		},
		{
			name:          "LIMIT with multiple variables should not get condition",
			sql:           "SELECT * FROM users LIMIT /*= limit1 */10 + /*= limit2 */5",
			expectedLimit: "",
		},
		{
			name:           "Both LIMIT and OFFSET with variables",
			sql:            "SELECT * FROM users LIMIT /*= limit */10 OFFSET /*= offset */5",
			expectedLimit:  "limit != null",
			expectedOffset: "offset != null",
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

			// Apply implicit if conditions
			ApplyImplicitIfConditions(statement)

			// Check LIMIT clause condition
			for _, clause := range statement.Clauses() {
				switch typedClause := clause.(type) {
				case *parsercommon.LimitClause:
					assert.Equal(t, tt.expectedLimit, typedClause.IfCondition(), "LIMIT clause if condition mismatch")
				case *parsercommon.OffsetClause:
					assert.Equal(t, tt.expectedOffset, typedClause.IfCondition(), "OFFSET clause if condition mismatch")
				}
			}
		})
	}
}

func TestExtractImplicitCondition(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		expected string
	}{
		{
			name: "Single variable directive",
			tokens: []tokenizer.Token{
				{
					Type:  tokenizer.BLOCK_COMMENT,
					Value: "/*= limit */",
					Directive: &tokenizer.Directive{
						Type: "variable",
					},
				},
				{
					Type:  tokenizer.NUMBER,
					Value: "10",
				},
			},
			expected: "limit != null",
		},
		{
			name: "No variable directive",
			tokens: []tokenizer.Token{
				{
					Type:  tokenizer.NUMBER,
					Value: "10",
				},
			},
			expected: "",
		},
		{
			name: "Multiple variable directives",
			tokens: []tokenizer.Token{
				{
					Type:  tokenizer.BLOCK_COMMENT,
					Value: "/*= limit1 */",
					Directive: &tokenizer.Directive{
						Type: "variable",
					},
				},
				{
					Type:  tokenizer.NUMBER,
					Value: "10",
				},
				{
					Type:  tokenizer.BLOCK_COMMENT,
					Value: "/*= limit2 */",
					Directive: &tokenizer.Directive{
						Type: "variable",
					},
				},
				{
					Type:  tokenizer.NUMBER,
					Value: "5",
				},
			},
			expected: "",
		},
		{
			name: "Non-variable directive",
			tokens: []tokenizer.Token{
				{
					Type:  tokenizer.BLOCK_COMMENT,
					Value: "/*$ env.limit */",
					Directive: &tokenizer.Directive{
						Type: "const",
					},
				},
				{
					Type:  tokenizer.IDENTIFIER,
					Value: "default_limit",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractImplicitCondition(tt.tokens)
			assert.Equal(t, tt.expected, result)
		})
	}
}
