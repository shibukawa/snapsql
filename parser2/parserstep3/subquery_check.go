package parserstep3

import (
	"errors"
	"fmt"
	"slices"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	ErrSubqueryNotAllowedInClause = errors.New("subquery is not allowed in this clause")
)

var subqueryCanBeIn = map[cmn.NodeType]bool{
	cmn.WITH_CLAUSE:   true, // all statements
	cmn.SELECT_CLAUSE: true, // in SELECT statement
	cmn.FROM_CLAUSE:   true, // in SELECT statement
	cmn.WHERE_CLAUSE:  true, // in SELECT, UPDATE, DELETE statement
	cmn.HAVING_CLAUSE: true, // in SELECT statement
	cmn.SET_CLAUSE:    true, // in UPDATE statement
}

var (
	space         = primitiveType("space", tok.WHITESPACE)
	comment       = primitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	parenOpen     = primitiveType("parenOpen", tok.OPENED_PARENS)
	selectKeyword = primitiveType("select", tok.SELECT)
	subQueryStart = pc.Seq(ws(parenOpen), selectKeyword)
)

func ws(token pc.Parser[tok.Token]) pc.Parser[tok.Token] {
	return pc.Seq(
		token,
		pc.ZeroOrMore("comment or space", pc.Or(space, comment)),
	)
}

func primitiveType(typeName string, types ...tok.TokenType) pc.Parser[tok.Token] {
	return func(pctx *pc.ParseContext[tok.Token], tokens []pc.Token[tok.Token]) (int, []pc.Token[tok.Token], error) {
		if slices.Contains(types, tokens[0].Val.Type) {
			return 1, tokens, nil
		}
		return 0, nil, pc.ErrNotMatch
	}
}

func toParserToken(tokens []tok.Token) []pc.Token[tok.Token] {
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

func CheckSubqueryUsage(clauses []cmn.ClauseNode, perr *cmn.ParseError) {
	pctx := pc.NewParseContext[tok.Token]()
	for _, clause := range clauses {
		_, _, _, _, found := pc.Find(pctx, subQueryStart, toParserToken(clause.RawTokens()))
		if found {
			if !subqueryCanBeIn[clause.Type()] {
				perr.Add(fmt.Errorf("%w: '%s' clause at %d:%d can't have sub query", ErrSubqueryNotAllowedInClause, clause.SourceText(), clause.Position().Line, clause.Position().Column))
			}
		}
	}
}
