package parserstep2

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
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

func (sq *SubQuery) Position() tok.Position { panic("not implemented") }
func (sq *SubQuery) RawTokens() []tok.Token { panic("not implemented") }
func (sq *SubQuery) String() string         { return "SubQuery" }

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

var (
	startSubquery = pc.Seq(ws(parenOpen), selectStatement)
	parens        = pc.Or(parenOpen, parenClose)
)

func subQuery(pctx *pc.ParseContext[Entity], t []pc.Token[Entity]) (consumed int, newTokens []pc.Token[Entity], err error) {
	current := 0

	var (
		selectOffset int
		parseStart   int
	)

	consumed, _, err = startSubquery(pctx, t)
	if err != nil {
		return 0, nil, err
	}

	selectOffset = consumed - 1
	parseStart = consumed

	// Start parsing the subquery
	stack := 1

	current = parseStart
	for _, part := range pc.FindIter(pctx, parens, t[current:]) {
		if part.Last { // not found
			break
		}

		current += len(part.Skipped) + part.Consume
		switch part.Match[0].Val.Original.Type {
		case tok.OPENED_PARENS:
			stack++
		case tok.CLOSED_PARENS:
			stack--
			if stack == 0 {
				return current, []pc.Token[Entity]{
					{
						Type: "subquery",
						Val: Entity{
							NewValue: &SubQuery{
								SelectOffset: selectOffset,
							},
							rawTokens: entityToToken(t[:current]),
						},
					},
				}, nil
			}
		}
	}

	return 0, nil, &IncompleteSubQueryError{
		Pos:           t[0].Val.Original.Position,
		MissingParent: stack,
	}
}

var (
	// firstCte returns: identity, as, subquery
	firstCte = pc.Seq(
		anyIdentifier,
		as,
		subQuery,
	)
	// subCte returns: comma, identity, as, subquery
	subCte = pc.Seq(
		comma,
		firstCte,
	)
)

func parseCTE() pc.Parser[Entity] {
	return pc.Trace("with-clause", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		consume, heading, err := withClause(pctx, tokens)
		if err != nil {
			return 0, nil, err
		}

		var result = &cmn.WithClause{
			Recursive:     heading[len(heading)-1].Val.Original.Type == tok.RECURSIVE,
			HeadingTokens: entityToToken(heading),
		}

		offset := consume

		// first CTE
		consume, match, err := firstCte(pctx, tokens[offset:])
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
			consume, match, err := subCte(pctx, tokens[offset:])
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
		consume, match, err = comma(pctx, tokens[offset:])
		if err == nil {
			offset += consume
			result.TrailingTokens = entityToToken(match)
		}

		return offset, []pc.Token[Entity]{
			{
				Type: "with-clause",
				Val: Entity{
					NewValue:  result,
					rawTokens: entityToToken(tokens[:offset]),
				},
			},
		}, nil
	})
}

var statementStart = pc.Or(
	selectStatement,
	insertIntoStatement,
	updateStatement,
	deleteFromStatement)

func DumpStatement(tokens []pc.Token[Entity]) string {
	var sb strings.Builder

	for i, token := range tokens {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("'")
		sb.WriteString(token.Val.Original.Value)
		sb.WriteString("'")
	}

	return sb.String()
}

