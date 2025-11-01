package snapsqlgo

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// MutationKind enumerates DML operation types that require WHERE guards.
type MutationKind int

const (
	MutationNone MutationKind = iota
	MutationUpdate
	MutationDelete
)

func (m MutationKind) String() string {
	switch m {
	case MutationUpdate:
		return "update"
	case MutationDelete:
		return "delete"
	default:
		return ""
	}
}

func (m MutationKind) label() string {
	switch m {
	case MutationUpdate:
		return "UPDATE"
	case MutationDelete:
		return "DELETE"
	default:
		return ""
	}
}

// NoWhereOperation configures which mutation kinds may proceed without a WHERE clause.
type NoWhereOperation uint8

const (
	AllowNoWhereUpdate NoWhereOperation = 1 << iota
	AllowNoWhereDelete
	AllowNoWhereAll = AllowNoWhereUpdate | AllowNoWhereDelete
)

const (
	WhereClauseStatusFullScan    = "fullscan"
	WhereClauseStatusExists      = "exists"
	WhereClauseStatusConditional = "conditional"
)

// RemovalLiteral represents a single boolean requirement controlling WHERE removal.
type RemovalLiteral struct {
	ExprIndex int
	When      bool
}

// WhereClauseMeta mirrors the intermediate metadata emitted for mutation statements.
type WhereClauseMeta struct {
	Status            string
	RemovalCombos     [][]RemovalLiteral
	ExpressionRefs    []int
	DynamicConditions []WhereDynamicCondition
	RawText           string
	FallbackTriggered bool
}

// WhereDynamicCondition describes a conditional construct that may remove the WHERE clause.
type WhereDynamicCondition struct {
	ExprIndex        int
	NegatedWhenEmpty bool
	HasElse          bool
	Description      string
}

// ErrEmptyWhereClause is returned when a mutation would execute without a WHERE clause.
var ErrEmptyWhereClause = errors.New("snapsqlgo: empty WHERE clause")

// WithAllowingNoWhereOperation opts specific functions into executing UPDATE/DELETE without WHERE.
func WithAllowingNoWhereOperation(ctx context.Context, funcPattern string, ops ...NoWhereOperation) context.Context {
	mask := NoWhereOperation(0)
	if len(ops) == 0 {
		mask = AllowNoWhereAll
	} else {
		for _, op := range ops {
			mask |= op
		}
	}

	return WithConfig(ctx, funcPattern, func(cfg *FuncConfig) {
		if mask&AllowNoWhereUpdate != 0 {
			cfg.AllowNoWhereUpdate = true
		}

		if mask&AllowNoWhereDelete != 0 {
			cfg.AllowNoWhereDelete = true
		}
	})
}

// EnforceNonEmptyWhereClause validates that the generated SQL still contains a WHERE clause.
func EnforceNonEmptyWhereClause(ctx context.Context, funcName string, mutation MutationKind, meta *WhereClauseMeta, query string) error {
	if mutation != MutationUpdate && mutation != MutationDelete {
		return nil
	}

	if isNoWhereAllowed(ctx, funcName, mutation) {
		return nil
	}

	if meta != nil && strings.EqualFold(meta.Status, WhereClauseStatusFullScan) {
		return buildEmptyWhereError(funcName, mutation, meta)
	}

	if meta != nil && strings.EqualFold(meta.Status, WhereClauseStatusConditional) && meta.FallbackTriggered {
		return buildEmptyWhereError(funcName, mutation, meta)
	}

	hasWhere, clause := extractTopLevelWhereClause(query)
	if hasWhere && strings.TrimSpace(clause) != "" {
		return nil
	}

	return buildEmptyWhereError(funcName, mutation, meta)
}

func isNoWhereAllowed(ctx context.Context, funcName string, mutation MutationKind) bool {
	cfg := GetFunctionConfig(ctx, funcName, mutation.String())
	if cfg == nil {
		return false
	}

	switch mutation {
	case MutationUpdate:
		return cfg.AllowNoWhereUpdate
	case MutationDelete:
		return cfg.AllowNoWhereDelete
	default:
		return false
	}
}

func buildEmptyWhereError(funcName string, mutation MutationKind, meta *WhereClauseMeta) error {
	base := fmt.Sprintf("%s %s attempted without WHERE clause", mutation.label(), funcName)

	var hints []string

	if meta == nil {
		hints = append(hints, "intermediate metadata unavailable")
	} else {
		status := strings.ToLower(meta.Status)
		switch status {
		case WhereClauseStatusFullScan:
			hints = append(hints, "template omits WHERE clause")
		case WhereClauseStatusConditional:
			if meta.FallbackTriggered {
				hints = append(hints, "conditional filters evaluated to empty (fallback guard engaged)")
			}

			if details := describeDynamicConditions(meta.DynamicConditions, true); details != "" {
				hints = append(hints, "controlled by "+details)
			}
		case WhereClauseStatusExists:
			hints = append(hints, "WHERE clause expected but missing in rendered SQL")
		default:
			hints = append(hints, "WHERE clause state unknown")
		}
	}

	if len(hints) > 0 {
		base = fmt.Sprintf("%s (%s)", base, strings.Join(hints, "; "))
	}

	suggestion := "use snapsqlgo.WithAllowingNoWhereOperation to opt-in when intentional"

	return fmt.Errorf("%w: %s. %s", ErrEmptyWhereClause, base, suggestion)
}

