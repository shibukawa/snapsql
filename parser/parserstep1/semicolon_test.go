package parserstep1

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestProcessSemicolons(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []tok.Token
		expectError bool
		errorType   error
		expectCount int // expected number of tokens after processing
	}{
		{
			name: "no semicolon",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
			},
			expectError: false,
			expectCount: 3,
		},
		{
			name: "trailing semicolon only",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
				{Type: tok.SEMICOLON, Value: ";"},
			},
			expectError: false,
			expectCount: 3, // semicolon should be removed
		},
		{
			name: "trailing semicolon with whitespace",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.WHITESPACE, Value: "\n"},
			},
			expectError: false,
			expectCount: 5, // semicolon removed, whitespace remains
		},
		{
			name: "trailing semicolon with line comment",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.LINE_COMMENT, Value: "-- this is a comment"},
			},
			expectError: false,
			expectCount: 5, // semicolon removed, comment remains
		},
		{
			name: "trailing semicolon with block comment",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.BLOCK_COMMENT, Value: "/* regular comment */"},
			},
			expectError: false,
			expectCount: 5, // semicolon removed, comment remains
		},
		{
			name: "semicolon in middle",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.IDENTIFIER, Value: "id"},
			},
			expectError: true,
			errorType:   ErrSemicolonNotAtEnd,
		},
		{
			name: "multiple semicolons",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.SEMICOLON, Value: ";"},
			},
			expectError: true,
			errorType:   ErrSemicolonNotAtEnd,
		},
		{
			name: "semicolon followed by directive",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.IDENTIFIER, Value: "id"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.BLOCK_COMMENT, Value: "/*# if condition */"},
			},
			expectError: true,
			errorType:   ErrSemicolonNotAtEnd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processSemicolons(tt.tokens)

			if tt.expectError {
				assert.Error(t, err)

				if tt.errorType != nil {
					assert.Contains(t, err.Error(), tt.errorType.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCount, len(result))
			}
		})
	}
}

func TestIsSemicolonAtEnd(t *testing.T) {
	tests := []struct {
		name           string
		tokens         []tok.Token
		semicolonIndex int
		expected       bool
	}{
		{
			name: "semicolon at very end",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
			},
			semicolonIndex: 1,
			expected:       true,
		},
		{
			name: "semicolon followed by whitespace",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.WHITESPACE, Value: " "},
				{Type: tok.WHITESPACE, Value: "\n"},
			},
			semicolonIndex: 1,
			expected:       true,
		},
		{
			name: "semicolon followed by regular comment",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.BLOCK_COMMENT, Value: "/* regular comment */"},
			},
			semicolonIndex: 1,
			expected:       true,
		},
		{
			name: "semicolon followed by directive",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.BLOCK_COMMENT, Value: "/*# if condition */"},
			},
			semicolonIndex: 1,
			expected:       false,
		},
		{
			name: "semicolon followed by EOF",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.EOF, Value: ""},
			},
			semicolonIndex: 1,
			expected:       true,
		},
		{
			name: "semicolon followed by identifier",
			tokens: []tok.Token{
				{Type: tok.IDENTIFIER, Value: "SELECT"},
				{Type: tok.SEMICOLON, Value: ";"},
				{Type: tok.IDENTIFIER, Value: "FROM"},
			},
			semicolonIndex: 1,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSemicolonAtEnd(tt.tokens, tt.semicolonIndex)
			assert.Equal(t, tt.expected, result)
		})
	}
}
