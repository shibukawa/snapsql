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
			tokens[0].Val.spaces = [][]tokenizer.Token{spaces}
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
	return ws(pc.Trace("literal",
		pc.Trans(
			pc.Or(str(), number(), boolean(), null()),
			func(pctx *pc.ParseContext[Entity], src []pc.Token[Entity]) (converted []pc.Token[Entity], err error) {
				t := src[0]
				o := t.Val.Original
				return []pc.Token[Entity]{
					{
						Type: "literal",
						Pos:  t.Pos,
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
			})))
}

func directive(name, directiveType string) pc.Parser[Entity] {
	return ws(pc.Trace(name, func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == directiveType {
				return 1, []pc.Token[Entity]{
					{
						Type: name,
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

func ifDirective() pc.Parser[Entity] {
	return directive("if-directive", "if")
}

func endDirective() pc.Parser[Entity] {
	return directive("end-directive", "end")
}

// --- Primitive Parsers ---

func primitiveType(typeName string, types ...tokenizer.TokenType) pc.Parser[Entity] {
	return pc.Trace(typeName, func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(tokens) > 0 && tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if slices.Contains(types, o.Type) {
				return 1, []pc.Token[Entity]{
					{
						Type: typeName,
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original:  o,
							rawTokens: []tokenizer.Token{o},
						},
						Raw: o.Value,
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	})
}

func space() pc.Parser[Entity] {
	return primitiveType("space", tokenizer.WHITESPACE)
}

func comment() pc.Parser[Entity] {
	return primitiveType("comment", tokenizer.BLOCK_COMMENT, tokenizer.LINE_COMMENT)
}

func identifier() pc.Parser[Entity] {
	return primitiveType("identifier", tokenizer.IDENTIFIER)
}

func number() pc.Parser[Entity] {
	return ws(primitiveType("number", tokenizer.NUMBER))
}

func str() pc.Parser[Entity] {
	return ws(primitiveType("string", tokenizer.STRING))
}

// booleanLiteral parses TRUE or FALSE with case-insensitive comparison
func boolean() pc.Parser[Entity] {
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

func comma() pc.Parser[Entity] {
	return ws(primitiveType("comma", tokenizer.COMMA))
}

func semicolon() pc.Parser[Entity] {
	return ws(primitiveType("semicolon", tokenizer.SEMICOLON))
}

func between() pc.Parser[Entity] {
	return ws(primitiveType("between", tokenizer.BETWEEN))
}

// This parser passes invalid combination like "NOT IS" or "LIKE NOT".
// This check will be done in later step.
func operator() pc.Parser[Entity] {
	p := ws(primitiveType("operator",
		tokenizer.EQUAL, tokenizer.NOT_EQUAL, tokenizer.LESS_THAN, tokenizer.LESS_EQUAL,
		tokenizer.GREATER_THAN, tokenizer.GREATER_EQUAL, tokenizer.PLUS, tokenizer.MINUS,
		tokenizer.MULTIPLY, tokenizer.DIVIDE,
		tokenizer.AND, tokenizer.OR, tokenizer.IN, tokenizer.LIKE, tokenizer.IS))

	return pc.Or(
		pc.Seq(ws(not()), p),
		pc.Seq(p, ws(not())),
		p,
	)
}

func andOp() pc.Parser[Entity] {
	return ws(primitiveType("and", tokenizer.AND))
}

func null() pc.Parser[Entity] {
	return ws(primitiveType("null", tokenizer.NULL))
}

func not() pc.Parser[Entity] {
	return ws(primitiveType("not", tokenizer.NOT))
}

func minus() pc.Parser[Entity] {
	return ws(primitiveType("minus", tokenizer.MINUS))
}

func parenOpen() pc.Parser[Entity] {
	return ws(primitiveType("parenOpen", tokenizer.OPENED_PARENS))
}

func parenClose() pc.Parser[Entity] {
	return ws(primitiveType("parenClose", tokenizer.CLOSED_PARENS))
}

// dot parses dot operator without ws wrapper (no spaces allowed)
func dot() pc.Parser[Entity] {
	return ws(primitiveType("dot", tokenizer.DOT))
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
	return ws(
		pc.Trace("column-reference", pc.Or(
			// Qualified column: table.column
			pc.Trans(
				pc.Seq(anyIdentifier(), dot(), anyIdentifier()),
				func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
					tableName := tokens[0].Val.Original.Value
					columnName := tokens[2].Val.Original.Value

					// Collect all raw tokens
					var allTokens []tokenizer.Token
					allTokens = append(allTokens, tokens[0].Val.Original) // table
					allTokens = append(allTokens, tokens[1].Val.Original) // dot
					allTokens = append(allTokens, tokens[2].Val.Original) // column

					return []pc.Token[Entity]{
						{
							Type: "column-reference",
							Pos:  tokens[0].Pos,
							Val: Entity{
								NewValue: &ColumnReferenceNode{
									BaseAstNode: cmn.BaseAstNode{
										NodeType: cmn.COLUMN_REFERENCE,
										Pos:      tokens[0].Val.Original.Position,
									},
									TableName:  tableName,
									ColumnName: columnName,
								},
								rawTokens: allTokens,
							},
							Raw: tableName + "." + columnName,
						},
					}, nil
				},
			),
			// Simple column: column
			pc.Trans(
				anyIdentifier(),
				func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
					columnName := tokens[0].Val.Original.Value

					return []pc.Token[Entity]{
						{
							Type: "column-reference",
							Pos:  tokens[0].Pos,
							Val: Entity{
								NewValue: &ColumnReferenceNode{
									BaseAstNode: cmn.BaseAstNode{
										NodeType: cmn.COLUMN_REFERENCE,
										Pos:      tokens[0].Val.Original.Position,
									},
									TableName:  "", // No table qualification
									ColumnName: columnName,
								},
								rawTokens: []tokenizer.Token{tokens[0].Val.Original},
							},
							Raw: columnName,
						},
					}, nil
				},
			),
		)))
}

// ColumnReferenceNode represents a column reference in SQL (e.g., "col" or "table.col")
type ColumnReferenceNode struct {
	cmn.BaseAstNode
	TableName  string // Optional table qualifier (empty string if not qualified)
	ColumnName string // Column name
}

func (c *ColumnReferenceNode) Type() cmn.NodeType           { return cmn.COLUMN_REFERENCE }
func (c *ColumnReferenceNode) Position() tokenizer.Position { return c.BaseAstNode.Position() }
func (c *ColumnReferenceNode) RawTokens() []tokenizer.Token { return c.BaseAstNode.RawTokens() }
func (c *ColumnReferenceNode) String() string {
	if c.TableName != "" {
		return "ColumnRef:" + c.TableName + "." + c.ColumnName
	}
	return "ColumnRef:" + c.ColumnName
}

func atomic() pc.Parser[Entity] {
	return pc.Seq(
		pc.Optional(pc.Or(minus(), not())),
		pc.Or(literal(), columnReference()),
	)
}
