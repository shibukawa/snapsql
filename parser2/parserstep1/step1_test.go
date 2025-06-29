package parserstep1

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	tokenizer "github.com/shibukawa/snapsql/tokenizer"
)

func tokenize(sql string) ([]tokenizer.Token, error) {
	tok := tokenizer.NewSqlTokenizer(sql, tokenizer.NewSQLiteDialect())
	return tok.AllTokens()
}

// TestValidateParentheses tests various cases of parentheses validation.
func TestValidateParentheses(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid single pair",
			input:   "(a)",
			wantErr: false,
		},
		{
			name:    "unmatched open",
			input:   "(a",
			wantErr: true,
		},
		{
			name:    "unmatched close",
			input:   "a)",
			wantErr: true,
		},
		{
			name:    "nested pairs",
			input:   "((a))",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenize(tt.input)
			assert.NoError(t, err)
			err = validateParentheses(tokens)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSnapSQLDirectives tests SnapSQL directive matching.
func TestValidateSnapSQLDirectives(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// --- if/end ---
		{
			name:    "valid if-end pair",
			input:   "/*# if */ a /*# end */",
			wantErr: false,
		},
		{
			name:    "nested if-end",
			input:   "/*# if */ /*# if */ a /*# end */ /*# end */",
			wantErr: false,
		},
		{
			name:    "unmatched if",
			input:   "/*# if */ a",
			wantErr: true,
		},
		// --- for/end ---
		{
			name:    "valid for-end pair",
			input:   "/*# for item : items */ a /*# end */",
			wantErr: false,
		},
		{
			name:    "nested for-end",
			input:   "/*# for i:items */ /*# for j:items2 */ a /*# end */ /*# end */",
			wantErr: false,
		},
		{
			name:    "unmatched for",
			input:   "/*# for item : items */ a",
			wantErr: true,
		},
		// --- if/for mix ---
		{
			name:    "nested if-for-end",
			input:   "/*# if */ /*# for item : items */ a /*# end */ /*# end */",
			wantErr: false,
		},
		{
			name:    "nested for-if-end",
			input:   "/*# for item : items */ /*# if */ a /*# end */ /*# end */",
			wantErr: false,
		},
		{
			name:    "end order mismatch",
			input:   "/*# if */ /*# for item : items */ a /*# end */",
			wantErr: true,
		},
		// --- else/elseif ---
		{
			name:    "else in if",
			input:   "/*# if */ a /*# else */ b /*# end */",
			wantErr: false,
		},
		{
			name:    "elseif in if",
			input:   "/*# if */ a /*# elseif cond */ b /*# end */",
			wantErr: false,
		},
		{
			name:    "else without if",
			input:   "a /*# else */",
			wantErr: true,
		},
		{
			name:    "elseif without if",
			input:   "a /*# elseif cond */",
			wantErr: true,
		},
		// --- end without open ---
		{
			name:    "unmatched end",
			input:   "a /*# end */",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenize(tt.input)
			assert.NoError(t, err)
			err = validateSnapSQLDirectives(tokens)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
