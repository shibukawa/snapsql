package parser

import "github.com/shibukawa/snapsql/tokenizer"

// DeferredVariableSubstitution represents deferred variable substitution
type DeferredVariableSubstitution struct {
	BaseAstNode
	Expression string     // CEL expression
	DummyValue string     // Dummy literal
	Namespace  *Namespace // Namespace used during validation
}

// GetNodeType returns the node type
func (d *DeferredVariableSubstitution) GetNodeType() NodeType {
	return d.nodeType
}

// GetPosition returns the position
func (d *DeferredVariableSubstitution) GetPosition() tokenizer.Position {
	return d.position
}

// String returns string representation
func (d *DeferredVariableSubstitution) String() string {
	return "DeferredVariableSubstitution"
}
