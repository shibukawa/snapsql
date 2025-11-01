package codegenerator

import (
	"sort"
	"strconv"
	"strings"
)

type removalComboSet [][]RemovalLiteral

func removalCombosImpossible() removalComboSet {
	return nil
}

func optionalComponent() removalComboSet {
	return removalComboSet{[]RemovalLiteral{}}
}

func alwaysEmitsComponent() removalComboSet {
	return removalCombosImpossible()
}

func combosFromLiteral(exprIndex int, when bool) removalComboSet {
	return removalComboSet{[]RemovalLiteral{{ExprIndex: exprIndex, When: when}}}
}

func combosAnd(a, b removalComboSet) removalComboSet {
	if a == nil || b == nil {
		return nil
	}

	result := make(removalComboSet, 0, len(a)*len(b))
	for _, comboA := range a {
		for _, comboB := range b {
			combined := append(append([]RemovalLiteral(nil), comboA...), comboB...)

			normalized, ok := normalizeCombo(combined)
			if !ok {
				continue
			}

			result = append(result, normalized)
		}
	}

	return dedupCombos(result)
}

func combosOr(a, b removalComboSet) removalComboSet {
	if a == nil {
		return b
	}

	if b == nil {
		return a
	}

	merged := make(removalComboSet, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)

	return dedupCombos(merged)
}

func normalizeCombo(combo []RemovalLiteral) ([]RemovalLiteral, bool) {
	if len(combo) == 0 {
		return []RemovalLiteral{}, true
	}

	merged := make(map[int]bool, len(combo))
	for _, lit := range combo {
		if existing, ok := merged[lit.ExprIndex]; ok {
			if existing != lit.When {
				return nil, false
			}

			continue
		}

		merged[lit.ExprIndex] = lit.When
	}

	normalized := make([]RemovalLiteral, 0, len(merged))
	for expr, when := range merged {
		normalized = append(normalized, RemovalLiteral{ExprIndex: expr, When: when})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ExprIndex == normalized[j].ExprIndex {
			if normalized[i].When == normalized[j].When {
				return false
			}

			return !normalized[i].When && normalized[j].When
		}

		return normalized[i].ExprIndex < normalized[j].ExprIndex
	})

	return normalized, true
}

func dedupCombos(set removalComboSet) removalComboSet {
	if len(set) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(set))

	result := make(removalComboSet, 0, len(set))
	for _, combo := range set {
		normalized, ok := normalizeCombo(combo)
		if !ok {
			continue
		}

		key := comboKey(normalized)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		result = append(result, normalized)
	}

	if len(result) == 0 {
		return nil
	}

	sort.Slice(result, func(i, j int) bool {
		if len(result[i]) == len(result[j]) {
			for k := range result[i] {
				if result[i][k].ExprIndex == result[j][k].ExprIndex {
					if result[i][k].When == result[j][k].When {
						continue
					}

					return !result[i][k].When && result[j][k].When
				}

				return result[i][k].ExprIndex < result[j][k].ExprIndex
			}

			return false
		}

		return len(result[i]) < len(result[j])
	})

	return result
}

func comboKey(combo []RemovalLiteral) string {
	if len(combo) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, lit := range combo {
		sb.WriteString(strconv.Itoa(lit.ExprIndex))

		if lit.When {
			sb.WriteByte('T')
		} else {
			sb.WriteByte('F')
		}

		sb.WriteByte(';')
	}

	return sb.String()
}

type whereAnalyzer struct {
	stack    []*whereFrame
	topLevel removalComboSet
}

func newWhereAnalyzer() *whereAnalyzer {
	return &whereAnalyzer{
		topLevel: optionalComponent(),
	}
}

func (a *whereAnalyzer) addComponent(component removalComboSet) {
	if len(a.stack) == 0 {
		a.topLevel = combosAnd(a.topLevel, component)
		return
	}

	frame := a.stack[len(a.stack)-1]
	if frame.current == nil {
		return
	}

	frame.current.combos = combosAnd(frame.current.combos, component)
}

func (a *whereAnalyzer) enterIf(exprIndex int) {
	branch := &whereBranch{
		literal: &RemovalLiteral{ExprIndex: exprIndex, When: true},
		combos:  optionalComponent(),
	}

	frame := &whereFrame{
		branches: []*whereBranch{branch},
		current:  branch,
	}

	a.stack = append(a.stack, frame)
}

func (a *whereAnalyzer) enterElseIf(exprIndex int) {
	if len(a.stack) == 0 {
		return
	}

	frame := a.stack[len(a.stack)-1]
	branch := &whereBranch{
		literal: &RemovalLiteral{ExprIndex: exprIndex, When: true},
		combos:  optionalComponent(),
	}
	frame.branches = append(frame.branches, branch)
	frame.current = branch
}

func (a *whereAnalyzer) enterElse() {
	if len(a.stack) == 0 {
		return
	}

	frame := a.stack[len(a.stack)-1]
	branch := &whereBranch{
		literal: nil,
		combos:  optionalComponent(),
	}
	frame.branches = append(frame.branches, branch)
	frame.current = branch
}

func (a *whereAnalyzer) exitConditional() {
	if len(a.stack) == 0 {
		return
	}

	frame := a.stack[len(a.stack)-1]
	a.stack = a.stack[:len(a.stack)-1]

	blockCombos := frame.emptyCombos()

	if len(a.stack) == 0 {
		a.topLevel = combosAnd(a.topLevel, blockCombos)
		return
	}

	parent := a.stack[len(a.stack)-1]
	if parent.current != nil {
		parent.current.combos = combosAnd(parent.current.combos, blockCombos)
	}
}

func (a *whereAnalyzer) finalize() [][]RemovalLiteral {
	if len(a.stack) > 0 {
		for len(a.stack) > 0 {
			a.exitConditional()
		}
	}

	return dedupCombos(a.topLevel)
}

type whereFrame struct {
	branches []*whereBranch
	current  *whereBranch
}

func (f *whereFrame) emptyCombos() removalComboSet {
	if len(f.branches) == 0 {
		return optionalComponent()
	}

	result := removalComboSet{}
	prev := optionalComponent()

	for _, branch := range f.branches {
		selection := prev
		if branch.literal != nil {
			selection = combosAnd(selection, combosFromLiteral(branch.literal.ExprIndex, true))
		}

		if branch.combos != nil {
			result = combosOr(result, combosAnd(selection, branch.combos))
		}

		if branch.literal != nil {
			prev = combosAnd(prev, combosFromLiteral(branch.literal.ExprIndex, false))
		} else {
			prev = removalCombosImpossible()
		}
	}

	if prev != nil {
		result = combosOr(result, prev)
	}

	return result
}

type whereBranch struct {
	literal *RemovalLiteral
	combos  removalComboSet
}
