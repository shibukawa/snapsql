package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	"github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

type Entity struct {
	Original  tokenizer.Token      // The original token from the tokenizer
	NewValue  parsercommon.AstNode // The parsed AST node (can be nil if not yet parsed)
	rawTokens []tokenizer.Token    // Tokens that are part of the same row (e.g., SELECT statement)
	spaces    [][]tokenizer.Token  // Tokens that represent spaces or comments before this entity
}

func (e *Entity) RawTokens() []tokenizer.Token {
	var result []tokenizer.Token
	for i, t := range e.rawTokens {
		result = append(result, t)
		for _, space := range e.spaces[i] {
			result = append(result, space)
		}
	}
	return result
}

func TokenToEntity(tokens []tokenizer.Token) []pc.Token[Entity] {
	results := make([]pc.Token[Entity], 0, len(tokens))
	for _, token := range tokens {
		if token.Type == tokenizer.EOF {
			continue
		}
		entity := Entity{
			Original:  token,
			rawTokens: []tokenizer.Token{token},
		}
		pcToken := pc.Token[Entity]{
			Type: "raw",
			Pos: &pc.Pos{
				Line:  token.Position.Line,
				Col:   token.Position.Column,
				Index: token.Position.Offset,
			},
			Val: entity,
			Raw: token.Value,
		}
		results = append(results, pcToken)
	}
	return results
}
