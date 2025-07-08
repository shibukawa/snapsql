package parserstep3

import (
	"errors"
	"fmt"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	ErrSubqueryNotAllowedInClause = errors.New("subquery is not allowed in this clause")
)

var (
	subQueryStart = pc.Seq(cmn.WS(cmn.ParenOpen), cmn.Select)
)

var subqueryCanBeIn = map[cmn.NodeType]bool{
	cmn.WITH_CLAUSE:   true, // all statements
	cmn.SELECT_CLAUSE: true, // in SELECT statement
	cmn.FROM_CLAUSE:   true, // in SELECT statement
	cmn.WHERE_CLAUSE:  true, // in SELECT, UPDATE, DELETE statement
	cmn.HAVING_CLAUSE: true, // in SELECT statement
	cmn.SET_CLAUSE:    true, // in UPDATE statement
}

func CheckSubqueryUsage(clauses []cmn.ClauseNode, perr *cmn.ParseError) {
	pctx := pc.NewParseContext[tok.Token]()
	for _, clause := range clauses {
		_, _, _, _, found := pc.Find(pctx, subQueryStart, cmn.ToParserToken(clause.RawTokens()))
		if found {
			if !subqueryCanBeIn[clause.Type()] {
				perr.Add(fmt.Errorf("%w: '%s' clause at %d:%d can't have sub query", ErrSubqueryNotAllowedInClause, clause.SourceText(), clause.Position().Line, clause.Position().Column))
			}
		}
	}
}
