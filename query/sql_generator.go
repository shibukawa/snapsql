package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/shibukawa/snapsql/intermediate"
)

// Error definitions for SQL generation
var (
	ErrInvalidExpressionIndex = errors.New("invalid expression index")
	ErrParameterNotFound      = errors.New("parameter not found")
	ErrExpressionEvaluation   = errors.New("expression evaluation failed")
	ErrUnsupportedOperation   = errors.New("unsupported operation")
)

// SQLGenerator generates SQL from intermediate format instructions
type SQLGenerator struct {
	instructions []intermediate.Instruction
	expressions  []intermediate.CELExpression
	dialect      string
	celEnv       *cel.Env
}

// NewSQLGenerator creates a new SQL generator
func NewSQLGenerator(instructions []intermediate.Instruction, expressions []intermediate.CELExpression, dialect string) *SQLGenerator {
	// Create CEL environment
	env, _ := cel.NewEnv(
		cel.Variable("params", cel.MapType(cel.StringType, cel.AnyType)),
	)

	return &SQLGenerator{
		instructions: instructions,
		expressions:  expressions,
		dialect:      dialect,
		celEnv:       env,
	}
}

// Generate generates SQL and parameters from the instructions
func (g *SQLGenerator) Generate(params map[string]interface{}) (string, []interface{}, error) {
	var result strings.Builder

	sqlParams := make([]interface{}, 0) // Initialize as empty slice instead of nil

	var conditionStack []bool // Stack to track if/else conditions

	for i, instr := range g.instructions {
		// Skip instructions if we're in a false condition block
		if len(conditionStack) > 0 && !conditionStack[len(conditionStack)-1] {
			if instr.Op == intermediate.OpEnd || instr.Op == intermediate.OpElse || instr.Op == intermediate.OpElseIf {
				// These operations need to be processed even in false blocks
			} else {
				continue
			}
		}

		switch instr.Op {
		case intermediate.OpEmitStatic:
			result.WriteString(instr.Value)

		case intermediate.OpEmitEval:
			if instr.ExprIndex == nil {
				return "", nil, fmt.Errorf("%w: instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return "", nil, fmt.Errorf("%w: instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			value, err := g.evaluateExpression(expr.Expression, params)
			if err != nil {
				return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			result.WriteString("?")

			sqlParams = append(sqlParams, value)

		case intermediate.OpIf:
			if instr.ExprIndex == nil {
				return "", nil, fmt.Errorf("%w: IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return "", nil, fmt.Errorf("%w: IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			condition, err := g.evaluateCondition(expr.Expression, params)
			if err != nil {
				return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			conditionStack = append(conditionStack, condition)

		case intermediate.OpElseIf:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: ELSE_IF without matching IF", ErrUnsupportedOperation)
			}

			// If the current condition is true, set it to false (we already processed the true branch)
			if conditionStack[len(conditionStack)-1] {
				conditionStack[len(conditionStack)-1] = false
			} else {
				// Evaluate the else-if condition
				if instr.ExprIndex == nil {
					return "", nil, fmt.Errorf("%w: ELSE_IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
				}

				if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
					return "", nil, fmt.Errorf("%w: ELSE_IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
				}

				expr := g.expressions[*instr.ExprIndex]

				condition, err := g.evaluateCondition(expr.Expression, params)
				if err != nil {
					return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
				}

				conditionStack[len(conditionStack)-1] = condition
			}

		case intermediate.OpElse:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: ELSE without matching IF", ErrUnsupportedOperation)
			}

			// Flip the condition for the else branch
			conditionStack[len(conditionStack)-1] = !conditionStack[len(conditionStack)-1]

		case intermediate.OpEnd:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: END without matching IF", ErrUnsupportedOperation)
			}

			// Pop the condition stack
			conditionStack = conditionStack[:len(conditionStack)-1]

		case intermediate.OpEmitUnlessBoundary:
			// For now, just emit the value (boundary handling is complex)
			result.WriteString(instr.Value)

		case intermediate.OpBoundary:
			// Skip boundary markers for now

		case intermediate.OpLoopStart, intermediate.OpLoopEnd:
			// Loop operations are not implemented yet
			return "", nil, fmt.Errorf("%w: loop operations not yet implemented", ErrUnsupportedOperation)

		case intermediate.OpIfSystemLimit, intermediate.OpIfSystemOffset,
			intermediate.OpEmitSystemLimit, intermediate.OpEmitSystemOffset, intermediate.OpEmitSystemValue:
			// System operations are not implemented yet
			return "", nil, fmt.Errorf("%w: system operations not yet implemented", ErrUnsupportedOperation)

		default:
			return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedOperation, instr.Op)
		}
	}

	// Check for unmatched conditions
	if len(conditionStack) > 0 {
		return "", nil, fmt.Errorf("%w: unmatched IF statement", ErrUnsupportedOperation)
	}

	return result.String(), sqlParams, nil
}

// evaluateExpression evaluates a CEL expression and returns the result
func (g *SQLGenerator) evaluateExpression(expression string, params map[string]interface{}) (interface{}, error) {
	// Simple parameter lookup for now
	if value, exists := params[expression]; exists {
		return value, nil
	}

	// Try to evaluate as CEL expression
	ast, issues := g.celEnv.Compile(expression)
	if issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", expression, issues.Err())
	}

	program, err := g.celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program for expression '%s': %w", expression, err)
	}

	// Create evaluation context
	evalParams := map[string]interface{}{
		"params": params,
	}

	// Add individual parameters to the context for direct access
	for k, v := range params {
		evalParams[k] = v
	}

	result, _, err := program.Eval(evalParams)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expression, err)
	}

	return result.Value(), nil
}

// evaluateCondition evaluates a condition expression and returns a boolean result
func (g *SQLGenerator) evaluateCondition(expression string, params map[string]interface{}) (bool, error) {
	result, err := g.evaluateExpression(expression, params)
	if err != nil {
		return false, err
	}

	// Convert result to boolean (duplicated logic from previous implementation)
	switch v := result.(type) {
	case bool:
		return v, nil
	case *types.Bool:
		return bool(*v), nil
	case nil:
		return false, nil
	case int, int32, int64:
		return v != 0, nil
	case float32, float64:
		return v != 0.0, nil
	case string:
		return v != "", nil
	default:
		return true, nil // treat non-zero/non-empty as truthy
	}
}
