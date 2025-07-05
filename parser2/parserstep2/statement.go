package parserstep2

import (
	"errors"
	"fmt"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var SUB_QUERY cmn.NodeType = cmn.LAST_NODE_TYPE + 1

type SubQuery struct {
	SelectOffset int
}

// Type implements parsercommon.AstNode.
func (sq *SubQuery) Type() cmn.NodeType {
	return SUB_QUERY
}

func (sq SubQuery) Position() tok.Position { panic("not implemented") }
func (sq SubQuery) RawTokens() []tok.Token { panic("not implemented") }
func (sq SubQuery) String() string         { panic("not implemented") }

var _ cmn.AstNode = (*SubQuery)(nil)

type IncompleteSubQueryError struct {
	Pos           tok.Position
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
			case tok.OPENED_PARENS:
				stack++
			case tok.CLOSED_PARENS:
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
			case tok.SEMICOLON:
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

// firstCte returns: identity, as, subquery
func firstCte() pc.Parser[Entity] {
	return pc.Seq(
		anyIdentifier(),
		as(),
		subQuery(),
	)
}

// subCte returns: comma, identity, as, subquery
func subCte() pc.Parser[Entity] {
	return pc.Seq(
		comma(),
		firstCte(),
	)
}

func parseCTE() pc.Parser[Entity] {
	return pc.Trace("with-clause", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		consume, heading, err := withClause()(pctx, tokens)
		if err != nil {
			return 0, nil, err
		}
		var result = &cmn.WithClause{
			Recursive:     heading[len(heading)-1].Val.Original.Type == tok.RECURSIVE,
			HeadingTokens: EntityToToken(heading),
		}
		offset := consume

		// first CTE
		consume, match, err := firstCte()(pctx, tokens[offset:])
		if err != nil {
			if len(tokens) < offset {
				return 0, nil, fmt.Errorf("%w: sub query is missing at last", pc.ErrCritical)
			}
			p := tokens[offset].Val.Original.Position
			return 0, nil, fmt.Errorf("%w: can't parse subquery at %d:%d", pc.ErrCritical, p.Line, p.Column)
		}
		offset += consume
		result.CTEs = append(result.CTEs, cmn.CTEDefinition{
			Name:   match[0].Val.Original.Value,
			Select: match[2].Val.NewValue,
		})

		// second and subsequent CTEs
		for {
			consume, match, err := subCte()(pctx, tokens[offset:])
			if errors.Is(err, pc.ErrNotMatch) {
				break
			} else if err != nil {
				if len(tokens) < offset {
					return 0, nil, fmt.Errorf("%w: sub query is missing at last", pc.ErrCritical)
				}
				p := tokens[offset].Val.Original.Position
				return 0, nil, fmt.Errorf("%w: can't parse subquery at %d:%d", pc.ErrCritical, p.Line, p.Column)
			}
			offset += consume
			result.CTEs = append(result.CTEs, cmn.CTEDefinition{
				Name:   match[1].Val.Original.Value,
				Select: match[3].Val.NewValue,
			})
		}

		// Extra comma
		consume, match, err = comma()(pctx, tokens[offset:])
		if err == nil {
			offset += consume
			result.TrailingTokens = EntityToToken(match)
		}

		return offset, []pc.Token[Entity]{
			{
				Type: "with-clause",
				Val: Entity{
					NewValue:  result,
					rawTokens: EntityToToken(tokens[:offset]),
				},
			},
		}, nil
	})
}

func statementStart() pc.Parser[Entity] {
	return pc.Or(
		selectStatement(),
		insertIntoStatement(),
		updateStatement(),
		deleteFromStatement())
}

// clauseStart is a parser that matches any SQL clause.
// It matches all clauses that can appear in a SQL statement.
// Availability of clauses is checked in next step.
func clauseStart(tt tok.TokenType) pc.Parser[Entity] {
	return pc.Or(
		semicolon(),
		// for subquery
		subQuery(),

		// for select
		selectStatement(),
		fromClause(),
		whereClause(),
		groupByClause(),
		orderByClause(),
		havingClause(),
		limitClause(),
		offsetClause(),
		forClause(),
		returningClause(),

		// for insert
		insertIntoStatement(),
		valuesClause(),
		onConflictClause(),

		// for update
		// on conflict do update, for update
		when(tt != tok.INSERT && tt != tok.SELECT, updateStatement()),
		// on conflict do update set
		when(tt != tok.INSERT, setClause()),

		// for delete
		deleteFromStatement(),
	)
}

func ParseStatement() pc.Parser[Entity] {
	return pc.Trace("statement", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		consumeForCTE, cte, err := ws(parseCTE())(pctx, tokens)
		var withClause *cmn.WithClause
		if len(cte) > 0 {
			if wc, ok := cte[0].Val.NewValue.(*cmn.WithClause); ok {
				withClause = wc
			}
		}
		offset := consumeForCTE
		skipped, match, _, _, found := pc.Find(pctx, statementStart(), tokens[offset:])
		if !found {
			return 0, nil, pc.ErrNotMatch
		}
		offset += len(skipped)
		tokenType := match[0].Val.Original.Type
		consume, clauses, err := parseClauses(pctx, tokenType, withClause, tokens[offset:])
		if err != nil {
			return 0, nil, err
		}
		offset += consume
		switch tokenType {
		case tok.SELECT:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewSelectStatement(EntityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.INSERT:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewInsertIntoStatement(EntityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.UPDATE:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewUpdateStatement(EntityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.DELETE:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewDeleteFromStatement(EntityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		default:
		}
		return 0, nil, pc.ErrNotMatch
	})
}

func parseClauses(pctx *pc.ParseContext[Entity], tt tok.TokenType, withClause *cmn.WithClause, tokens []pc.Token[Entity]) (int, []cmn.ClauseNode, error) {
	var clauseHead []pc.Token[Entity]
	var clauseBody []pc.Token[Entity]
	var clauses []cmn.ClauseNode
	if withClause != nil {
		clauses = append(clauses, withClause)
	}
	var consumes int
	for i, clause := range pc.FindIter(pctx, clauseStart(tt), tokens) {
		if i != 0 {
			ct := clauseHead[0].Val.Original.Type
			if ct == tok.OPENED_PARENS {
				// Subquery
				clauseBody = append(clauseBody, clause.Skipped...)
				clauseBody = append(clauseBody, clause.Match...)
				continue
			}
			switch ct {
			// Select
			case tok.SELECT:
				clauses = append(clauses,
					cmn.NewSelectClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.FROM:
				clauses = append(clauses,
					cmn.NewFromClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.WHERE:
				clauses = append(clauses,
					cmn.NewWhereClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.GROUP:
				clauses = append(clauses,
					cmn.NewGroupByClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.HAVING:
				clauses = append(clauses,
					cmn.NewHavingClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.ORDER:
				clauses = append(clauses,
					cmn.NewOrderByClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.LIMIT:
				clauses = append(clauses,
					cmn.NewLimitClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.OFFSET:
				clauses = append(clauses,
					cmn.NewOffsetClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.FOR:
				clauses = append(clauses,
					cmn.NewForClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.RETURNING:
				clauses = append(clauses,
					cmn.NewReturningClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))

			// Insert
			case tok.INSERT:
				clauses = append(clauses,
					cmn.NewInsertIntoClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.VALUES:
				clauses = append(clauses,
					cmn.NewValuesClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.ON:
				clauses = append(clauses,
					cmn.NewOnConflictClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))

			// Update
			case tok.UPDATE:
				clauses = append(clauses,
					cmn.NewUpdateClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			case tok.SET:
				clauses = append(clauses,
					cmn.NewSetClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))

			// Delete
			case tok.DELETE:
				clauses = append(clauses,
					cmn.NewDeleteFromClause(EntityToToken(clauseHead),
						EntityToToken(append(clauseBody, clause.Skipped...))))
			}
			clauseBody = nil
		}
		clauseHead = clause.Match
		consumes += clause.Consume + len(clause.Skipped)
	}
	return consumes, clauses, nil
}
