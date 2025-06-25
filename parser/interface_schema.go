package parser

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

// InterfaceSchema represents the complete interface definition for SQL templates
// InterfaceSchema represents the complete interface definition for SQL templates
type InterfaceSchema struct {
	// Template metadata
	Name        string `yaml:"name"`        // Template name for function generation
	Description string `yaml:"description"` // Template description
	Version     string `yaml:"version"`     // Template version

	// Function generation
	FunctionName string `yaml:"function_name"` // Generated function name
	Package      string `yaml:"package"`       // Target package/namespace

	// Parameters definition (hierarchical structure preserved for type generation)
	Parameters map[string]any `yaml:"parameters"`

	// Additional metadata
	Tags     []string          `yaml:"tags"`     // Template tags
	Metadata map[string]string `yaml:"metadata"` // Additional metadata

	// Processed information
	OrderedParams *OrderedParameters `yaml:"-"` // Parameters with definition order preserved
}

// ProcessSchema processes the schema (currently minimal processing needed)
func (schema *InterfaceSchema) ProcessSchema() {
	// Initialize processed information if needed
	if schema.OrderedParams == nil {
		schema.OrderedParams = NewOrderedParameters()
	}
}

// ProcessSchemaWithOrder processes the schema with order preservation
func (schema *InterfaceSchema) ProcessSchemaWithOrder(orderedParams *OrderedParameters) {
	// Set ordered parameters
	schema.OrderedParams = orderedParams
}

// ValidateParameterReference validates if a parameter reference exists in the schema
// Uses hierarchical parameter traversal instead of flattened lookup
func (schema *InterfaceSchema) ValidateParameterReference(reference string) error {
	if !schema.hasParameterPath(reference, schema.Parameters) {
		return fmt.Errorf("%w: %s", ErrParameterNotFound, reference)
	}
	return nil
}

// hasParameterPath checks if a dot-separated parameter path exists in the hierarchical structure
func (schema *InterfaceSchema) hasParameterPath(path string, params map[string]any) bool {
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
func (schema *InterfaceSchema) GetParameterType(reference string) (string, bool) {
	paramType := schema.getParameterTypeFromPath(reference, schema.Parameters)
	return paramType, paramType != ""
}

// getParameterTypeFromPath traverses the hierarchical structure to find parameter type
func (schema *InterfaceSchema) getParameterTypeFromPath(path string, params map[string]any) string {
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
func (schema *InterfaceSchema) GetFunctionMetadata() map[string]string {
	metadata := make(map[string]string)

	if schema.Name != "" {
		metadata["name"] = schema.Name
	}
	if schema.FunctionName != "" {
		metadata["function_name"] = schema.FunctionName
	}
	if schema.Package != "" {
		metadata["package"] = schema.Package
	}
	if schema.Description != "" {
		metadata["description"] = schema.Description
	}
	if schema.Version != "" {
		metadata["version"] = schema.Version
	}

	// Add custom metadata
	for key, value := range schema.Metadata {
		metadata[key] = value
	}

	return metadata
}

// GetTags returns template tags
func (schema *InterfaceSchema) GetTags() []string {
	return schema.Tags
}

// NewInterfaceSchemaFromFrontMatter parses interface schema from frontmatter YAML with parameter order preservation
func NewInterfaceSchemaFromFrontMatter(frontmatterYAML string) (*InterfaceSchema, error) {
	if frontmatterYAML == "" {
		// Return minimal schema if no schema found
		return &InterfaceSchema{
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

	// Parse regular schema
	var schema InterfaceSchema
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse interface schema YAML: %w", err)
	}

	// Apply defaults (inline)
	if schema.Parameters == nil {
		schema.Parameters = make(map[string]any)
	}
	if schema.OrderedParams == nil {
		schema.OrderedParams = NewOrderedParameters()
	}

	// Process schema with order preservation (inline)
	schema.OrderedParams = orderedParams

	return &schema, nil
}

// NewInterfaceSchemaFromSQL parses interface schema from SQL tokens with comment blocks
// This is the preferred method as it avoids regex parsing and preserves parameter order
func NewInterfaceSchemaFromSQL(tokens []tokenizer.Token) (*InterfaceSchema, error) {
	// Extract from comment tokens (inline extractCommentSchemaFromTokens)
	var commentBlocks []string

	for _, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT {
			content := strings.TrimSpace(token.Value)

			// Only process comments that start with /*@ (schema marker)
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

	var schemaYAML string
	if len(commentBlocks) > 0 {
		schemaYAML = commentBlocks[0]
	}

	// Delegate to NewInterfaceSchemaFromFrontMatter
	return NewInterfaceSchemaFromFrontMatter(schemaYAML)
}