func ParseStatement(perr *cmn.ParseError) pc.Parser[Entity] {
	return pc.Trace("statement", func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		consumeForCTE, cte, _ := ws(parseCTE())(pctx, tokens)

		var withClause *cmn.WithClause
		if len(cte) > 0 {
			if wc, ok := cte[0].Val.NewValue.(*cmn.WithClause); ok {
				withClause = wc
			}
		}

		offset := consumeForCTE

		skipped, match, _, _, found := pc.Find(pctx, statementStart, tokens[offset:])
		if !found {
			return 0, nil, pc.ErrNotMatch
		}

		offset += len(skipped)
		tokenType := match[0].Val.Original.Type
		consume, clauses := parseClauses(pctx, tokenType, withClause, tokens[offset:], perr)
		offset += consume

		switch tokenType {
		case tok.SELECT:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewSelectStatement(entityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.INSERT:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewInsertIntoStatement(entityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.UPDATE:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewUpdateStatement(entityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		case tok.DELETE:
			return offset, []pc.Token[Entity]{
				{
					Val: Entity{
						NewValue: cmn.NewDeleteFromStatement(entityToToken(skipped), withClause, clauses),
					},
				},
			}, nil
		default:
			perr.Add(fmt.Errorf("%w: unsupported statement type %s at %s",
				cmn.ErrInvalidForSnapSQL,
				tokenType,
				skipped[0].Val.Original.Position.String(),
			))
		}

		return 0, nil, pc.ErrNotMatch
	})
}

func clauseTokenSourceText(start, last int, tokens []pc.Token[Entity]) string {
	result := make([]string, 0, last-start)
	for i := start; i <= last; i++ {
		result = append(result, tokens[i].Val.Original.Value)
	}

	return strings.Join(result, " ")
}

// detectWrappedIfCondition extracts if condition from clause tokens if the clause is wrapped with if/end directives
// Also removes the if/end directive tokens from the token slices
func detectWrappedIfCondition(clauseHead []pc.Token[Entity], clauseBody []pc.Token[Entity], prevClauseBody []pc.Token[Entity]) (ifCondition string, ifIndex, endIndex int, err error) {
	// Detect end directive in the current clause body
	endIndex = -1

	for i := len(clauseBody) - 1; i >= 0; i-- {
		t := clauseBody[i]
		found := false

		switch t.Val.Original.Type {
		case tok.LINE_COMMENT, tok.WHITESPACE:
			continue
		case tok.BLOCK_COMMENT:
			if t.Val.Original.Directive != nil && t.Val.Original.Directive.Type == "end" {
				endIndex = i
				found = true
			}
		default:
			found = true
		}

		if found {
			break
		}
	}

	// Find if directive at the end of previous clause
	ifIndex = -1

	for i := len(prevClauseBody) - 1; i >= 0; i-- {
		found := false

		t := prevClauseBody[i]
		switch t.Val.Original.Type {
		case tok.LINE_COMMENT, tok.WHITESPACE:
			continue
		case tok.BLOCK_COMMENT:
			if t.Val.Original.Directive != nil && t.Val.Original.Directive.Type == "if" {
				ifCondition = t.Val.Original.Directive.Condition
				ifIndex = i
				found = true
			}
		default:
			found = true
		}

		if found {
			break
		}
	}

	if ifIndex < 0 && endIndex < 0 {
		return "", -1, -1, nil
	} else if ifIndex < 0 {
		return "", -1, -1, fmt.Errorf("%w: if condition is missing for end directive at %s",
			cmn.ErrInvalidForSnapSQL,
			clauseBody[endIndex].Val.Original.Position.String(),
		)
	} else if endIndex < 0 {
		return "", -1, -1, fmt.Errorf("%w: end directive is missing for if condition at %s",
			cmn.ErrInvalidForSnapSQL,
			prevClauseBody[ifIndex].Val.Original.Position.String(),
		)
	}

	switch clauseHead[0].Val.Original.Type {
	case tok.WHERE, tok.ORDER, tok.LIMIT, tok.OFFSET, tok.FOR:
		break
	default:
		return "", -1, -1, fmt.Errorf("%w at %s: unsupported clause type for if condition: %s",
			cmn.ErrInvalidForSnapSQL,
			prevClauseBody[ifIndex].Val.Original.Position.String(),
			clauseHead[0].Val.Original.Type)
	}

	return ifCondition, ifIndex, endIndex, nil
}

func splitter(tt tok.TokenType) pc.Parser[Entity] {
	return pc.Or(
		ws(parenOpen),
		ws(parenClose),

		// for select
		selectStatement,
		fromClause,
		whereClause,
		groupByClause,
		orderByClause,
		havingClause,
		limitClause,
		offsetClause,
		forClause,
		returningClause,

		// for insert
		insertIntoStatement,
		valuesClause,
		onConflictClause,

		// for update
		// on conflict do update, for update
		when(tt != tok.INSERT && tt != tok.SELECT, updateStatement),
		// on conflict do update set
		when(tt != tok.INSERT, setClause),

		// for delete
		deleteFromStatement,
	)
}

func clauseIter(pctx *pc.ParseContext[Entity], tt tok.TokenType, tokens []pc.Token[Entity]) iter.Seq2[int, pc.Consume[Entity]] {
	return func(yield func(index int, consume pc.Consume[Entity]) bool) {
		count := 0
		consume := 0
		nest := 0

		var skipped []pc.Token[Entity]
		for _, part := range pc.FindIter(pctx, splitter(tt), tokens) {
			if part.Last {
				yield(count, pc.Consume[Entity]{
					Consume: part.Consume,
					Skipped: append(skipped, part.Skipped...),
					Match:   nil,
					Last:    true,
				})
			} else if nest > 0 {
				switch part.Match[0].Val.Original.Type {
				case tok.OPENED_PARENS:
					nest++
				case tok.CLOSED_PARENS:
					nest--
				}

				skipped = append(skipped, part.Skipped...)
				for _, m := range part.Match {
					skipped = append(skipped, tokenToEntity(m.Val.RawTokens())...)
				}

				consume += part.Consume
			} else {
				if part.Match[0].Val.Original.Type == tok.OPENED_PARENS {
					skipped = append(skipped, part.Skipped...)
					for _, m := range part.Match {
						skipped = append(skipped, tokenToEntity(m.Val.RawTokens())...)
					}

					consume += part.Consume + len(part.Skipped)
					nest = 1
				} else {
					yield(count, pc.Consume[Entity]{
						Consume: part.Consume,
						Skipped: append(skipped, part.Skipped...),
						Match:   part.Match,
						Last:    part.Last,
					})
					skipped = nil
					consume = 0
					count++
				}
			}
		}
	}
}

func parseClauses(pctx *pc.ParseContext[Entity], tt tok.TokenType, withClause *cmn.WithClause, tokens []pc.Token[Entity], perr *cmn.ParseError) (int, []cmn.ClauseNode) {
	var (
		clauseHead     []pc.Token[Entity]
		prevClauseBody []pc.Token[Entity]
		clauses        []cmn.ClauseNode
		prevClause     cmn.ClauseNode
	)

	if withClause != nil {
		clauses = append(clauses, withClause)
		prevClause = withClause
	}

	var consumes int
	for i, clause := range clauseIter(pctx, tt, tokens) {
		consumes += clause.Consume + len(clause.Skipped)

		clauseBody := clause.Skipped
		if i == 0 {
			clauseHead = clause.Match
			prevClauseBody = clauseBody
		} else {
			if len(clauseHead) > 0 {
				// Extract if condition and update clause bodies
				/*ifCondition, newClauseBody, newPrevClauseBody := extractIfCondition(clauseHead, clauseBody, prevClauseBody)

				// Create new clause with updated bodies*/
				newClause := newClauseNode(clauseHead, clauseBody, prevClauseBody, prevClause)

				ifCondition, ifIndex, endIndex, err := detectWrappedIfCondition(clauseHead, clauseBody, prevClauseBody)
				if err != nil {
					perr.Add(err)
				}

				if ifIndex != -1 && endIndex != -1 {
					newClause.SetIfCondition(ifCondition, ifIndex, endIndex, prevClause)
				}

				clauses = append(clauses, newClause)
				prevClause = newClause
				prevClauseBody = clauseBody
			} else {
				prevClauseBody = clauseBody
			}

			clauseHead = clause.Match
		}
	}

	return consumes, clauses
}

func newClauseNode(clauseHead []pc.Token[Entity], clauseBody []pc.Token[Entity], prevClauseBody []pc.Token[Entity], prevClause cmn.ClauseNode) cmn.ClauseNode {
	var clauseNode cmn.ClauseNode

	switch clauseHead[0].Val.Original.Type {
	// Select
	case tok.SELECT:
		clauseNode = cmn.NewSelectClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.FROM:
		clauseNode = cmn.NewFromClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.WHERE:
		clauseNode = cmn.NewWhereClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.GROUP:
		clauseNode = cmn.NewGroupByClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.HAVING:
		clauseNode = cmn.NewHavingClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.ORDER:
		clauseNode = cmn.NewOrderByClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.LIMIT:
		clauseNode = cmn.NewLimitClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.OFFSET:
		clauseNode = cmn.NewOffsetClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.FOR:
		clauseNode = cmn.NewForClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.RETURNING:
		clauseNode = cmn.NewReturningClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))

	// Insert
	case tok.INSERT:
		clauseNode = cmn.NewInsertIntoClause(
			clauseTokenSourceText(1, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.VALUES:
		clauseNode = cmn.NewValuesClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.ON:
		clauseNode = cmn.NewOnConflictClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))

	// Update
	case tok.UPDATE:
		clauseNode = cmn.NewUpdateClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))
	case tok.SET:
		clauseNode = cmn.NewSetClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))

	// Delete
	case tok.DELETE:
		clauseNode = cmn.NewDeleteFromClause(
			clauseTokenSourceText(1, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(clauseBody))

	default:
		panic("unknown clause type")
	}

	return clauseNode
}
