package parsercommon

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
	"gopkg.in/yaml.v3"
)

var (
	ErrParameterNotFound = fmt.Errorf("parameter not found")
)

// CELVariable represents a CEL variable definition
type CELVariable struct {
	Name string // Variable name (dot notation for nested)
	Type string // CEL type
}

// FunctionDefinition represents the complete interface definition for SQL templates
type FunctionDefinition struct {
	// Template metadata
	Name        string `yaml:"name"`        // Template name for function generation
	Description string `yaml:"description"` // Template description

	// Function generation
	FunctionName string `yaml:"function_name"` // Generated function name

	// Parameters definition (hierarchical structure preserved for type generation)
	Parameters map[string]any `yaml:"parameters"`

	// Processed information
	OrderedParams *OrderedParameters `yaml:"-"` // Parameters with definition order preserved
}

// ProcessDefinition processes the def (currently minimal processing needed)
func (def *FunctionDefinition) ProcessDefinition() {
	// Initialize processed information if needed
	if def.OrderedParams == nil {
		def.OrderedParams = NewOrderedParameters()
	}
}

// ProcessDefinitionWithOrder processes the def with order preservation
func (def *FunctionDefinition) ProcessDefinitionWithOrder(orderedParams *OrderedParameters) {
	// Set ordered parameters
	def.OrderedParams = orderedParams
}

// ValidateParameterReference validates if a parameter reference exists in the def
// Uses hierarchical parameter traversal instead of flattened lookup
func (def *FunctionDefinition) ValidateParameterReference(reference string) error {
	if !def.hasParameterPath(reference, def.Parameters) {
		return fmt.Errorf("%w: %s", ErrParameterNotFound, reference)
	}
	return nil
}

// hasParameterPath checks if a dot-separated parameter path exists in the hierarchical structure
func (def *FunctionDefinition) hasParameterPath(path string, params map[string]any) bool {
	parts := strings.Split(path, ".")
	current := params

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return false
		}

		// If this is the last part, we found the parameter
		if i == len(parts)-1 {
			return true
		}

		// Otherwise, continue traversing
		if nested, ok := value.(map[string]any); ok {
			current = nested
		} else {
			return false // Path continues but current value is not a map
		}
	}

	return false
}

// GetParameterType returns the type of a parameter using hierarchical traversal
func (def *FunctionDefinition) GetParameterType(reference string) (string, bool) {
	paramType := def.getParameterTypeFromPath(reference, def.Parameters)
	return paramType, paramType != ""
}

// getParameterTypeFromPath traverses the hierarchical structure to find parameter type
func (def *FunctionDefinition) getParameterTypeFromPath(path string, params map[string]any) string {
	parts := strings.Split(path, ".")
	current := params

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return ""
		}

		// If this is the last part, return the type
		if i == len(parts)-1 {
			if typeStr, ok := value.(string); ok {
				return typeStr
			}
			// Handle new slice format []any{"str"}
			if slice, ok := value.([]any); ok && len(slice) > 0 {
				if elementType, ok := slice[0].(string); ok {
					return "list[" + elementType + "]" // Convert to old format for compatibility
				}
			}
			return ""
		}

		// Otherwise, continue traversing
		if nested, ok := value.(map[string]any); ok {
			current = nested
		} else {
			return "" // Path continues but current value is not a map
		}
	}

	return ""
}

// GetFunctionMetadata returns metadata for function generation
func (def *FunctionDefinition) GetFunctionMetadata() map[string]string {
	metadata := make(map[string]string)

	if def.Name != "" {
		metadata["name"] = def.Name
	}
	if def.FunctionName != "" {
		metadata["function_name"] = def.FunctionName
	}
	if def.Description != "" {
		metadata["description"] = def.Description
	}

	return metadata
}

// GetTags returns empty slice (tags removed from def)
func (def *FunctionDefinition) GetTags() []string {
	return []string{}
}

// NewFunctionDefinitionFromFrontMatter parses interface def from frontmatter YAML with parameter order preservation
func NewFunctionDefinitionFromFrontMatter(frontmatterYAML string) (*FunctionDefinition, error) {
	if frontmatterYAML == "" {
		// Return minimal def if no def found
		return &FunctionDefinition{
			Parameters:    make(map[string]any),
			OrderedParams: NewOrderedParameters(),
		}, nil
	}

	// Parse YAML with order preservation
	var yamlNode yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &yamlNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML node: %w", err)
	}

	// Extract ordered parameters
	orderedParams, err := parseParametersWithOrder(&yamlNode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ordered parameters: %w", err)
	}

	// Parse regular def
	var def FunctionDefinition
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &def); err != nil {
		return nil, fmt.Errorf("failed to parse interface def YAML: %w", err)
	}

	// Apply defaults (inline)
	if def.Parameters == nil {
		def.Parameters = make(map[string]any)
	}
	if def.OrderedParams == nil {
		def.OrderedParams = NewOrderedParameters()
	}

	// Process def with order preservation (inline)
	def.OrderedParams = orderedParams

	return &def, nil
}

// NewFunctionDefinitionFromSQL parses interface def from SQL tokens with comment blocks
// This is the preferred method as it avoids regex parsing and preserves parameter order
func NewFunctionDefinitionFromSQL(tokens []tokenizer.Token) (*FunctionDefinition, error) {
	// Extract from comment tokens (inline extractCommentDefinitionFromTokens)
	var commentBlocks []string

	for _, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT {
			content := strings.TrimSpace(token.Value)

			// Only process comments that start with /*@ (def marker)
			if strings.HasPrefix(content, "/*@") && strings.HasSuffix(content, "*/") {
				// Remove /*@ and */ markers
				content = strings.TrimSpace(content[3 : len(content)-2])

				// This should be YAML content
				if content != "" {
					commentBlocks = append(commentBlocks, content)
				}
			}
		}
	}

	var defYAML string
	if len(commentBlocks) > 0 {
		defYAML = commentBlocks[0]
	}

	// Delegate to NewInterfaceDefinitionFromFrontMatter
	return NewFunctionDefinitionFromFrontMatter(defYAML)
}

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
