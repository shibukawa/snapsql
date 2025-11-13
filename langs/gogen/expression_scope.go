package gogen

import "github.com/shibukawa/snapsql/intermediate"

// expressionScope keeps track of explang root identifiers that can be
// referenced at a given point within SQL builder generation. The scope starts
// with function parameters and grows/shrinks as loop variables are entered or
// exited.
type expressionScope struct {
	layers []map[string]string
}

func newExpressionScope(params []intermediate.Parameter) *expressionScope {
	base := make(map[string]string, len(params))
	for _, param := range params {
		base[param.Name] = snakeToCamelLower(param.Name)
	}

	return &expressionScope{layers: []map[string]string{base}}
}

func (s *expressionScope) lookup(name string) (string, bool) {
	for i := len(s.layers) - 1; i >= 0; i-- {
		if value, ok := s.layers[i][name]; ok {
			return value, true
		}
	}

	return "", false
}

func (s *expressionScope) push(bindings map[string]string) {
	if len(bindings) == 0 {
		return
	}

	s.layers = append(s.layers, bindings)
}

func (s *expressionScope) pushSingle(name, goName string) {
	s.push(map[string]string{name: goName})
}

func (s *expressionScope) pushLoopVar(name string) string {
	goName := snakeToCamelLower(name) + "LoopItem"
	s.pushSingle(name, goName)

	return goName
}

func (s *expressionScope) pop() {
	if len(s.layers) <= 1 {
		return
	}

	s.layers = s.layers[:len(s.layers)-1]
}
