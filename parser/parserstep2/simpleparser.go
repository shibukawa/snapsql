// --- SQL keyword primitive parsers for test coverage ---
package parserstep2

import (
	"slices"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
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
			pc.ZeroOrMore("comment or space", pc.Or(space, comment)),
		),
		func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
			spaceStart := len(tokens)
			for i := len(tokens) - 1; i >= 1; i-- {
				if tokens[i].Type == "comment" || tokens[i].Type == "space" {
					spaceStart--
				}
			}

			tokens[spaceStart-1].Val.spaces = entityToToken(tokens[spaceStart:])
			return tokens[:spaceStart], nil
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

// anyIdentifier parses any valid identifier (including contextual and quoted)
var anyIdentifier = ws(
	pc.Trace("any-identifier", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
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

var (
	// --- CTE (Common Table Expression) Parser ---
	withClause = pc.SeqWithLabel("with clause",
		pc.ZeroOrMore("leading comment, space", pc.Or(comment, space)),
		ws(primitiveType("with", tokenizer.WITH)),
		pc.Optional(recursive))

	// --- Statement Keyword Parsers (for SELECT) ---

	groupByClause = pc.SeqWithLabel("group by clause",
		ws(primitiveType("group", tokenizer.GROUP)),
		ws(primitiveType("by", tokenizer.BY)))

	orderByClause = ws(pc.SeqWithLabel("order by clause",
		ws(primitiveType("order", tokenizer.ORDER)),
		primitiveType("by", tokenizer.BY),
	))

	// --- Statement Keyword Parsers (for INSERT) ---
	insertIntoStatement = pc.SeqWithLabel("insert into statement",
		ws(primitiveType("insert", tokenizer.INSERT)),
		ws(pc.Optional(primitiveType("into", tokenizer.INTO))))

	// valuesClause
	valuesClause = ws(primitiveType("values", tokenizer.VALUES))

	onConflictClause = pc.SeqWithLabel("on conflict clause",
		ws(primitiveType("on", tokenizer.ON)),
		ws(primitiveType("conflict", tokenizer.CONFLICT)))

	deleteFromStatement = pc.SeqWithLabel("delete from clause",
		ws(primitiveType("delete", tokenizer.DELETE)),
		ws(primitiveType("from", tokenizer.FROM)))

	// --- Statement Keyword Parsers (for DELETE) ---
)
