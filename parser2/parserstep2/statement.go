package parserstep2

import (
	"errors"
	"fmt"
	"strings"

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
	var selectOffset int
	var parseStart int

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
		current += len(part.Skipped) + part.Consume
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

// clauseStart is a parser that matches any SQL clause.
// It matches all clauses that can appear in a SQL statement.
// Availability of clauses is checked in next step.
func clauseStart(tt tok.TokenType) pc.Parser[Entity] {
	return pc.Or(
		// for subquery
		subQuery,

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
		skipped, match, _, _, found := pc.Find(pctx, statementStart, tokens[offset:])
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

// extractIfCondition extracts if condition from clause tokens if the clause is wrapped with if/end directives
// Also removes the if/end directive tokens from the token slices
func extractIfCondition(clauseHead []pc.Token[Entity], clauseBody []pc.Token[Entity], prevClauseBody []pc.Token[Entity]) (string, []pc.Token[Entity], []pc.Token[Entity]) {
	// Check if this clause type supports conditional behavior (WHERE, ORDER BY, LIMIT, OFFSET)
	if len(clauseHead) == 0 {
		return "", clauseBody, prevClauseBody
	}

	clauseType := clauseHead[0].Val.Original.Type
	supportedTypes := []tok.TokenType{tok.WHERE, tok.ORDER, tok.LIMIT, tok.OFFSET}
	isSupported := false
	for _, t := range supportedTypes {
		if clauseType == t {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return "", clauseBody, prevClauseBody
	}

	// Find if directive at the end of previous clause (excluding whitespace and non-directive comments)
	ifCondition := ""
	ifIndex := -1
	if len(prevClauseBody) > 0 {
		for i := len(prevClauseBody) - 1; i >= 0; i-- {
			token := prevClauseBody[i]
			// Skip whitespace and non-directive comments
			if token.Val.Original.Type == tok.WHITESPACE ||
				(token.Val.Original.Type == tok.BLOCK_COMMENT && token.Val.Original.Directive == nil) ||
				(token.Val.Original.Type == tok.LINE_COMMENT && token.Val.Original.Directive == nil) {
				continue
			}
			// Check if this is an if directive
			if token.Val.Original.Directive != nil && token.Val.Original.Directive.Type == "if" {
				ifCondition = token.Val.Original.Directive.Condition
				ifIndex = i
				break
			}
			// If we found a non-whitespace, non-comment token, stop looking
			break
		}
	}

	// Find end directive at the end of current clause body (excluding whitespace and non-directive comments)
	// If the last token is an if directive, look before it
	foundEnd := false
	endIndex := -1
	if len(clauseBody) > 0 {
		startIdx := len(clauseBody) - 1
		// If the last token is an if directive, start looking before it
		if clauseBody[startIdx].Val.Original.Directive != nil && clauseBody[startIdx].Val.Original.Directive.Type == "if" {
			startIdx--
		}

		for i := startIdx; i >= 0; i-- {
			token := clauseBody[i]
			// Skip whitespace and non-directive comments
			if token.Val.Original.Type == tok.WHITESPACE ||
				(token.Val.Original.Type == tok.BLOCK_COMMENT && token.Val.Original.Directive == nil) ||
				(token.Val.Original.Type == tok.LINE_COMMENT && token.Val.Original.Directive == nil) {
				continue
			}
			// Check if this is an end directive
			if token.Val.Original.Directive != nil && token.Val.Original.Directive.Type == "end" {
				foundEnd = true
				endIndex = i
				break
			}
			// If we found a non-whitespace, non-comment token, stop looking
			break
		}
	}

	// Only return the condition and remove tokens if we found both if and end directives
	if ifCondition != "" && foundEnd {
		// Remove if directive from previous clause body
		newPrevClauseBody := make([]pc.Token[Entity], 0, len(prevClauseBody))
		for i, token := range prevClauseBody {
			if i != ifIndex {
				newPrevClauseBody = append(newPrevClauseBody, token)
			}
		}

		// Remove end directive from current clause body
		newClauseBody := make([]pc.Token[Entity], 0, len(clauseBody))
		for i, token := range clauseBody {
			if i != endIndex {
				newClauseBody = append(newClauseBody, token)
			}
		}

		return ifCondition, newClauseBody, newPrevClauseBody
	}

	return "", clauseBody, prevClauseBody
}

func parseClauses(pctx *pc.ParseContext[Entity], tt tok.TokenType, withClause *cmn.WithClause, tokens []pc.Token[Entity]) (int, []cmn.ClauseNode, error) {
	var clauseHead []pc.Token[Entity]
	var clauseBody []pc.Token[Entity]
	var prevClauseBody []pc.Token[Entity]
	var clauses []cmn.ClauseNode
	if withClause != nil {
		clauses = append(clauses, withClause)
	}
	var consumes int
	for i, clause := range pc.FindIter(pctx, clauseStart(tt), tokens) {
		consumes += clause.Consume + len(clause.Skipped)
		if i != 0 {
			if clause.Match != nil && clause.Match[0].Val.NewValue != nil && clause.Match[0].Val.NewValue.Type() == SUB_QUERY {
				clauseBody = append(clauseBody, clause.Skipped...)
				clauseBody = append(clauseBody, clause.Match...)
				continue
			}
			clauseBody = append(clauseBody, clause.Skipped...)
			clauses = append(clauses, newClauseNode(clauseHead, clauseBody, prevClauseBody))
			prevClauseBody = clauseBody
			clauseBody = nil
		}
		clauseHead = clause.Match
	}
	return consumes, clauses, nil
}

func newClauseNode(clauseHead []pc.Token[Entity], clauseBody []pc.Token[Entity], prevClauseBody []pc.Token[Entity]) cmn.ClauseNode {
	var clauseNode cmn.ClauseNode

	// Extract if condition for conditional clauses and remove if/end directives
	ifCondition, newClauseBody, _ := extractIfCondition(clauseHead, clauseBody, prevClauseBody)

	switch clauseHead[0].Val.Original.Type {
	// Select
	case tok.SELECT:
		clauseNode = cmn.NewSelectClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.FROM:
		clauseNode = cmn.NewFromClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.WHERE:
		clauseNode = cmn.NewWhereClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody),
			ifCondition)
	case tok.GROUP:
		clauseNode = cmn.NewGroupByClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.HAVING:
		clauseNode = cmn.NewHavingClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.ORDER:
		clauseNode = cmn.NewOrderByClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody),
			ifCondition)
	case tok.LIMIT:
		clauseNode = cmn.NewLimitClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody),
			ifCondition)
	case tok.OFFSET:
		clauseNode = cmn.NewOffsetClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody),
			ifCondition)
	case tok.FOR:
		clauseNode = cmn.NewForClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.RETURNING:
		clauseNode = cmn.NewReturningClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))

	// Insert
	case tok.INSERT:
		clauseNode = cmn.NewInsertIntoClause(
			clauseTokenSourceText(1, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.VALUES:
		clauseNode = cmn.NewValuesClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.ON:
		clauseNode = cmn.NewOnConflictClause(
			clauseTokenSourceText(0, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))

	// Update
	case tok.UPDATE:
		clauseNode = cmn.NewUpdateClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))
	case tok.SET:
		clauseNode = cmn.NewSetClause(
			clauseTokenSourceText(0, 0, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))

	// Delete
	case tok.DELETE:
		clauseNode = cmn.NewDeleteFromClause(
			clauseTokenSourceText(1, 1, clauseHead),
			entityToToken(clauseHead),
			entityToToken(newClauseBody))

	default:
		panic("unknown clause type")
	}
	return clauseNode
}
