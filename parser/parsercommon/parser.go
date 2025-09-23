package parsercommon

import (
	"slices"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	// Space parses a whitespace token.
	Space = PrimitiveType("space", tok.WHITESPACE)
	// Comment parses block or line comments.
	Comment = PrimitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	// ParenOpen parses an opening parenthesis.
	ParenOpen = PrimitiveType("parenOpen", tok.OPENED_PARENS)
	// ParenClose parses a closing parenthesis.
	ParenClose = PrimitiveType("parentClose", tok.CLOSED_PARENS)
	// Comma parses a comma delimiter.
	Comma = PrimitiveType("comma", tok.COMMA)
	// Dot parses a dot token.
	Dot = PrimitiveType("Dot", tok.DOT)

	// Number parses a numeric literal.
	Number = PrimitiveType("number", tok.NUMBER)
	// String parses a string literal.
	String = PrimitiveType("string", tok.STRING)
	// Boolean parses a boolean literal.
	Boolean = PrimitiveType("boolean", tok.BOOLEAN)
	// Null parses a NULL literal.
	Null = PrimitiveType("null", tok.NULL)
	// DummyLiteral parses a SnapSQL dummy literal placeholder.
	DummyLiteral = PrimitiveType("dummy_literal", tok.DUMMY_LITERAL)
	// Literal parses any primitive literal or dummy literal.
	Literal = pc.Or(Number, String, Boolean, Null, DummyLiteral)
	// Identifier parses any identifier or contextual identifier.
	Identifier = PrimitiveType("identifier", tok.IDENTIFIER, tok.RESERVED_IDENTIFIER, tok.CONTEXTUAL_IDENTIFIER)
	// ContextualIdentifier parses a contextual identifier token.
	ContextualIdentifier = PrimitiveType("contextualIdentifier", tok.CONTEXTUAL_IDENTIFIER)

	// Minus parses a minus operator token.
	Minus = PrimitiveType("minus", tok.MINUS)
	// Not parses a NOT operator token.
	Not = PrimitiveType("not", tok.NOT)

	// NumericOperator parses arithmetic operator tokens.
	NumericOperator = PrimitiveType("numericOperator", tok.PLUS, tok.MINUS, tok.MULTIPLY, tok.DIVIDE)

	// Select parses a SELECT keyword token.
	Select = PrimitiveType("select", tok.SELECT)
	// Keyword parses any reserved keyword token.
	Keyword = PrimitiveType("keyword", tok.RESERVED_IDENTIFIER)

	// SP consumes zero or more space/comment tokens.
	SP = pc.Drop(pc.ZeroOrMore("comment or space", pc.Or(Space, Comment)))
	// EOS matches end of stream.
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

func KeywordType(typeName string, word ...string) pc.Parser[tok.Token] {
	return func(pctx *pc.ParseContext[tok.Token], tokens []pc.Token[tok.Token]) (int, []pc.Token[tok.Token], error) {
		if len(tokens) > 0 && (tokens[0].Val.Type == tok.IDENTIFIER || tokens[0].Val.Type == tok.RESERVED_IDENTIFIER) {
			v := tokens[0].Val.Value
			for _, w := range word {
				if strings.EqualFold(v, w) {
					return 1, tokens[:1], nil
				}
			}
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
