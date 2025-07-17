package parserstep2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestConditionalClauseSimple(t *testing.T) {
	sql := `SELECT id, name FROM users /*# if user_id */ WHERE id = 1 /*# end */`

	// Tokenize
	tokens, err := tok.Tokenize(sql)
	assert.NoError(t, err)

	// Convert to Entity tokens
	pcTokens := tokenToEntity(tokens)

	// Find WHERE token
	whereFound := false
	ifDirectiveFound := false
	endDirectiveFound := false

	for _, token := range pcTokens {
		if token.Val.Original.Type == tok.WHERE {
			whereFound = true
		}
		if token.Val.Original.Directive != nil {
			if token.Val.Original.Directive.Type == "if" {
				ifDirectiveFound = true
				assert.Equal(t, "user_id", token.Val.Original.Directive.Condition)
			}
			if token.Val.Original.Directive.Type == "end" {
				endDirectiveFound = true
			}
		}
	}

	assert.True(t, whereFound, "WHERE token should be found")
	assert.True(t, ifDirectiveFound, "if directive should be found")
	assert.True(t, endDirectiveFound, "end directive should be found")

	t.Logf("Test completed successfully. Found WHERE: %v, if directive: %v, end directive: %v",
		whereFound, ifDirectiveFound, endDirectiveFound)
}
