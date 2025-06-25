package parser

import (
	"gopkg.in/yaml.v3"
)

// OrderedParameter represents a parameter with simplified structure
type OrderedParameter struct {
	Name string // Parameter name
	Type any    // Parameter type (can be string, []any, map[string]any for recursive structure)
}

// OrderedParameters holds parameters with their definition order
type OrderedParameters struct {
	Parameters []OrderedParameter // Parameters in definition order
	nameMap    map[string]int     // Name to index mapping for quick lookup
}

// NewOrderedParameters creates a new ordered parameters container
func NewOrderedParameters() *OrderedParameters {
	return &OrderedParameters{
		Parameters: make([]OrderedParameter, 0),
		nameMap:    make(map[string]int),
	}
}

// Add adds a parameter (simplified - no path or order)
func (op *OrderedParameters) Add(name string, paramType any) {
	param := OrderedParameter{
		Name: name,
		Type: paramType,
	}

	op.Parameters = append(op.Parameters, param)
	op.nameMap[name] = len(op.Parameters) - 1
}

// GetByName returns parameter by name
func (op *OrderedParameters) GetByName(name string) (*OrderedParameter, bool) {
	if index, exists := op.nameMap[name]; exists {
		return &op.Parameters[index], true
	}
	return nil, false
}

// GetInOrder returns parameters in definition order
func (op *OrderedParameters) GetInOrder() []OrderedParameter {
	// Return copy of parameters (already in definition order)
	ordered := make([]OrderedParameter, len(op.Parameters))
	copy(ordered, op.Parameters)
	return ordered
}

// parseParametersWithOrder parses YAML parameters while preserving order
func parseParametersWithOrder(yamlNode *yaml.Node) (*OrderedParameters, error) {
	ordered := NewOrderedParameters()

	if yamlNode.Kind != yaml.DocumentNode {
		return nil, ErrExpectedDocumentNode
	}

	if len(yamlNode.Content) == 0 {
		return ordered, nil
	}

	rootNode := yamlNode.Content[0]
	if rootNode.Kind != yaml.MappingNode {
		return nil, ErrExpectedMappingNode
	}

	// Find parameters section
	var parametersNode *yaml.Node
	for i := 0; i < len(rootNode.Content); i += 2 {
		if i+1 < len(rootNode.Content) {
			keyNode := rootNode.Content[i]
			valueNode := rootNode.Content[i+1]

			if keyNode.Value == "parameters" {
				parametersNode = valueNode
				break
			}
		}
	}

	if parametersNode == nil {
		return ordered, nil // No parameters section
	}

	// Parse parameters
	err := parseParametersNode(parametersNode, ordered)
	if err != nil {
		return nil, err
	}

	return ordered, nil
}

// parseParametersNode recursively parses parameter nodes
func parseParametersNode(node *yaml.Node, ordered *OrderedParameters) error {
	if node.Kind != yaml.MappingNode {
		return ErrExpectedMappingForParams
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		paramName := keyNode.Value

		switch valueNode.Kind {
		case yaml.ScalarNode:
			// Simple type definition
			paramType := valueNode.Value
			ordered.Add(paramName, paramType)

		case yaml.MappingNode:
			// Nested object - convert to map[string]any
			nestedMap, err := convertYamlNodeToMap(valueNode)
			if err != nil {
				return err
			}
			ordered.Add(paramName, nestedMap)

		case yaml.SequenceNode:
			// Array type definition - convert to []any
			arrayType, err := convertYamlNodeToArray(valueNode)
			if err != nil {
				return err
			}
			ordered.Add(paramName, arrayType)

		default:
			return ErrUnsupportedParameterType
		}
	}

	return nil
}

// convertYamlNodeToMap converts YAML mapping node to map[string]any
func convertYamlNodeToMap(node *yaml.Node) (map[string]any, error) {
	if node.Kind != yaml.MappingNode {
		return nil, ErrExpectedMappingNode
	}

	result := make(map[string]any)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		value, err := convertYamlNodeToAny(valueNode)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, nil
}

// convertYamlNodeToArray converts YAML sequence node to []any
func convertYamlNodeToArray(node *yaml.Node) ([]any, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, ErrExpectedSequenceNode
	}

	result := make([]any, len(node.Content))
	for i, childNode := range node.Content {
		value, err := convertYamlNodeToAny(childNode)
		if err != nil {
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}

// convertYamlNodeToAny converts any YAML node to appropriate Go type
func convertYamlNodeToAny(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value, nil
	case yaml.MappingNode:
		return convertYamlNodeToMap(node)
	case yaml.SequenceNode:
		return convertYamlNodeToArray(node)
	default:
		return nil, ErrUnsupportedParameterType
	}
}
