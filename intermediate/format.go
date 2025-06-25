package intermediate

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/shibukawa/snapsql/parser"
)

// IntermediateFormat represents the complete intermediate JSON format for SnapSQL templates
type IntermediateFormat struct {
	Source          SourceInfo                `json:"source"`
	InterfaceSchema *InterfaceSchemaFormatted `json:"interface_schema,omitempty"`
	AST             ASTNode                   `json:"ast"`
}

// SourceInfo represents source file information
type SourceInfo struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

// InterfaceSchemaFormatted represents the simplified interface schema for JSON output
type InterfaceSchemaFormatted struct {
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	FunctionName string      `json:"function_name,omitempty"`
	Parameters   []Parameter `json:"parameters,omitempty"`
}

// Parameter represents a parameter in the recursive structure
type Parameter struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []Parameter `json:"children,omitempty"`
}

// ASTNode represents a serializable AST node
type ASTNode struct {
	Type     string         `json:"type"`
	Pos      [3]int         `json:"pos"`
	Children map[string]any `json:",inline"`
}

// NewFormat creates a new intermediate format instance
func NewFormat() *IntermediateFormat {
	return &IntermediateFormat{}
}

// SetSource sets the source information
func (f *IntermediateFormat) SetSource(file, content string) {
	f.Source = SourceInfo{
		File:    file,
		Content: content,
	}
}

// SetInterfaceSchema sets the interface schema from parser.InterfaceSchema
func (f *IntermediateFormat) SetInterfaceSchema(schema *parser.InterfaceSchema) {
	if schema == nil {
		return
	}

	f.InterfaceSchema = &InterfaceSchemaFormatted{
		Name:         schema.Name,
		Description:  schema.Description,
		FunctionName: schema.FunctionName,
		Parameters:   convertParameters(schema.OrderedParams),
	}
}

// SetAST sets the AST from parser AST node
func (f *IntermediateFormat) SetAST(ast parser.AstNode) {
	f.AST = convertASTNode(ast)
}

// WriteJSON writes the intermediate format as JSON to the provided writer
func (f *IntermediateFormat) WriteJSON(w io.Writer, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(f)
}

// ToJSON serializes the intermediate format to JSON (deprecated: use WriteJSON)
func (f *IntermediateFormat) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

// ToJSONPretty serializes the intermediate format to pretty-printed JSON (deprecated: use WriteJSON)
func (f *IntermediateFormat) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// convertParameters converts OrderedParameters to Parameter slice
func convertParameters(orderedParams *parser.OrderedParameters) []Parameter {
	if orderedParams == nil {
		return nil
	}

	inOrder := orderedParams.GetInOrder()
	params := make([]Parameter, len(inOrder))

	for i, param := range inOrder {
		params[i] = convertParameter(param)
	}

	return params
}

// convertParameter converts a single OrderedParameter to Parameter
func convertParameter(param parser.OrderedParameter) Parameter {
	p := Parameter{
		Name: param.Name,
		Type: convertType(param.Type),
	}

	// Handle nested structures recursively
	if children := extractChildren(param.Type); len(children) > 0 {
		p.Children = children
	}

	return p
}

// convertType converts parameter type to string representation
func convertType(paramType any) string {
	switch t := paramType.(type) {
	case string:
		return t
	case []any:
		if len(t) > 0 {
			return "array"
		}
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

// extractChildren extracts children from complex types
func extractChildren(paramType any) []Parameter {
	var children []Parameter
	objectCounter := 1

	switch t := paramType.(type) {
	case []any:
		// Array type - create children for array elements
		for _, elem := range t {
			child := Parameter{
				Name: generateObjectName(&objectCounter),
				Type: convertType(elem),
			}
			if grandChildren := extractChildren(elem); len(grandChildren) > 0 {
				child.Children = grandChildren
			}
			children = append(children, child)
		}
	case map[string]any:
		// Object type - create children for object properties
		for key, value := range t {
			child := Parameter{
				Name: key,
				Type: convertType(value),
			}
			if grandChildren := extractChildren(value); len(grandChildren) > 0 {
				child.Children = grandChildren
			}
			children = append(children, child)
		}
	}

	return children
}

// generateObjectName generates object names like o1, o2, etc.
func generateObjectName(counter *int) string {
	name := fmt.Sprintf("o%d", *counter)
	*counter++
	return name
}

// convertASTNode converts parser.AstNode to serializable ASTNode
func convertASTNode(node parser.AstNode) ASTNode {
	if node == nil {
		return ASTNode{}
	}

	pos := node.Position()
	astNode := ASTNode{
		Type:     node.Type().String(),
		Pos:      [3]int{pos.Line, pos.Column, pos.Offset},
		Children: make(map[string]any),
	}

	// Add node-specific fields based on type
	switch n := node.(type) {
	case *parser.SelectStatement:
		if n.SelectClause != nil {
			astNode.Children["select_clause"] = convertASTNode(n.SelectClause)
		}
		if n.FromClause != nil {
			astNode.Children["from_clause"] = convertASTNode(n.FromClause)
		}
		if n.WhereClause != nil {
			astNode.Children["where_clause"] = convertASTNode(n.WhereClause)
		}
		if n.OrderByClause != nil {
			astNode.Children["order_by_clause"] = convertASTNode(n.OrderByClause)
		}
		if n.GroupByClause != nil {
			astNode.Children["group_by_clause"] = convertASTNode(n.GroupByClause)
		}
		if n.HavingClause != nil {
			astNode.Children["having_clause"] = convertASTNode(n.HavingClause)
		}
		if n.LimitClause != nil {
			astNode.Children["limit_clause"] = convertASTNode(n.LimitClause)
		}
		if n.OffsetClause != nil {
			astNode.Children["offset_clause"] = convertASTNode(n.OffsetClause)
		}
	case *parser.SelectClause:
		if len(n.Fields) > 0 {
			var fields []ASTNode
			for _, field := range n.Fields {
				fields = append(fields, convertASTNode(field))
			}
			astNode.Children["fields"] = fields
		}
	case *parser.FromClause:
		if len(n.Tables) > 0 {
			var tables []ASTNode
			for _, table := range n.Tables {
				tables = append(tables, convertASTNode(table))
			}
			astNode.Children["tables"] = tables
		}
	case *parser.WhereClause:
		if n.Condition != nil {
			astNode.Children["condition"] = convertASTNode(n.Condition)
		}
	case *parser.Identifier:
		astNode.Children["name"] = n.Name
	case *parser.VariableSubstitution:
		astNode.Children["expression"] = n.Expression
		astNode.Children["dummy_value"] = n.DummyValue
	}

	return astNode
}
