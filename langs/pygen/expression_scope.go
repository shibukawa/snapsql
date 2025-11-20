package pygen

import "maps"

import "github.com/shibukawa/snapsql/intermediate"

// expressionScope tracks identifiers that can be referenced in generated Python code.
type expressionScope struct {
	layers []map[string]string
}

func newExpressionScope(format *intermediate.IntermediateFormat) *expressionScope {
	base := map[string]string{}

	if format != nil {
		for _, param := range format.Parameters {
			base[param.Name] = pythonIdentifier(param.Name)
		}

		for _, param := range format.ImplicitParameters {
			base[param.Name] = pythonIdentifier(param.Name)
		}
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
	layer := map[string]string{}
	maps.Copy(layer, bindings)

	s.layers = append(s.layers, layer)
}

func (s *expressionScope) pushSingle(name, value string) {
	if name == "" {
		return
	}

	s.push(map[string]string{name: value})
}

func (s *expressionScope) pop() {
	if len(s.layers) == 0 {
		return
	}

	s.layers = s.layers[:len(s.layers)-1]
}
