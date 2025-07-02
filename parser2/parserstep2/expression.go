package parserstep2

import (
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// ExpressionNode: Expression AST node that holds token sequences
// NodeType: EXPRESSION
// Maintains token sequences for reconstruction purposes
// String() concatenates token sequences with spaces
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

// expression parses SQL expressions with context awareness
func expression(clause SQLClause) pc.Parser[Entity] {
	// Define primary parser for this context
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
			columnReference(),
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
				pc.Lazy(func() pc.Parser[Entity] { return expression(clause) }), // Pass clause to recursive call
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

	// Return the main expression parser
	return pc.Or(
		pc.Trans(
			pc.Seq(
				primary,
				operator(),
				pc.Lazy(func() pc.Parser[Entity] { return expression(clause) }), // Pass clause to recursive call
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
