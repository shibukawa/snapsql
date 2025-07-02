package parserstep2

import (
	"slices"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

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

// LiteralAstNode is a minimal AST node for literals (number/string)
type LiteralNode struct {
	cmn.BaseAstNode
	LiteralType tokenizer.TokenType // "NUMBER" or "STRING"
	Value       string
}

func (l LiteralNode) Type() cmn.NodeType           { return cmn.LITERAL }
func (l LiteralNode) Position() tokenizer.Position { return l.BaseAstNode.Position() }
func (l LiteralNode) RawTokens() []tokenizer.Token { return l.BaseAstNode.RawTokens() }
func (l LiteralNode) String() string               { return "Literal:" + l.Value }

// literal parses numeric, string, boolean, or null literals and returns a LiteralAstNode.
func literal() pc.Parser[Entity] {
	return ws(pc.Or(
		// Number and string literals
		pc.Trace("number-string-literal", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
			if tokens[0].Type == "raw" {
				o := tokens[0].Val.Original
				switch o.Type {
				case tokenizer.NUMBER, tokenizer.STRING:
					return 1, []pc.Token[Entity]{
						{
							Type: "literal",
							Pos:  tokens[0].Pos,
							Val: Entity{
								NewValue: &LiteralNode{
									BaseAstNode: cmn.BaseAstNode{
										NodeType: cmn.LITERAL,
										Pos:      o.Position,
									},
									LiteralType: o.Type,
									Value:       o.Value,
								},
								rawTokens: []tokenizer.Token{o},
							},
							Raw: o.Value,
						},
					}, nil
				}
			}
			return 0, nil, pc.ErrNotMatch
		}),
		// Boolean literals (TRUE/FALSE)
		pc.Trace("boolean-literal", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
			if len(tokens) > 0 && tokens[0].Type == "raw" {
				o := tokens[0].Val.Original
				if (o.Type == tokenizer.RESERVED_IDENTIFIER || o.Type == tokenizer.IDENTIFIER) &&
					(strings.EqualFold(o.Value, "TRUE") || strings.EqualFold(o.Value, "FALSE")) {
					return 1, []pc.Token[Entity]{
						{
							Type: "literal",
							Pos:  tokens[0].Pos,
							Val: Entity{
								NewValue: &LiteralNode{
									BaseAstNode: cmn.BaseAstNode{
										NodeType: cmn.LITERAL,
										Pos:      o.Position,
									},
									LiteralType: o.Type, // IDENTIFIER or RESERVED_IDENTIFIER for boolean
									Value:       o.Value,
								},
								rawTokens: []tokenizer.Token{o},
							},
							Raw: o.Value,
						},
					}, nil
				}
			}
			return 0, nil, pc.ErrNotMatch
		}),
		// NULL literal
		pc.Trace("null-literal", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
			if len(tokens) > 0 && tokens[0].Type == "raw" {
				o := tokens[0].Val.Original
				if o.Type == tokenizer.RESERVED_IDENTIFIER && strings.EqualFold(o.Value, "NULL") {
					return 1, []pc.Token[Entity]{
						{
							Type: "literal",
							Pos:  tokens[0].Pos,
							Val: Entity{
								NewValue: &LiteralNode{
									BaseAstNode: cmn.BaseAstNode{
										NodeType: cmn.LITERAL,
										Pos:      o.Position,
									},
									LiteralType: o.Type, // RESERVED_IDENTIFIER for null
									Value:       o.Value,
								},
								rawTokens: []tokenizer.Token{o},
							},
							Raw: o.Value,
						},
					}, nil
				}
			}
			return 0, nil, pc.ErrNotMatch
		}),
	))
}

func ifDirective() pc.Parser[Entity] {
	return ws(pc.Trace("if-directive", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == "if" {
				return 1, []pc.Token[Entity]{
					{
						Type: "if-directive",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value, // Store the raw comment text
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func endDirective() pc.Parser[Entity] {
	return ws(pc.Trace("end-directive", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == "end" {
				return 1, []pc.Token[Entity]{
					{
						Type: "end-directive",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value, // Store the raw comment text
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

// --- Primitive Parsers ---

func primitiveType(tokens []pc.Token[Entity], typeName string, types ...tokenizer.TokenType) (int, []pc.Token[Entity], error) {
	if len(tokens) > 0 && tokens[0].Type == "raw" {
		o := tokens[0].Val.Original
		if slices.Contains(types, o.Type) {
			return 1, []pc.Token[Entity]{
				{
					Type: typeName,
					Pos:  tokens[0].Pos,
					Val: Entity{
						Original: o,
					},
					Raw: o.Value, // Store the raw comment text
				},
			}, nil
		}
	}
	return 0, nil, pc.ErrNotMatch
}

func space() pc.Parser[Entity] {
	return pc.Trace("space", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "space", tokenizer.WHITESPACE)
	})
}

func comment() pc.Parser[Entity] {
	return pc.Trace("comment", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "comment", tokenizer.BLOCK_COMMENT, tokenizer.LINE_COMMENT)
	})
}

func identifier() pc.Parser[Entity] {
	return pc.Trace("identifier", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "identifier", tokenizer.IDENTIFIER)
	})
}

func number() pc.Parser[Entity] {
	return ws(pc.Trace("number", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "number", tokenizer.NUMBER)
	}))
}

func str() pc.Parser[Entity] {
	return ws(pc.Trace("string", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "string", tokenizer.STRING)
	}))
}

func comma() pc.Parser[Entity] {
	return ws(pc.Trace("comma", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "comma", tokenizer.COMMA)
	}))
}