func describeDynamicConditions(conds []WhereDynamicCondition, filterRemovable bool) string {
	if len(conds) == 0 {
		return ""
	}

	var labels []string

	for _, cond := range conds {
		if filterRemovable && (!cond.NegatedWhenEmpty || cond.HasElse) {
			continue
		}

		label := fmt.Sprintf("expr[%d]", cond.ExprIndex)
		if cond.Description != "" {
			label += " " + cond.Description
		}

		labels = append(labels, label)
	}

	return strings.Join(labels, ", ")
}

func extractTopLevelWhereClause(query string) (bool, string) {
	const keywordWhere = "WHERE"

	upper := strings.ToUpper(query)
	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	depth := 0

	i := 0
	for i < len(upper) {
		if inLineComment {
			if upper[i] == '\n' {
				inLineComment = false
			}

			i++

			continue
		}

		if inBlockComment {
			if upper[i] == '*' && i+1 < len(upper) && upper[i+1] == '/' {
				inBlockComment = false
				i += 2

				continue
			}

			i++

			continue
		}

		ch := upper[i]

		switch {
		case !inSingle && !inDouble && !inBacktick && ch == '-' && i+1 < len(upper) && upper[i+1] == '-':
			inLineComment = true
			i += 2

			continue
		case !inSingle && !inDouble && !inBacktick && ch == '/' && i+1 < len(upper) && upper[i+1] == '*':
			inBlockComment = true
			i += 2

			continue
		case !inDouble && !inBacktick && ch == '\'':
			inSingle = !inSingle
			i++

			continue
		case !inSingle && !inBacktick && ch == '"':
			inDouble = !inDouble
			i++

			continue
		case !inSingle && !inDouble && ch == '`':
			inBacktick = !inBacktick
			i++

			continue
		}

		if inSingle || inDouble || inBacktick {
			i++
			continue
		}

		switch ch {
		case '(':
			depth++
			i++

			continue
		case ')':
			if depth > 0 {
				depth--
			}

			i++

			continue
		}

		if depth == 0 && hasKeywordAt(upper, keywordWhere, i) {
			before := byte(' ')
			if i > 0 {
				before = upper[i-1]
			}

			after := byte(' ')
			if i+len(keywordWhere) < len(upper) {
				after = upper[i+len(keywordWhere)]
			}

			if isIdentifierChar(before) || isIdentifierChar(after) {
				i++
				continue
			}

			clauseStart := i + len(keywordWhere)

			clauseEnd := findClauseEnd(upper, clauseStart)
			if clauseEnd < clauseStart {
				clauseEnd = len(query)
			}

			return true, query[clauseStart:clauseEnd]
		}

		i++
	}

	return false, ""
}

func hasKeywordAt(upper string, keyword string, idx int) bool {
	if idx+len(keyword) > len(upper) {
		return false
	}

	return upper[idx:idx+len(keyword)] == keyword
}

func isIdentifierChar(ch byte) bool {
	if ch >= 'A' && ch <= 'Z' {
		return true
	}

	if ch >= '0' && ch <= '9' {
		return true
	}

	switch ch {
	case '_', '$':
		return true
	default:
		return false
	}
}

func findClauseEnd(upper string, start int) int {
	keywords := []string{"RETURNING", "ORDER", "GROUP", "LIMIT", "FOR", "USING"}

	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	depth := 0

	i := start
	for i < len(upper) {
		if inLineComment {
			if upper[i] == '\n' {
				inLineComment = false
			}

			i++

			continue
		}

		if inBlockComment {
			if upper[i] == '*' && i+1 < len(upper) && upper[i+1] == '/' {
				inBlockComment = false
				i += 2

				continue
			}

			i++

			continue
		}

		ch := upper[i]
		switch {
		case !inSingle && !inDouble && !inBacktick && ch == '-' && i+1 < len(upper) && upper[i+1] == '-':
			inLineComment = true
			i += 2

			continue
		case !inSingle && !inDouble && !inBacktick && ch == '/' && i+1 < len(upper) && upper[i+1] == '*':
			inBlockComment = true
			i += 2

			continue
		case !inDouble && !inBacktick && ch == '\'':
			inSingle = !inSingle
			i++

			continue
		case !inSingle && !inBacktick && ch == '"':
			inDouble = !inDouble
			i++

			continue
		case !inSingle && !inDouble && ch == '`':
			inBacktick = !inBacktick
			i++

			continue
		}

		if inSingle || inDouble || inBacktick {
			i++
			continue
		}

		switch ch {
		case '(':
			depth++
			i++

			continue
		case ')':
			if depth > 0 {
				depth--
			}

			i++

			continue
		}

		if depth == 0 {
			for _, kw := range keywords {
				if hasKeywordAt(upper, kw, i) {
					before := byte(' ')
					if i > 0 {
						before = upper[i-1]
					}

					if isIdentifierChar(before) {
						continue
					}

					return i
				}
			}
		}

		i++
	}

	return len(upper)
}
