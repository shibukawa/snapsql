package pygen

import (
	"fmt"

	"github.com/shibukawa/snapsql/intermediate"
)

// pythonExpressionRenderer renders explang expressions into Python code.
type pythonExpressionRenderer struct {
	format *intermediate.IntermediateFormat
	scope  *expressionScope
}

func newPythonExpressionRenderer(format *intermediate.IntermediateFormat, scope *expressionScope) *pythonExpressionRenderer {
	return &pythonExpressionRenderer{format: format, scope: scope}
}

func (r *pythonExpressionRenderer) render(index int) (string, error) {
	if r.format == nil || index < 0 || index >= len(r.format.Expressions) {
		return "", fmt.Errorf("explang expression %d out of range", index)
	}

	expr := r.format.Expressions[index]
	if len(expr.Steps) == 0 {
		return "", fmt.Errorf("explang expression %d has no steps", index)
	}

	root := expr.Steps[0]
	baseName, ok := r.scope.lookup(root.Identifier)
	if !ok {
		baseName = pythonIdentifier(root.Identifier)
	}

	return renderExpressionSteps(baseName, expr.Steps[1:]), nil
}

func renderExpressionSteps(base string, steps []intermediate.Expressions) string {
	if len(steps) == 0 {
		return base
	}

	step := steps[0]
	nextBase := accessExpression(base, step)
	result := renderExpressionSteps(nextBase, steps[1:])

	if step.Safe {
		guard := safeGuardCondition(base, step)
		result = fmt.Sprintf("(None if %s else %s)", guard, result)
	}

	return result
}

func accessExpression(base string, step intermediate.Expressions) string {
	switch step.Kind {
	case intermediate.StepMember:
		attr := pythonIdentifier(step.Property)
		return fmt.Sprintf("%s.%s", base, attr)
	case intermediate.StepIndex:
		return fmt.Sprintf("%s[%d]", base, step.Index)
	default:
		return base
	}
}

func safeGuardCondition(base string, step intermediate.Expressions) string {
	switch step.Kind {
	case intermediate.StepIndex:
		return fmt.Sprintf("(%s is None or len(%s) <= %d)", base, base, step.Index)
	default:
		return fmt.Sprintf("%s is None", base)
	}
}
