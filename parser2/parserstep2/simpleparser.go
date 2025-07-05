// --- SQL keyword primitive parsers for test coverage ---
package parserstep2

import (
	"slices"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

func when(flag bool, parser pc.Parser[Entity]) pc.Parser[Entity] {
	return func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if !flag {
			return 0, nil, pc.ErrNotMatch
		}
		return parser(pctx, tokens)
	}
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
			tokens[0].Val.spaces = [][]tokenizer.Token{spaces}
			return tokens[:1], nil
		})
}

// LiteralAstNode is a minimal AST node for literals (number/string)
type LiteralNode struct {
	LiteralType tokenizer.TokenType // "NUMBER", "STRING", "BOOLEAN" or "NULL"
	Value       string
	rawTokens   []tokenizer.Token
}

func (l LiteralNode) Type() cmn.NodeType {
	return cmn.LITERAL
}

func (l LiteralNode) Position() tokenizer.Position {
	return l.rawTokens[0].Position
}

func (l LiteralNode) RawTokens() []tokenizer.Token {
	return l.rawTokens
}

func (l LiteralNode) String() string {
	return "Literal:" + l.Value
}

var _ cmn.AstNode = (*LiteralNode)(nil)

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
								LiteralType: o.Type,
								Value:       o.Value,
								rawTokens:   []tokenizer.Token{o},
							},
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

// This parser passes invalid combination like "NOT IS" or "LIKE NOT".
// This check will be done in later step.
func operator() pc.Parser[Entity] {
	p := ws(primitiveType("operator",
		tokenizer.EQUAL, tokenizer.NOT_EQUAL, tokenizer.LESS_THAN, tokenizer.LESS_EQUAL,
		tokenizer.GREATER_THAN, tokenizer.GREATER_EQUAL, tokenizer.PLUS, tokenizer.MINUS,
		tokenizer.MULTIPLY, tokenizer.DIVIDE,
		tokenizer.AND, tokenizer.OR, tokenizer.IN, tokenizer.IS,
		tokenizer.LIKE, tokenizer.ILIKE, tokenizer.RLIKE, tokenizer.REGEXP))

	return pc.Or(
		pc.Seq(pc.Optional(not()), similar(), to()),
		pc.Seq(not(), p),
		pc.Seq(p, not()),
		p,
	)
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

// --- CTE (Common Table Expression) Parser ---

func withClause() pc.Parser[Entity] {
	return pc.SeqWithLabel("with clause",
		pc.ZeroOrMore("leading comment, space", pc.Or(comment(), space())),
		ws(primitiveType("with", tokenizer.WITH)),
		pc.Optional(recursive()))
}

// --- Statement Keyword Parsers (for SELECT) ---

func groupByClause() pc.Parser[Entity] {
	return pc.SeqWithLabel("group by clause",
		ws(primitiveType("group", tokenizer.GROUP)),
		ws(primitiveType("by", tokenizer.BY)))
}

func orderByClause() pc.Parser[Entity] {
	return ws(pc.SeqWithLabel("order by clause",
		ws(primitiveType("order", tokenizer.ORDER)),
		primitiveType("by", tokenizer.BY),
	))
}

// --- Statement Keyword Parsers (for INSERT) ---

func insertIntoStatement() pc.Parser[Entity] {
	return pc.SeqWithLabel("insert into statement",
		ws(primitiveType("insert", tokenizer.INSERT)),
		ws(pc.Optional(primitiveType("into", tokenizer.INTO))))
}

func valuesClause() pc.Parser[Entity] {
	return ws(primitiveType("values", tokenizer.VALUES))
}
func onConflictClause() pc.Parser[Entity] {
	return pc.SeqWithLabel("on conflict clause",
		ws(primitiveType("on", tokenizer.ON)),
		ws(primitiveType("conflict", tokenizer.CONFLICT)))
}

// --- Statement Keyword Parsers (for DELETE) ---

func deleteFromStatement() pc.Parser[Entity] {
	return pc.SeqWithLabel("on conflict clause",
		ws(primitiveType("delete", tokenizer.DELETE)),
		ws(primitiveType("from", tokenizer.FROM)))
}
