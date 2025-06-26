package snapsqlgo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
)

// Sentinel errors
var (
	ErrUnknownInstruction = errors.New("unknown instruction")
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op          string `json:"op"`
	Pos         []int  `json:"pos"`                   // Position [line, column, offset] from original template (required)
	Value       string `json:"value,omitempty"`       // For EMIT_LITERAL
	Param       string `json:"param,omitempty"`       // For EMIT_PARAM
	Exp         string `json:"exp,omitempty"`         // For EMIT_EVAL, JUMP_IF_EXP
	Placeholder string `json:"placeholder,omitempty"` // For EMIT_PARAM, EMIT_EVAL
	Target      int    `json:"target,omitempty"`      // For JUMP, JUMP_IF_EXP
	Name        string `json:"name,omitempty"`        // For LABEL
	Variable    string `json:"variable,omitempty"`    // For LOOP_START, LOOP_END
	Collection  string `json:"collection,omitempty"`  // For LOOP_START
	EndLabel    string `json:"end_label,omitempty"`   // For LOOP_START
	StartLabel  string `json:"start_label,omitempty"` // For LOOP_NEXT
	Label       string `json:"label,omitempty"`       // For LOOP_END
}

// LoopState represents the state of a loop during execution
type LoopState struct {
	Variable   string
	Collection []any
	Index      int
	StartPC    int
}

// InstructionExecutor executes a sequence of instructions
type InstructionExecutor struct {
	instructions []Instruction
	pc           int
	params       map[string]any
	output       strings.Builder
	parameters   []any
	loops        []LoopState
	variables    map[string]any // Loop variables with scoping
	celEnv       *cel.Env
}

// NewInstructionExecutor creates a new instruction executor
func NewInstructionExecutor(instructions []Instruction, params map[string]any) (*InstructionExecutor, error) {
	// Create simple CEL environment - we'll handle variable resolution manually
	env, err := cel.NewEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &InstructionExecutor{
		instructions: instructions,
		pc:           0,
		params:       params,
		parameters:   make([]any, 0),
		loops:        make([]LoopState, 0),
		variables:    make(map[string]any),
		celEnv:       env,
	}, nil
}

// Execute runs the instruction sequence and returns the generated SQL and parameters
func (e *InstructionExecutor) Execute() (string, []any, error) {
	e.pc = 0
	e.output.Reset()
	e.parameters = e.parameters[:0]
	e.loops = e.loops[:0]
	e.variables = make(map[string]any)

	for e.pc < len(e.instructions) {
		if err := e.executeInstruction(e.instructions[e.pc]); err != nil {
			return "", nil, fmt.Errorf("error executing instruction at PC %d: %w", e.pc, err)
		}
		e.pc++
	}

	return e.output.String(), e.parameters, nil
}

// executeInstruction executes a single instruction
func (e *InstructionExecutor) executeInstruction(inst Instruction) error {
	switch inst.Op {
	case "EMIT_LITERAL":
		e.output.WriteString(inst.Value)
	case "EMIT_PARAM":
		e.output.WriteString("?")
		e.addParameter(inst.Param)
	case "EMIT_EVAL":
		e.output.WriteString("?")
		e.addExpression(inst.Exp)
	case "JUMP":
		e.pc = inst.Target - 1
	case "JUMP_IF_EXP":
		if e.isExpressionTruthy(inst.Exp) {
			e.pc = inst.Target - 1
		}
	case "LOOP_START":
		collection := e.evaluateCELExpression(inst.Collection)
		if collectionSlice, ok := collection.([]any); ok && len(collectionSlice) > 0 {
			// Set loop variable to first element
			e.variables[inst.Variable] = collectionSlice[0]
			// Push loop state
			e.loops = append(e.loops, LoopState{
				Variable:   inst.Variable,
				Collection: collectionSlice,
				Index:      0,
				StartPC:    e.pc + 1,
			})
		} else {
			// Empty collection, jump to LOOP_END instruction
			endIndex := e.findLoopEnd(inst.Variable)
			if endIndex >= 0 {
				e.pc = endIndex - 1
			}
		}
	case "LOOP_NEXT":
		if len(e.loops) > 0 {
			loop := &e.loops[len(e.loops)-1]
			loop.Index++
			if loop.Index < len(loop.Collection) {
				// Update loop variable and jump back
				e.variables[loop.Variable] = loop.Collection[loop.Index]
				e.pc = loop.StartPC - 1
			}
			// Otherwise continue to LOOP_END
		}
	case "LOOP_END":
		if len(e.loops) > 0 {
			// Remove loop variable from scope
			delete(e.variables, inst.Variable)
			// Pop loop state
			e.loops = e.loops[:len(e.loops)-1]
		}
	case "LABEL":
		// No operation, just a marker
	case "NOP":
		// No operation
	default:
		return fmt.Errorf("%w: %s", ErrUnknownInstruction, inst.Op)
	}
	return nil
}