func semicolon() pc.Parser[Entity] {
	return ws(pc.Trace("semicolon", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "semicolon", tokenizer.SEMICOLON)
	}))
}

func operator() pc.Parser[Entity] {
	return ws(pc.Trace("operator", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "operator",
			tokenizer.EQUAL, tokenizer.NOT_EQUAL, tokenizer.LESS_THAN, tokenizer.LESS_EQUAL,
			tokenizer.GREATER_THAN, tokenizer.GREATER_EQUAL, tokenizer.PLUS, tokenizer.MINUS,
			tokenizer.MULTIPLY, tokenizer.DIVIDE)
	}))
}

func parenOpen() pc.Parser[Entity] {
	return ws(pc.Trace("parenOpen", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "parenOpen", tokenizer.OPENED_PARENS)
	}))
}

func parenClose() pc.Parser[Entity] {
	return ws(pc.Trace("parenClose", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "parenClose", tokenizer.CLOSED_PARENS)
	}))
}

// dot parses dot operator without ws wrapper (no spaces allowed)
func dot() pc.Parser[Entity] {
	return pc.Trace("dot", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		return primitiveType(tokens, "dot", tokenizer.DOT)
	})
}

// anyIdentifier parses any valid identifier (including contextual and quoted)
func anyIdentifier() pc.Parser[Entity] {
	return ws(pc.Trace("any-identifier", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(tokens) > 0 && tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.IDENTIFIER ||
				o.Type == tokenizer.CONTEXTUAL_IDENTIFIER {
				return 1, []pc.Token[Entity]{
					{
						Type: "identifier",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value,
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

// Update columnReference to use anyIdentifier
func columnReference() pc.Parser[Entity] {
	return ws(pc.Trans(
		pc.SeqWithLabel("column-reference",
			pc.Optional(
				pc.Seq(
					anyIdentifier(), // Use anyIdentifier for table name
					dot(),
				),
			),
			anyIdentifier(), // Use anyIdentifier for column name
		),
		func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
			rawTokens := make([]tokenizer.Token, len(tokens))
			// Handle optional table qualifier
			for i, token := range tokens {
				rawTokens[i] = token.Val.Original
			}
			return []pc.Token[Entity]{
				{
					Type: "column-reference",
					Pos:  tokens[0].Pos,
					Val: Entity{
						rawTokens: rawTokens,
					},
					Raw: joinRawTokens(rawTokens),
				},
			}, nil
		}))
}

// Helper function to join raw tokens into a string
func joinRawTokens(tokens []tokenizer.Token) string {
	var result string
	for i, token := range tokens {
		if i > 0 {
			result += " "
		}
		result += token.Value
	}
	return result
}

// keyword parses specific SQL keywords with case-insensitive comparison
func keyword(word string) pc.Parser[Entity] {
	return ws(pc.Trace("keyword-"+word, func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(tokens) > 0 && tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.RESERVED_IDENTIFIER && strings.EqualFold(o.Value, word) { // Case-insensitive comparison
				return 1, []pc.Token[Entity]{
					{
						Type: "keyword",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value, // Preserve original case
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

// booleanLiteral parses TRUE or FALSE with case-insensitive comparison
func booleanLiteral() pc.Parser[Entity] {
	return ws(pc.Trace("boolean", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(tokens) > 0 && tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if (o.Type == tokenizer.RESERVED_IDENTIFIER || o.Type == tokenizer.IDENTIFIER) &&
				(strings.EqualFold(o.Value, "TRUE") || strings.EqualFold(o.Value, "FALSE")) { // Case-insensitive
				return 1, []pc.Token[Entity]{
					{
						Type: "boolean",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value, // Preserve original case
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

// nullLiteral parses NULL with case-insensitive comparison
func nullLiteral() pc.Parser[Entity] {
	return ws(pc.Trace("null", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(tokens) > 0 && tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.RESERVED_IDENTIFIER && strings.EqualFold(o.Value, "NULL") { // Case-insensitive
				return 1, []pc.Token[Entity]{
					{
						Type: "null",
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value, // Preserve original case
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}
