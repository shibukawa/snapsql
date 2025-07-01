package parserstep2

import (
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// ExpressionNode: 式全体をトークン列で保持するASTノード
// NodeType: EXPRESSION
// 復元性重視のため、トークン列をそのまま保持
// String()はトークン列をスペース区切りで連結

type ExpressionNode struct {
	cmn.BaseAstNode
}

func (e *ExpressionNode) Type() cmn.NodeType { return cmn.EXPRESSION }
func (e *ExpressionNode) String() string {
	parts := make([]string, len(e.BaseAstNode.RawTokens()))
	for i, t := range e.BaseAstNode.RawTokens() {
		parts[i] = t.Value
	}
	return strings.Join(parts, " ")
}

var expression pc.Parser[Entity]

func init() {
	primary := pc.Or(
		pc.Trans(
			literal(),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				return []pc.Token[Entity]{
					{
						Type: "expression",
						Pos:  tokens[0].Pos,
						Val: Entity{
							NewValue: &ExpressionNode{
								BaseAstNode: cmn.BaseAstNode{
									NodeType: cmn.EXPRESSION,
									Pos:      tokens[0].Val.Original.Position,
								},
							},
							rawTokens: tokens[0].Val.rawTokens,
						},
					},
				}, nil
			},
		),
		pc.Trans(
			pc.Seq(
				parenOpen(),
				pc.Lazy(func() pc.Parser[Entity] { return expression }),
				parenClose(),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				var allTokens []tokenizer.Token
				allTokens = append(allTokens, tokens[0].Val.Original) // (
				allTokens = append(allTokens, tokens[1].Val.rawTokens...)
				allTokens = append(allTokens, tokens[2].Val.Original) // )
				return []pc.Token[Entity]{
					{
						Type: "expression",
						Pos:  tokens[0].Pos,
						Val: Entity{
							NewValue: &ExpressionNode{
								BaseAstNode: cmn.BaseAstNode{
									NodeType: cmn.EXPRESSION,
									Pos:      tokens[0].Val.Original.Position,
								},
							},
							rawTokens: allTokens,
						},
					},
				}, nil
			},
		),
	)
	expression = pc.Or(
		pc.Trans(
			pc.Seq(
				primary,
				operator(),
				pc.Lazy(func() pc.Parser[Entity] { return expression }),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				var allTokens []tokenizer.Token
				allTokens = append(allTokens, tokens[0].Val.rawTokens...)
				allTokens = append(allTokens, tokens[1].Val.Original) // operator
				allTokens = append(allTokens, tokens[2].Val.rawTokens...)
				return []pc.Token[Entity]{
					{
						Type: "expression",
						Pos:  tokens[0].Pos,
						Val: Entity{
							NewValue: &ExpressionNode{
								BaseAstNode: cmn.BaseAstNode{
									NodeType: cmn.EXPRESSION,
									Pos:      tokens[0].Val.Original.Position,
								},
							},
							rawTokens: allTokens,
						},
					},
				}, nil
			},
		),
		primary,
	)
}
