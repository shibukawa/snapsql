package snapsqlgo

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrUnknownNodeType = errors.New("unknown node type")
)

// InstructionCompiler compiles AST nodes to instruction sequences
type InstructionCompiler struct {
	instructions []Instruction
	labelCounter int
}

// NewInstructionCompiler creates a new instruction compiler
func NewInstructionCompiler() *InstructionCompiler {
	return &InstructionCompiler{
		instructions: make([]Instruction, 0),
		labelCounter: 0,
	}
}

// ASTNode represents a node in the Abstract Syntax Tree
// This is a simplified version for demonstration
type ASTNode struct {
	Type        string     `json:"type"`
	Pos         []int      `json:"pos"`                   // Position [line, column, offset] (required)
	Value       string     `json:"value,omitempty"`
	Placeholder string     `json:"placeholder,omitempty"`
	Condition   string     `json:"condition,omitempty"`
	Variable    string     `json:"variable,omitempty"`
	Collection  string     `json:"collection,omitempty"`
	Body        *ASTNode   `json:"body,omitempty"`
	ElseBody    *ASTNode   `json:"else_body,omitempty"`
	Children    []*ASTNode `json:"children,omitempty"`
}

// Compile compiles an AST to instruction sequence
func (c *InstructionCompiler) Compile(ast *ASTNode) ([]Instruction, error) {
	c.instructions = make([]Instruction, 0)
	c.labelCounter = 0

	err := c.compileNode(ast)
	if err != nil {
		return nil, err
	}

	return c.instructions, nil
}

// compileNode compiles a single AST node
func (c *InstructionCompiler) compileNode(node *ASTNode) error {
	if node == nil {
		return nil
	}

	switch node.Type {
	case "LITERAL":
		c.emitLiteral(node.Value, node.Pos)
	case "VARIABLE":
		c.emitParam(node.Value, node.Placeholder, node.Pos)
	case "EXPRESSION":
		c.emitEval(node.Value, node.Placeholder, node.Pos)
	case "IF_BLOCK":
		return c.compileIfBlock(node)
	case "FOR_BLOCK":
		return c.compileForBlock(node)
	case "SEQUENCE":
		return c.compileSequence(node)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownNodeType, node.Type)
	}

	return nil
}

// compileSequence compiles a sequence of nodes
func (c *InstructionCompiler) compileSequence(node *ASTNode) error {
	for _, child := range node.Children {
		if err := c.compileNode(child); err != nil {
			return err
		}
	}
	return nil
}

// compileIfBlock compiles an IF block to conditional jump instructions
func (c *InstructionCompiler) compileIfBlock(node *ASTNode) error {
	endLabel := c.generateLabel("if_end")
	elseLabel := c.generateLabel("if_else")

	// JUMP_IF_EXP: if condition is false, jump to else or end
	var targetIndex int
	if node.ElseBody != nil {
		targetIndex = -1 // Will be resolved to else label
	} else {
		targetIndex = -1 // Will be resolved to end label
	}

	// Negate condition for jump (jump if condition is false)
	negatedCondition := fmt.Sprintf("!(%s)", node.Condition)
	c.emitJumpIfExp(negatedCondition, targetIndex, node.Pos) // Target will be resolved later
	jumpIndex := len(c.instructions) - 1

	// Compile body
	if err := c.compileNode(node.Body); err != nil {
		return err
	}

	// If there's an else body, jump over it
	if node.ElseBody != nil {
		c.emitJump(-1, node.Pos) // Target will be resolved later
		jumpToEndIndex := len(c.instructions) - 1

		// Else label
		c.emitLabel(elseLabel, node.Pos)
		elseIndex := len(c.instructions) - 1

		// Compile else body
		if err := c.compileNode(node.ElseBody); err != nil {
			return err
		}

		// Resolve jump to end
		c.instructions[jumpToEndIndex].Target = len(c.instructions)

		// Resolve jump to else
		c.instructions[jumpIndex].Target = elseIndex
	} else {
		// Resolve jump to end
		c.instructions[jumpIndex].Target = len(c.instructions)
	}

	// End label
	c.emitLabel(endLabel, node.Pos)

	return nil
}

// compileForBlock compiles a FOR block to loop instructions
func (c *InstructionCompiler) compileForBlock(node *ASTNode) error {
	startLabel := c.generateLabel("loop_start")
	endLabel := c.generateLabel("loop_end")

	// LOOP_START instruction
	c.emitLoopStart(node.Variable, node.Collection, endLabel, node.Pos)

	// Label for loop start
	c.emitLabel(startLabel, node.Pos)

	// Compile loop body
	if err := c.compileNode(node.Body); err != nil {
		return err
	}

	// LOOP_NEXT instruction
	c.emitLoopNext(startLabel, node.Pos)

	// LOOP_END instruction
	c.emitLoopEnd(node.Variable, endLabel, node.Pos)

	return nil
}

// Instruction emission methods (all require pos information)
func (c *InstructionCompiler) emitLiteral(value string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:    "EMIT_LITERAL",
		Pos:   pos,
		Value: value,
	})
}

func (c *InstructionCompiler) emitParam(param, placeholder string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:          "EMIT_PARAM",
		Pos:         pos,
		Param:       param,
		Placeholder: placeholder,
	})
}

func (c *InstructionCompiler) emitEval(expression, placeholder string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:          "EMIT_EVAL",
		Pos:         pos,
		Exp:         expression,
		Placeholder: placeholder,
	})
}

func (c *InstructionCompiler) emitJump(target int, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:     "JUMP",
		Pos:    pos,
		Target: target,
	})
}

func (c *InstructionCompiler) emitJumpIfExp(expression string, target int, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:     "JUMP_IF_EXP",
		Pos:    pos,
		Exp:    expression,
		Target: target,
	})
}

func (c *InstructionCompiler) emitLabel(name string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:   "LABEL",
		Pos:  pos,
		Name: name,
	})
}

func (c *InstructionCompiler) emitLoopStart(variable, collection, endLabel string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:         "LOOP_START",
		Pos:        pos,
		Variable:   variable,
		Collection: collection,
		EndLabel:   endLabel,
	})
}

func (c *InstructionCompiler) emitLoopNext(startLabel string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:         "LOOP_NEXT",
		Pos:        pos,
		StartLabel: startLabel,
	})
}

func (c *InstructionCompiler) emitLoopEnd(variable, label string, pos []int) {
	c.instructions = append(c.instructions, Instruction{
		Op:       "LOOP_END",
		Pos:      pos,
		Variable: variable,
		Label:    label,
	})
}

// generateLabel generates a unique label name
func (c *InstructionCompiler) generateLabel(prefix string) string {
	c.labelCounter++
	return fmt.Sprintf("%s_%d", prefix, c.labelCounter)
}
