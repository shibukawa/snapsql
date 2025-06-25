package parser

import (
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// OrderedParameter represents a parameter with order information
type OrderedParameter struct {
	Name  string // Parameter name
	Type  any    // Parameter type
	Order int    // Order in YAML definition
	Path  string // Full path (for nested parameters)
}

// OrderedParameters holds parameters with their definition order
type OrderedParameters struct {
	Parameters []OrderedParameter // Parameters in definition order
	pathMap    map[string]int     // Path to index mapping for quick lookup
}

// NewOrderedParameters creates a new ordered parameters container
func NewOrderedParameters() *OrderedParameters {
	return &OrderedParameters{
		Parameters: make([]OrderedParameter, 0),
		pathMap:    make(map[string]int),
	}
}

// Add adds a parameter with order information
func (op *OrderedParameters) Add(name string, paramType any, path string, order int) {
	param := OrderedParameter{
		Name:  name,
		Type:  paramType,
		Order: order,
		Path:  path,
	}

	op.Parameters = append(op.Parameters, param)
	op.pathMap[path] = len(op.Parameters) - 1
}

// GetByPath returns parameter by path
func (op *OrderedParameters) GetByPath(path string) (*OrderedParameter, bool) {
	if index, exists := op.pathMap[path]; exists {
		return &op.Parameters[index], true
	}
	return nil, false
}

// GetInOrder returns parameters sorted by definition order
func (op *OrderedParameters) GetInOrder() []OrderedParameter {
	// Create a copy and sort by order
	ordered := make([]OrderedParameter, len(op.Parameters))
	copy(ordered, op.Parameters)

	slices.SortFunc(ordered, func(a, b OrderedParameter) int {
		if a.Order < b.Order {
			return -1
		} else if a.Order > b.Order {
			return 1
		}
		return 0
	})

	return ordered
}

// GetTopLevelInOrder returns only top-level parameters in definition order
func (op *OrderedParameters) GetTopLevelInOrder() []OrderedParameter {
	var topLevel []OrderedParameter

	for _, param := range op.GetInOrder() {
		// Top-level parameters don't contain dots
		if !strings.Contains(param.Path, ".") {
			topLevel = append(topLevel, param)
		}
	}

	return topLevel
}

// GetNestedInOrder returns nested parameters under a given path in definition order
func (op *OrderedParameters) GetNestedInOrder(parentPath string) []OrderedParameter {
	var nested []OrderedParameter
	prefix := parentPath + "."

	for _, param := range op.GetInOrder() {
		if strings.HasPrefix(param.Path, prefix) {
			// Only direct children (no further nesting)
			remaining := strings.TrimPrefix(param.Path, prefix)
			if !strings.Contains(remaining, ".") {
				nested = append(nested, param)
			}
		}
	}

	return nested
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

	// Parse parameters with order
	err := parseParametersNode(parametersNode, "", ordered, 0)
	if err != nil {
		return nil, err
	}

	return ordered, nil
}

// parseParametersNode recursively parses parameter nodes
func parseParametersNode(node *yaml.Node, prefix string, ordered *OrderedParameters, baseOrder int) error {
	if node.Kind != yaml.MappingNode {
		return ErrExpectedMappingForParams
	}

	order := baseOrder

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		paramName := keyNode.Value
		fullPath := paramName
		if prefix != "" {
			fullPath = prefix + "." + paramName
		}

		switch valueNode.Kind {
		case yaml.ScalarNode:
			// Simple type definition
			paramType := valueNode.Value
			ordered.Add(paramName, paramType, fullPath, order)
			order++

		case yaml.MappingNode:
			// Nested object - check if it's a type definition or nested parameters
			if isTypeDefinition(valueNode) {
				// This is a complex type definition (like []any{"str"})
				paramType := extractTypeFromNode(valueNode)
				ordered.Add(paramName, paramType, fullPath, order)
				order++
			} else {
				// Nested parameters
				err := parseParametersNode(valueNode, fullPath, ordered, order)
				if err != nil {
					return err
				}
				// Update order based on nested parameters count
				order += countNestedParameters(valueNode)
			}

		case yaml.SequenceNode:
			// Array type definition - use nested slice format
			elementType := extractArrayElementType(valueNode)
			paramType := []any{elementType}
			ordered.Add(paramName, paramType, fullPath, order)
			order++

		default:
			return ErrUnsupportedParameterType
		}
	}

	return nil
}

// isTypeDefinition checks if a mapping node represents a type definition
func isTypeDefinition(node *yaml.Node) bool {
	// Simple heuristic: if it contains "type" key, it's a type definition
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) {
			keyNode := node.Content[i]
			if keyNode.Value == "type" {
				return true
			}
		}
	}
	return false
}

// extractTypeFromNode extracts type from a type definition node
func extractTypeFromNode(node *yaml.Node) string {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.Value == "type" {
				return valueNode.Value
			}
		}
	}
	return "any"
}

// extractArrayElementType extracts element type from array definition
func extractArrayElementType(node *yaml.Node) string {
	if len(node.Content) > 0 {
		firstElement := node.Content[0]
		if firstElement.Kind == yaml.ScalarNode {
			return firstElement.Value
		}
	}
	return "any"
}

// countNestedParameters counts the number of nested parameters
func countNestedParameters(node *yaml.Node) int {
	if node.Kind != yaml.MappingNode {
		return 0
	}

	count := 0
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) {
			valueNode := node.Content[i+1]
			if valueNode.Kind == yaml.MappingNode && !isTypeDefinition(valueNode) {
				count += 1 + countNestedParameters(valueNode)
			} else {
				count++
			}
		}
	}

	return count
}
