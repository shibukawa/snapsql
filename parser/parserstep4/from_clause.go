package parserstep4

import (
	"fmt"
	"slices"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	ErrTargetTableIsEmpty       = fmt.Errorf("%w: target table is empty", cmn.ErrInvalidSQL)
	ErrInvalidJoinType          = fmt.Errorf("%w: invalid join type", cmn.ErrInvalidSQL)
	ErrImplicitInnerJoin        = fmt.Errorf("%w: implicit inner join", cmn.ErrInvalidForSnapSQL)
	ErrNaturalJoinWithCondition = fmt.Errorf("%w: natural join can't have condition (on/using)", cmn.ErrInvalidSQL)
	ErrNaturalJoin              = fmt.Errorf("%w: natural join is not allowed in SnapSQL", cmn.ErrInvalidForSnapSQL)
	ErrSubQueryNeedsAlias       = fmt.Errorf("%w: sub query needs alias", cmn.ErrInvalidSQL)
)

var (
	fromClauseTableName = pc.Or(
		tag("table with schema", cmn.Identifier, cmn.Dot, cmn.Identifier),
		tag("subquery", cmn.ParenOpen, cmn.SP, subQuery),
		tag("table", cmn.Identifier),
	)

	fromClauseSplitter = pc.Or(
		cmn.WS2(cmn.Comma),
		pc.Seq(
			pc.Repeat("join qualifier", 1, 8, pc.Or(
				natural, left, right, full, inner, outer, cross, join,
			)),
		),
	)
)

// finalizeFromClause checks if a FROM clause contains a subquery without an alias.
// If found, it adds an error to perr. No return value.
func finalizeFromClause(clause *cmn.FromClause, perr *cmn.ParseError) {
	var joinHead []pc.Token[tok.Token]

	tokens := clause.ContentTokens()
	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)

	for _, part := range pc.FindIter(pctx, fromClauseSplitter, pTokens) {
		joinBody := part.Skipped
		tableRef, err := parseTableReference(pctx, joinHead, joinBody)
		if err != nil {
			perr.Add(err)
		} else {
			clause.Tables = append(clause.Tables, tableRef)
		}
		joinHead = part.Match
	}

	if len(clause.Tables) == 0 {
		perr.Add(fmt.Errorf("%w: at %s", cmn.ErrInvalidSQL, clause.RawTokens()[0].Position.String()))
	}
}

func parseJoin(pToken []pc.Token[tok.Token]) (cmn.JoinType, error) {
	pos := pToken[0].Val.Position
	if pToken[0].Val.Type == tok.COMMA {
		// comma means INNER JOIN, but explicit JOIN is required from snapsql
		return cmn.JoinInvalid, fmt.Errorf("%w: %s", ErrImplicitInnerJoin, pos.String())
	}
	joinTokens := make([]tok.TokenType, len(pToken))
	for i, m := range pToken {
		joinTokens[i] = m.Val.Type
	}
	// max 4 words (NATURAL LEFT OUTER JOIN etc)
	if len(joinTokens) > 4 || len(slices.Compact(joinTokens)) != len(joinTokens) {
		return cmn.JoinInvalid, fmt.Errorf("%w: %s", ErrInvalidJoinType, pos.String())
	}
	// JOIN should be the last
	if !slices.Contains(joinTokens, tok.JOIN) && joinTokens[len(joinTokens)-1] != tok.JOIN {
		return cmn.JoinInvalid, fmt.Errorf("%w JOIN should be last: %s", ErrInvalidJoinType, pos.String())
	}
	joinTokens = joinTokens[:len(joinTokens)-1] // remove JOIN

	switch slices.Index(joinTokens, tok.NATURAL) {
	case 0:
		if slices.Contains(joinTokens, tok.CROSS) {
			// NATURAL CROSS JOIN is not allowed
			return cmn.JoinInvalid, fmt.Errorf("%w: %s", ErrInvalidJoinType, pos.String())
		} else {
			// NATURAL JOIN is not allowed in SnapSQL
			return cmn.JoinInvalid, fmt.Errorf("%w: %s", ErrNaturalJoin, pos.String())
		}
	case -1:
	default:
		return cmn.JoinInvalid, fmt.Errorf("%w at %s: NATURAL should be first", ErrInvalidJoinType, pos.String())
	}

	switch len(joinTokens) {
	case 0: // implicit INNER JOIN
		return cmn.JoinInner, nil
	case 1:
		if joinTokens[0] == tok.OUTER {
			return cmn.JoinInvalid, fmt.Errorf("%w at %s: OUTER without LEFT/WRITE/FULL", ErrInvalidJoinType, pos.String())
		}
	case 2:
		// OUTER should not be first position. Second position should be OUTER
		if joinTokens[0] == tok.OUTER || joinTokens[1] != tok.OUTER {
			return cmn.JoinInvalid, fmt.Errorf("%w at %s: OUTER should be front of LEFT/WRITE/FULL", ErrInvalidJoinType, pos.String())
		}
		// INNER OUTER JOIN and CROSS OUTER JOIN is not allowed
		if joinTokens[0] == tok.INNER {
			return cmn.JoinInvalid, fmt.Errorf("%w at %s: can't use INNER and OUTER at the same time", ErrInvalidJoinType, pos.String())
		}
		if joinTokens[0] == tok.INNER {
			return cmn.JoinInvalid, fmt.Errorf("%w at %s: can't use CROSS and OUTER at the same time", ErrInvalidJoinType, pos.String())
		}
	}

	switch joinTokens[0] {
	case tok.LEFT:
		return cmn.JoinLeft, nil
	case tok.RIGHT:
		return cmn.JoinRight, nil
	case tok.FULL:
		return cmn.JoinFull, nil
	case tok.INNER:
		return cmn.JoinInner, nil
	case tok.CROSS:
		return cmn.JoinCross, nil
	}
	panic("should not reach here")
}

