package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

type Entity struct {
	Original  tok.Token   // The original token from the tokenizer
	NewValue  cmn.AstNode // The parsed AST node (can be nil if not yet parsed)
	rawTokens []tok.Token // Tokens that are part of the same row (e.g., SELECT statement)
	spaces    []tok.Token // Tokens that represent spaces or comments before this entity
}

func (e *Entity) RawTokens() []tok.Token {
	result := make([]tok.Token, 0, len(e.rawTokens))
	for _, t := range e.rawTokens {
		result = append(result, t)
	}
	result = append(result, e.spaces...)
	return result
}

func TokenToEntity(tokens []tok.Token) []pc.Token[Entity] {
	results := make([]pc.Token[Entity], 0, len(tokens))
	for _, token := range tokens {
		if token.Type == tok.EOF {
			continue
		}
		entity := Entity{
			Original:  token,
			rawTokens: []tok.Token{token},
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

func entityToToken(entities []pc.Token[Entity]) []tok.Token {
	results := make([]tok.Token, len(entities))
	for i, entity := range entities {
		results[i] = entity.Val.Original
	}
	return results
}
