package parserstep2

import (
	"fmt"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

var SUB_QUERY cmn.NodeType = cmn.LAST_NODE_TYPE + 1

type SubQuery struct {
	SelectOffset int
}

// Type implements parsercommon.AstNode.
func (sq *SubQuery) Type() cmn.NodeType {
	return SUB_QUERY
}

func (sq SubQuery) Position() tokenizer.Position { panic("not implemented") }
func (sq SubQuery) RawTokens() []tokenizer.Token { panic("not implemented") }
func (sq SubQuery) String() string               { panic("not implemented") }

var _ cmn.AstNode = (*SubQuery)(nil)

type IncompleteSubQueryError struct {
	Pos           tokenizer.Position
	MissingParent int
	Reason        string
}

func (e IncompleteSubQueryError) Error() string {
	if e.MissingParent == 0 {
		return fmt.Sprintf("Incomplete subquery at %d:%d: %s", e.Pos.Line, e.Pos.Column, e.Reason)
	}
	return fmt.Sprintf("Incomplete subquery at %d:%d (%d close parens are missing): %s", e.Pos.Line, e.Pos.Column, e.MissingParent, e.Reason)
}

func (e IncompleteSubQueryError) Unwrap() error {
	return pc.ErrCritical
}

func startSubquery() pc.Parser[Entity] {
	return pc.Seq(ws(parenOpen()), selectKeyword())
}

func parentAndSemicolon() pc.Parser[Entity] {
	return pc.Or(parenOpen(), parenClose(), semicolon())
}

func subQuery() pc.Parser[Entity] {
	return func(pctx *pc.ParseContext[Entity], t []pc.Token[Entity]) (consumed int, newTokens []pc.Token[Entity], err error) {
		current := 0
		var selectOffset int
		var parseStart int

		consumed, _, err = startSubquery()(pctx, t)
		if err != nil {
			return 0, nil, err
		}
		selectOffset = consumed - 1
		parseStart = consumed

		// Start parsing the subquery
		stack := 1
		current = parseStart
		for _, part := range pc.FindIter(pctx, parentAndSemicolon(), t[current:]) {
			if part.Last { // not found
				break
			}
			switch part.Match[0].Val.Original.Type {
			case tokenizer.OPENED_PARENS:
				stack++
			case tokenizer.CLOSED_PARENS:
				stack--
				if stack == 0 {
					current += len(part.Skipped) + part.Consume
					return current, []pc.Token[Entity]{
						{
							Type: "subquery",
							Val: Entity{
								NewValue: &SubQuery{
									SelectOffset: selectOffset,
								},
							},
						},
					}, nil
				}
			case tokenizer.SEMICOLON:
				return 0, nil, &IncompleteSubQueryError{
					Pos:           t[0].Val.Original.Position,
					MissingParent: stack,
					Reason:        "Subquery ended with a semicolon, but closed paren is missing before EOF",
				}
			}
			current += len(part.Skipped) + part.Consume
		}
		return 0, nil, &IncompleteSubQueryError{
			Pos:           t[0].Val.Original.Position,
			MissingParent: stack,
		}
	}
}

func statementStart() pc.Parser[Entity] {
	return pc.Or(selectStatement(), insertStatement(), updateStatement(), deleteStatement())
}

// clauseStart is a parser that matches any SQL clause.
// It matches all clauses that can appear in a SQL statement.
// Availability of clauses is checked in next step.
func clauseStart() pc.Parser[Entity] {
	return pc.Or(
		selectClause(),
		fromClause(),
		whereClause(),
		groupByClause(),
		orderByClause(),
		havingClause(),
		limitClause(),
		offsetClause(),
		returningClause(),
		semicolon(),
	)
}

func ParseStatement() pc.Parser[Entity] {
	return pc.Trace("statement", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		skipped, match, _, _, found := pc.Find(pctx, statementStart(), tokens)
		if !found {
			return 0, nil, pc.ErrNotMatch
		}
		switch match[0].Val.Original.Type {
		case tokenizer.SELECT:
			consume, selectStatement, err := parseSelectStatement(pctx, tokens[len(skipped):])
			if err != nil {
				return 0, nil, err
			}
			return len(skipped) + consume, []pc.Token[Entity]{
				pc.Token[Entity]{
					Val: Entity{
						NewValue: selectStatement,
					},
				},
			}, nil
		case tokenizer.INSERT:
		case tokenizer.UPDATE:
		case tokenizer.DELETE:
		default:
		}
		return 0, nil, pc.ErrNotMatch
	})
}

func parseSelectStatement(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, *cmn.SelectStatement, error) {
	var clauseHead []pc.Token[Entity]
	var clauses []cmn.ClauseNode
	var consumes int
	for i, clause := range pc.FindIter(pctx, clauseStart(), tokens) {
		if i != 0 {
			switch clauseHead[0].Val.Original.Type {
			case tokenizer.SELECT:
				clauses = append(clauses, cmn.NewSelectClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.FROM:
				clauses = append(clauses, cmn.NewFromClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.WHERE:
				clauses = append(clauses, cmn.NewWhereClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.GROUP:
				clauses = append(clauses, cmn.NewGroupByClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.HAVING:
				clauses = append(clauses, cmn.NewHavingClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.ORDER:
				clauses = append(clauses, cmn.NewOrderByClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.LIMIT:
				clauses = append(clauses, cmn.NewLimitClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.OFFSET:
				clauses = append(clauses, cmn.NewOffsetClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			case tokenizer.RETURNING:
				clauses = append(clauses, cmn.NewReturningClause(EntityToToken(clauseHead), EntityToToken(clause.Skipped)))
			}
		}
		clauseHead = clause.Match
		consumes += clause.Consume + len(clause.Skipped)
	}
	return consumes, cmn.NewSelectStatement(
		EntityToToken(clauseHead), nil, clauses), nil
}
