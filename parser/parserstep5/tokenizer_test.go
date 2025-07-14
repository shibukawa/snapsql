package parserstep5

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestTokenizerDirectives(t *testing.T) {
	sql := `SELECT /*= user.name */'default_name' FROM users`

	// Test raw tokenizer output
	tokens, err := tokenizer.Tokenize(sql)
	assert.NoError(t, err)

	t.Logf("Raw tokenizer output for: %s", sql)
	for i, token := range tokens {
		if token.Directive != nil {
			t.Logf("Token[%d]: %s = %q (Directive: %s, Condition: %q)",
				i, token.Type, token.Value, token.Directive.Type, token.Directive.Condition)
		} else {
			t.Logf("Token[%d]: %s = %q", i, token.Type, token.Value)
		}
	}

	// Check if directive token is present
	foundDirective := false
	for _, token := range tokens {
		if token.Directive != nil && token.Directive.Type == "variable" {
			foundDirective = true
			break
		}
	}

	assert.True(t, foundDirective, "Should find variable directive in raw tokens")
}
