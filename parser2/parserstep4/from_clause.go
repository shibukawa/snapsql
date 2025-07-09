package parserstep4

import (
	"errors"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// DetectFromClauseSubqueryAliasOmission checks if a FROM clause contains a subquery without an alias.
// If found, it adds an error to perr. No return value.
func FinalizeFromClause(clause *cmn.FromClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()
	parenDepth := 0
	for i, tok := range tokens {
		if tok.Type == tokenizer.OPENED_PARENS {
			// Possible subquery start
			parenDepth++
			// Look ahead for SELECT
			if i+1 < len(tokens) && tokens[i+1].Type == tokenizer.SELECT {
				// Find closing paren
				j := i + 2
				for ; j < len(tokens); j++ {
					if tokens[j].Type == tokenizer.OPENED_PARENS {
						parenDepth++
					}
					if tokens[j].Type == tokenizer.CLOSED_PARENS {
						parenDepth--
						if parenDepth == 0 {
							break
						}
					}
				}
				// After closing paren, expect AS/IDENTIFIER (alias)
				if j+1 >= len(tokens) ||
					(tokens[j+1].Type != tokenizer.IDENTIFIER && tokens[j+1].Type != tokenizer.AS) {
					perr.Add(errors.New("Subquery in FROM clause must have an alias"))
					return
				}
				if tokens[j+1].Type == tokenizer.AS {
					if j+2 >= len(tokens) || tokens[j+2].Type != tokenizer.IDENTIFIER {
						perr.Add(errors.New("Subquery in FROM clause must have an alias after AS"))
						return
					}
				}
			}
		}
	}
}
