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
							spaces:    tokens[0].Val.spaces,
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
				allTokens, spaces := margeTokens(tokens)
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
							spaces:    spaces,
						},
					},
				}, nil
			},
		),
	)

	// Return the main expression parser
	return pc.Or(
		// a between b like c
		pc.Trans(
			pc.Seq(
				primary,
				between(),
				primary,
				andOp(),
				pc.Lazy(func() pc.Parser[Entity] { return expression(clause) }),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				allTokens, spaces := margeTokens(tokens)
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
							spaces:    spaces,
						},
					},
				}, nil
			},
		),
		// a not like b
		pc.Trans(
			pc.Seq(
				primary,
				not(),
				like(),
				pc.Lazy(func() pc.Parser[Entity] { return expression(clause) }),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				allTokens, spaces := margeTokens(tokens)
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
							spaces:    spaces,
						},
					},
				}, nil
			},
		),
		// a operator b
		pc.Trans(
			pc.Seq(
				primary,
				operator(),
				pc.Lazy(func() pc.Parser[Entity] { return expression(clause) }),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				allTokens, spaces := margeTokens(tokens)
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
							spaces:    spaces,
						},
					},
				}, nil
			},
		),
		primary,
	)
}

func margeTokens(tokens []pc.Token[Entity]) ([]tokenizer.Token, [][]tokenizer.Token) {
	var allTokens []tokenizer.Token
	var spaces [][]tokenizer.Token
	for _, t := range tokens {
		allTokens = append(allTokens, t.Val.rawTokens...)
		spaces = append(spaces, t.Val.spaces...)
	}
	return allTokens, spaces
}
