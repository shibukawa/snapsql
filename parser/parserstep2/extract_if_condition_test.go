package parserstep2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestExtractIfConditionUnit(t *testing.T) {
	tests := []struct {
		name              string
		prevClauseSQL     string
		currentClauseSQL  string
		expectedCondition string
	}{
		{
			name:              "Valid if/end pair",
			prevClauseSQL:     `FROM users /*# if user_id */`,
			currentClauseSQL:  `WHERE id = 1 /*# end */`,
			expectedCondition: "user_id",
		},
		{
			name:              "No if directive",
			prevClauseSQL:     `FROM users`,
			currentClauseSQL:  `WHERE id = 1 /*# end */`,
			expectedCondition: "",
		},
		{
			name:              "No end directive",
			prevClauseSQL:     `FROM users /*# if user_id */`,
			currentClauseSQL:  `WHERE id = 1`,
			expectedCondition: "",
		},
		{
			name:              "Both directives missing",
			prevClauseSQL:     `FROM users`,
			currentClauseSQL:  `WHERE id = 1`,
			expectedCondition: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize previous clause
			prevTokens, err := tok.Tokenize(tt.prevClauseSQL)
			assert.NoError(t, err)
			prevEntityTokens := tokenToEntity(prevTokens)

			// Tokenize current clause
			currentTokens, err := tok.Tokenize(tt.currentClauseSQL)
			assert.NoError(t, err)
			currentEntityTokens := tokenToEntity(currentTokens)

			// Find WHERE clause head and body
			var clauseHead, clauseBody []pc.Token[Entity]
			for i, token := range currentEntityTokens {
				if token.Val.Original.Type == tok.WHERE {
					clauseHead = currentEntityTokens[i : i+1]
					clauseBody = currentEntityTokens[i+1:]
					break
				}
			}

			// Extract previous clause body (everything after FROM)
			var prevClauseBody []pc.Token[Entity]
			for i, token := range prevEntityTokens {
				if token.Val.Original.Type == tok.FROM {
					prevClauseBody = prevEntityTokens[i+1:]
					break
				}
			}

			// Test the function
			condition, newClauseBody, newPrevClauseBody := extractIfCondition(clauseHead, clauseBody, prevClauseBody)
			t.Logf("Expected: %q, Got: %q", tt.expectedCondition, condition)
			assert.Equal(t, tt.expectedCondition, condition)

			// Verify that directive tokens are removed when condition is found
			if condition != "" {
				// Check that if directive is removed from previous clause body
				for _, token := range newPrevClauseBody {
					if token.Val.Original.Directive != nil && token.Val.Original.Directive.Type == "if" {
						t.Errorf("if directive should be removed from previous clause body")
					}
				}

				// Check that end directive is removed from current clause body
				for _, token := range newClauseBody {
					if token.Val.Original.Directive != nil && token.Val.Original.Directive.Type == "end" {
						t.Errorf("end directive should be removed from current clause body")
					}
				}
			}
		})
	}
}
