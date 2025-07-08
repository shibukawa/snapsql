package parsercommon

import (
	"slices"

	pc "github.com/shibukawa/parsercombinator"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	Space      = PrimitiveType("space", tok.WHITESPACE)
	Comment    = PrimitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	ParenOpen  = PrimitiveType("parenOpen", tok.OPENED_PARENS)
	ParenClose = PrimitiveType("parentClose", tok.CLOSED_PARENS)
	Comma      = PrimitiveType("comma", tok.COMMA)
	Dot        = PrimitiveType("Dot", tok.DOT)

	// Primitives
	Number               = PrimitiveType("number", tok.NUMBER)
	String               = PrimitiveType("string", tok.STRING)
	Boolean              = PrimitiveType("boolean", tok.BOOLEAN)
	Null                 = PrimitiveType("null", tok.NULL)
	Literal              = pc.Or(Number, String, Boolean, Null)
	Identifier           = PrimitiveType("identifier", tok.IDENTIFIER)
	ContextualIdentifier = PrimitiveType("contextualIdentifier", tok.CONTEXTUAL_IDENTIFIER)

	// Operators
	Minus = PrimitiveType("minus", tok.MINUS)
	Not   = PrimitiveType("not", tok.NOT)

	NumericOperator = PrimitiveType("numericOperator", tok.PLUS, tok.MINUS, tok.MULTIPLY, tok.DIVIDE)

	// Keywords
	Select = PrimitiveType("select", tok.SELECT)

	SP  = pc.Drop(pc.ZeroOrMore("comment or space", pc.Or(Space, Comment)))
	EOS = pc.EOS[tok.Token]()
)

func WS(token pc.Parser[tok.Token]) pc.Parser[tok.Token] {
	return pc.Seq(
		token,
		pc.ZeroOrMore("comment or space", pc.Or(Space, Comment)),
	)
}

func WS2(token pc.Parser[tok.Token]) pc.Parser[tok.Token] {
	return pc.Seq(
		token,
		pc.Drop(pc.ZeroOrMore("comment or space", pc.Or(Space, Comment))),
	)
}

func FilterSpace(tokens []pc.Token[tok.Token]) []pc.Token[tok.Token] {
	results := make([]pc.Token[tok.Token], 0, len(tokens))
	for _, token := range tokens {
		if token.Val.Type != tok.WHITESPACE && token.Val.Type != tok.BLOCK_COMMENT && token.Val.Type != tok.LINE_COMMENT {
			results = append(results, token)
		}
	}
	return results
}

func PrimitiveType(typeName string, types ...tok.TokenType) pc.Parser[tok.Token] {
	return func(pctx *pc.ParseContext[tok.Token], tokens []pc.Token[tok.Token]) (int, []pc.Token[tok.Token], error) {
		if len(tokens) > 0 && slices.Contains(types, tokens[0].Val.Type) {
			return 1, tokens[:1], nil
		}
		return 0, nil, pc.ErrNotMatch
	}
}

func ToParserToken(tokens []tok.Token) []pc.Token[tok.Token] {
	results := make([]pc.Token[tok.Token], len(tokens))
	for i, token := range tokens {
		pcToken := pc.Token[tok.Token]{
			Type: "raw",
			Pos: &pc.Pos{
				Line:  token.Position.Line,
				Col:   token.Position.Column,
				Index: token.Position.Offset,
			},
			Val: token,
			Raw: token.Value,
		}
		results[i] = pcToken
	}
	return results
}

func ToToken(entities []pc.Token[tok.Token]) []tok.Token {
	results := make([]tok.Token, 0, len(entities))
	for _, entity := range entities {
		results = append(results, entity.Val)
	}
	return results
}

func ToSrc(entities []pc.Token[tok.Token]) string {
	src := make([]byte, 0, 256)
	for _, entity := range entities {
		src = append(src, entity.Raw...)
	}
	return string(src)
}
