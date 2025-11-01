package codegenerator

import "strings"

const (
	// StatusFullScan indicates that the WHERE clause is absent and the statement would perform a full scan.
	StatusFullScan = "fullscan"
	// StatusExists indicates that the WHERE clause always produces at least one predicate.
	StatusExists = "exists"
	// StatusConditional indicates that the WHERE clause may be removed depending on runtime conditions.
	StatusConditional = "conditional"
)

// RemovalLiteral represents a single boolean requirement (expression index + desired truth value)
// that must hold for the WHERE clause to be removed.
type RemovalLiteral struct {
	ExprIndex int  `json:"expr_index"`
	When      bool `json:"when"`
}

// WhereClauseMeta captures metadata about a statement's WHERE clause, including whether it
// exists, whether it can be removed by templating, and which CEL expressions control it.
type WhereClauseMeta struct {
	Present           bool
	Dynamic           bool
	ExpressionRefs    []int
	DynamicConditions []*WhereDynamicCondition
	RawText           string

	Status        string
	RemovalCombos [][]RemovalLiteral

	analyzer *whereAnalyzer
}

// WhereDynamicCondition describes a single conditional or loop construct that can cause the
// WHERE clause to be omitted when evaluated to a certain value.
type WhereDynamicCondition struct {
	ExprIndex        int
	NegatedWhenEmpty bool
	HasElse          bool
	Description      string
}

func newWhereClauseMeta(raw string) *WhereClauseMeta {
	return &WhereClauseMeta{
		Present:  true,
		RawText:  raw,
		analyzer: newWhereAnalyzer(),
	}
}

func (m *WhereClauseMeta) ensureAnalyzer() {
	if m != nil && m.analyzer == nil {
		m.analyzer = newWhereAnalyzer()
	}
}

// RecordStatic marks a static SQL fragment emitted within the current WHERE context.
func (m *WhereClauseMeta) RecordStatic(value string) {
	if m == nil {
		return
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}

	if strings.EqualFold(trimmed, "WHERE") {
		return
	}

	m.ensureAnalyzer()
	m.analyzer.addComponent(alwaysEmitsComponent())
}

// RecordEval marks an evaluated parameter emitted within the current WHERE context.
func (m *WhereClauseMeta) RecordEval() {
	if m == nil {
		return
	}

	m.ensureAnalyzer()
	m.analyzer.addComponent(alwaysEmitsComponent())
}

// EnterIf notifies the analyzer that an IF directive was encountered.
func (m *WhereClauseMeta) EnterIf(exprIndex int) {
	if m == nil {
		return
	}

	m.ensureAnalyzer()
	m.analyzer.enterIf(exprIndex)
}

// EnterElseIf notifies the analyzer that an ELSEIF directive was encountered.
func (m *WhereClauseMeta) EnterElseIf(exprIndex int) {
	if m == nil {
		return
	}

	if m.analyzer == nil {
		return
	}

	m.analyzer.enterElseIf(exprIndex)
}

// EnterElse notifies the analyzer that an ELSE directive was encountered.
func (m *WhereClauseMeta) EnterElse() {
	if m == nil {
		return
	}

	if m.analyzer == nil {
		return
	}

	m.analyzer.enterElse()
}

// ExitConditional finalizes the most recent conditional block and integrates its metadata.
func (m *WhereClauseMeta) ExitConditional() {
	if m == nil {
		return
	}

	if m.analyzer == nil {
		return
	}

	m.analyzer.exitConditional()
}

// Finalize computes the status and removal combinations after the WHERE clause has been processed.
func (m *WhereClauseMeta) Finalize() {
	if m == nil {
		return
	}

	if !m.Present {
		m.Status = StatusFullScan
		m.RemovalCombos = nil
		m.analyzer = nil

		return
	}

	if m.analyzer == nil {
		// No analyzer activity implies the WHERE clause contained no meaningful tokens.
		m.Status = StatusFullScan
		m.RemovalCombos = [][]RemovalLiteral{{}}

		return
	}

	combos := m.analyzer.finalize()
	if combos == nil {
		m.Status = StatusExists
		m.RemovalCombos = nil
		m.analyzer = nil

		return
	}

	if containsEmptyCombo(combos) {
		m.Status = StatusFullScan
		m.RemovalCombos = [][]RemovalLiteral{{}}
	} else {
		m.Status = StatusConditional
		m.RemovalCombos = combos
	}

	m.analyzer = nil
}

// clone creates a deep copy of WhereClauseMeta.
func (m *WhereClauseMeta) clone() *WhereClauseMeta {
	if m == nil {
		return nil
	}

	clone := &WhereClauseMeta{
		Present:        m.Present,
		Dynamic:        m.Dynamic,
		RawText:        m.RawText,
		ExpressionRefs: append([]int(nil), m.ExpressionRefs...),
		Status:         m.Status,
	}

	if len(m.DynamicConditions) > 0 {
		clone.DynamicConditions = make([]*WhereDynamicCondition, len(m.DynamicConditions))
		for i, cond := range m.DynamicConditions {
			if cond == nil {
				continue
			}

			copyCond := *cond
			clone.DynamicConditions[i] = &copyCond
		}
	}

	if len(m.RemovalCombos) > 0 {
		clone.RemovalCombos = make([][]RemovalLiteral, len(m.RemovalCombos))
		for i, combo := range m.RemovalCombos {
			clone.RemovalCombos[i] = append([]RemovalLiteral(nil), combo...)
		}
	}

	return clone
}

// containsEmptyCombo returns true if combos contain an unconditional removal entry (empty conjunction).
func containsEmptyCombo(combos [][]RemovalLiteral) bool {
	for _, combo := range combos {
		if len(combo) == 0 {
			return true
		}
	}

	return false
}
