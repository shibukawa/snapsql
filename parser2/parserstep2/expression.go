package parserstep2

import (
	"log"
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
var expressionAlias pc.Parser[Entity]

func init() {
	var expressionBody func(pc.Parser[Entity]) pc.Parser[Entity]
	expressionBody, expressionAlias = pc.NewAlias[Entity]("expression")
	expression = expressionBody(
		pc.Or(
			pc.Trans(
				pc.Seq(expressionAlias, operator(), expressionAlias),
				func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
					var allTokens []tokenizer.Token
					log.Println(tokens[0].Val.NewValue.RawTokens())
					log.Println(tokens[1].Val.NewValue.RawTokens())
					allTokens = append(allTokens, tokens[0].Val.NewValue.RawTokens()...)
					allTokens = append(allTokens, tokens[1].Val.Original) // operator
					allTokens = append(allTokens, tokens[2].Val.NewValue.RawTokens()...)
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
								rowTokens: allTokens,
							},
						},
					}, nil
				},
			),
			pc.Trans(
				pc.Seq(parenOpen(), expressionAlias, parenClose()),
				func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
					var allTokens []tokenizer.Token
					allTokens = append(allTokens, tokens[0].Val.Original) // (
					allTokens = append(allTokens, tokens[1].Val.NewValue.RawTokens()...)
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
								rowTokens: allTokens,
							},
						},
					}, nil
				},
			),
			pc.Trans(
				literal(),
				func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
					log.Println("literal matched:", tokens[0].Val.Original)
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
								rowTokens: []tokenizer.Token{tokens[0].Val.Original},
							},
						},
					}, nil
				},
			),
		),
	)
}