// addParameter adds a simple variable value to parameter list
func (e *InstructionExecutor) addParameter(paramName string) {
	value := e.getVariableValue(paramName)
	e.parameters = append(e.parameters, value)
}

// addExpression evaluates CEL expression and adds result to parameter list
func (e *InstructionExecutor) addExpression(celExpression string) {
	value := e.evaluateCELExpression(celExpression)
	e.parameters = append(e.parameters, value)
}

// getVariableValue gets variable value with scoping (loop variables shadow parameters)
func (e *InstructionExecutor) getVariableValue(name string) any {
	// Check loop variables first (they shadow parameters)
	if value, exists := e.variables[name]; exists {
		return value
	}
	// Fall back to input parameters
	return e.getParamValue(name)
}

// getParamValue gets value from input parameters using dot notation
func (e *InstructionExecutor) getParamValue(path string) any {
	parts := strings.Split(path, ".")
	var current any = e.params

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			if currentMap, ok := current.(map[string]any); ok {
				return currentMap[part]
			}
			return nil
		}
		// Navigate deeper
		if currentMap, ok := current.(map[string]any); ok {
			if next, exists := currentMap[part]; exists {
				current = next
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}

// evaluateCELExpression evaluates a CEL expression
func (e *InstructionExecutor) evaluateCELExpression(expression string) any {
	// Handle simple cases first
	if result := e.evaluateSimpleExpression(expression); result != nil {
		return result
	}

	// For complex expressions, try CEL
	vars := e.getAllVariables()

	// Create a CEL environment with all variables
	envOptions := []cel.EnvOption{}
	for key := range vars {
		envOptions = append(envOptions, cel.Variable(key, cel.AnyType))
	}

	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return nil
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil
	}

	result, _, err := prg.Eval(vars)
	if err != nil {
		return nil
	}

	return result.Value()
}

// evaluateSimpleExpression handles simple expressions without CEL
func (e *InstructionExecutor) evaluateSimpleExpression(expression string) any {
	vars := e.getAllVariables()

	// Handle negation
	if strings.HasPrefix(expression, "!") {
		inner := strings.TrimPrefix(expression, "!")
		if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
			inner = strings.TrimPrefix(strings.TrimSuffix(inner, ")"), "(")
		}
		innerResult := e.evaluateSimpleExpression(inner)
		return !e.isTruthy(innerResult)
	}

	// Handle OR expressions
	if strings.Contains(expression, " || ") {
		parts := strings.Split(expression, " || ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if e.isTruthy(e.evaluateSimpleExpression(part)) {
				return true
			}
		}
		return false
	}

	// Handle AND expressions
	if strings.Contains(expression, " && ") {
		parts := strings.Split(expression, " && ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !e.isTruthy(e.evaluateSimpleExpression(part)) {
				return false
			}
		}
		return true
	}

	// Handle simple variable lookup
	if value, exists := vars[expression]; exists {
		return value
	}

	// Handle dot notation
	if strings.Contains(expression, ".") {
		return e.getParamValueFromVars(expression, vars)
	}

	return nil
}

// getAllVariables returns all available variables (params + loop variables)
func (e *InstructionExecutor) getAllVariables() map[string]any {
	vars := make(map[string]any)

	// Add all parameters
	for k, v := range e.params {
		vars[k] = v
	}

	// Add loop variables (they shadow parameters)
	for k, v := range e.variables {
		vars[k] = v
	}

	return vars
}

// isTruthy determines if a value is truthy
func (e *InstructionExecutor) isTruthy(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case nil:
		return false
	case string:
		return v != ""
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

// getParamValueFromVars gets value from variables map using dot notation
func (e *InstructionExecutor) getParamValueFromVars(path string, vars map[string]any) any {
	parts := strings.Split(path, ".")
	var current any = vars

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			if currentMap, ok := current.(map[string]any); ok {
				return currentMap[part]
			}
			return nil
		}
		// Navigate deeper
		if currentMap, ok := current.(map[string]any); ok {
			if next, exists := currentMap[part]; exists {
				current = next
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}

// isExpressionTruthy evaluates a CEL expression and returns its truthiness
func (e *InstructionExecutor) isExpressionTruthy(expression string) bool {
	result := e.evaluateCELExpression(expression)
	return e.isTruthy(result)
}

// findLoopEnd finds the LOOP_END instruction for a given variable
func (e *InstructionExecutor) findLoopEnd(variable string) int {
	for i := e.pc + 1; i < len(e.instructions); i++ {
		inst := e.instructions[i]
		if inst.Op == "LOOP_END" && inst.Variable == variable {
			return i
		}
	}
	return -1
}