func parseTableReference(pctx *pc.ParseContext[tok.Token], head, body []pc.Token[tok.Token]) (cmn.TableReferenceForFrom, error) {
	// head: semi-colon, JOIN clause
	// body: join type and ON/USING clause
	result := cmn.TableReferenceForFrom{}

	// Join
	if head != nil {
		if len(body) == 0 {
			return cmn.TableReferenceForFrom{}, fmt.Errorf("%w: at %s", ErrTargetTableIsEmpty, head[0].Val.Position.String())
		}
		joinType, err := parseJoin(head)
		if err != nil {
			return cmn.TableReferenceForFrom{}, err
		}
		result.JoinType = joinType
		skipped, matched, _, remained, ok := pc.Find(pctx, pc.Or(on, using), body)
		if ok {
			if joinType == cmn.JoinCross {
				cond := skipped[0].Val
				return cmn.TableReferenceForFrom{}, fmt.Errorf("%w: CROSS JOIN can't have '%s' condition at %s", cmn.ErrInvalidSQL, cond.Value, cond.Position.String())
			}
			body = skipped
		} else if joinType != cmn.JoinCross {
			return cmn.TableReferenceForFrom{}, fmt.Errorf("%w: %s should have condition at %s", cmn.ErrInvalidSQL, joinType, head[0].Val.Position.String())
		}
		result.JoinCondition = cmn.ToToken(append(matched, remained...))
	}
	if len(head) == 0 && len(body) == 0 {
		return cmn.TableReferenceForFrom{}, fmt.Errorf("%w: JOIN can't be use for first table reference", ErrInvalidJoinType)
	}

	beforeAlias, alias, _, _, ok := pc.Find(pctx, alias, body)
	if ok {
		// Alias
		switch len(alias) {
		case 1: // alias without AS, but it requires before the word
			if len(beforeAlias) > 0 {
				result.Name = alias[0].Val.Value
				result.ExplicitName = true
				result.Expression = cmn.ToToken(beforeAlias)
				_, match, err := fromClauseTableName(pctx, beforeAlias)
				if err != nil {
					v := beforeAlias[0].Val
					return cmn.TableReferenceForFrom{}, fmt.Errorf("%w at %s: '%s' is invalid name for table", cmn.ErrInvalidSQL, v.Position.String(), v.Value)
				}
				switch match[0].Type {
				case "table":
					result.Name = alias[0].Val.Value
					result.TableName = match[0].Val.Value
				case "table with schema":
					result.Name = alias[0].Val.Value
					result.SchemaName = match[0].Val.Value
					result.TableName = match[1].Val.Value
				case "subquery":
					result.Name = alias[0].Val.Value
				}
			} else {
				// no alias: identifier
				result.Name = alias[0].Val.Value
				result.TableName = alias[0].Val.Value
				result.Expression = cmn.ToToken(alias[:1])
			}
		case 2:
			// as alias
			_, match, err := fromClauseTableName(pctx, beforeAlias)
			if err != nil {
				v := body[0].Val
				return cmn.TableReferenceForFrom{}, fmt.Errorf("%w at %s: '%s' is invalid name for table", cmn.ErrInvalidSQL, v.Position.String(), v.Value)
			}
			switch len(match) {
			case 1: // id as alias
				result.Name = alias[1].Val.Value
				result.TableName = match[0].Val.Value
			case 2: // subquery
				result.Name = alias[0].Val.Value
			case 3: // table with schema
				result.Name = alias[0].Val.Value
				result.SchemaName = match[0].Val.Value
				result.TableName = match[1].Val.Value
			}
			result.Name = alias[1].Val.Value
			result.ExplicitName = true
		}
	} else {
		// No Alias
		_, match, err := fromClauseTableName(pctx, body)
		if err != nil {
			if len(body) == 0 {
				h := head[0].Val
				return cmn.TableReferenceForFrom{}, fmt.Errorf("%w at %s: table name is empty", cmn.ErrInvalidSQL, h.Value)
			} else {
				v := body[0].Val
				return cmn.TableReferenceForFrom{}, fmt.Errorf("%w at %s: '%s' is invalid name for table", cmn.ErrInvalidSQL, v.Position.String(), v.Value)
			}
		} else {
			switch len(match) {
			case 1: // table only (no alias)
				result.Name = match[0].Val.Value
				result.TableName = match[0].Val.Value
			case 2: // subquery without alias
				return cmn.TableReferenceForFrom{}, fmt.Errorf("%w: at %s", ErrSubQueryNeedsAlias, body[0].Val.Position.String())
			case 3: // table with schema (no alias)
				result.Name = match[1].Val.Value
				result.SchemaName = match[0].Val.Value
				result.TableName = match[1].Val.Value
			}
		}
	}

	return result, nil
}
