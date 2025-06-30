package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	"github.com/shibukawa/snapsql/tokenizer"
)

func space() pc.Parser[Entity] {
	return pc.Trace("space", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.WHITESPACE {
			return 1, []pc.Token[Entity]{
				{
					Type: "space",
					Pos:  tokens[0].Pos,
					Val: Entity{
						Original: o,
					},
				},
			}, nil
		}
		return 0, nil, pc.ErrNotMatch
	})
}

func comment() pc.Parser[Entity] {
	return pc.Trace("comment", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.BLOCK_COMMENT || o.Type == tokenizer.LINE_COMMENT {
			return 1, []pc.Token[Entity]{
				{
					Type: "comment",
					Pos:  tokens[0].Pos,
					Val: Entity{
						Original: o,
					},
				},
			}, nil
		}
		return 0, nil, pc.ErrNotMatch
	})
}

// ws means with space. Appends trailing spaces or comments to the token.
func ws(token pc.Parser[Entity]) pc.Parser[Entity] {
	return pc.Trans(
		pc.SeqWithLabel("token with comment or space",
			token,
			pc.ZeroOrMore("comment or space", pc.Or(space(), comment())),
		),
		func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
			var spaces []tokenizer.Token
			for _, t := range tokens[1:] {
				spaces = append(spaces, t.Val.Original)
			}
			tokens[0].Val.spaces = spaces
			return tokens[:1], nil
		})
}

func ifDirective() pc.Parser[Entity] {
	return ws(pc.Trace("if-directive", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == "if" {
			return 1, []pc.Token[Entity]{
				{
					Type: "if-directive",
					Pos:  tokens[0].Pos,
					Val: Entity{
						Original: tokens[0].Val.Original,
					},
				},
			}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func endDirective() pc.Parser[Entity] {
	return ws(pc.Trace("end-directive", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == "end" {
			return 1, []pc.Token[Entity]{
				{
					Type: "end-directive",
					Pos:  tokens[0].Pos,
					Val: Entity{
						Original: tokens[0].Val.Original,
					},
				},
			}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

// --- Primitive Parsers ---

func identifier() pc.Parser[Entity] {
	return ws(pc.Trace("identifier", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.IDENTIFIER {
			return 1, []pc.Token[Entity]{{Type: "identifier", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func number() pc.Parser[Entity] {
	return ws(pc.Trace("number", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.NUMBER {
			return 1, []pc.Token[Entity]{{Type: "number", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func str() pc.Parser[Entity] {
	return ws(pc.Trace("string", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.STRING {
			return 1, []pc.Token[Entity]{{Type: "string", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func paren() pc.Parser[Entity] {
	return ws(pc.Trace("paren", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.OPENED_PARENS || o.Type == tokenizer.CLOSED_PARENS {
			return 1, []pc.Token[Entity]{{Type: "paren", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func comma() pc.Parser[Entity] {
	return ws(pc.Trace("comma", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.COMMA {
			return 1, []pc.Token[Entity]{{Type: "comma", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func semicolon() pc.Parser[Entity] {
	return ws(pc.Trace("semicolon", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		if o.Type == tokenizer.SEMICOLON {
			return 1, []pc.Token[Entity]{{Type: "semicolon", Pos: tokens[0].Pos, Val: Entity{Original: o}}}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func operator() pc.Parser[Entity] {
	return ws(pc.Trace("operator", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		o := tokens[0].Val.Original
		switch o.Type {
		case tokenizer.EQUAL, tokenizer.NOT_EQUAL, tokenizer.LESS_THAN, tokenizer.LESS_EQUAL, tokenizer.GREATER_THAN, tokenizer.GREATER_EQUAL, tokenizer.PLUS, tokenizer.MINUS, tokenizer.MULTIPLY, tokenizer.DIVIDE:
			return 1, []pc.Token[Entity]{
				{
					Type: "operator",
					Pos:  tokens[0].Pos,
					Val:  Entity{Original: o},
					Raw:  o.Value,
				},
			}, nil
		}
		return 0, nil, pc.ErrNotMatch
	}))
}
