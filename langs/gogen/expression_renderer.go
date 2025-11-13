package gogen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql/intermediate"
)

type expressionMode int

const (
	modeValue expressionMode = iota
	modeIterable
)

type expressionRenderer struct {
	scope       *expressionScope
	exprs       []intermediate.ExplangExpression
	celExprs    []intermediate.CELExpression
	tempCounter int
}

type renderedAccess struct {
	ValueVar string
	Setup    []string
	ValidVar string
}

func newExpressionRenderer(format *intermediate.IntermediateFormat, scope *expressionScope) *expressionRenderer {
	return &expressionRenderer{
		scope:    scope,
		exprs:    format.Expressions,
		celExprs: format.CELExpressions,
	}
}

func (r *expressionRenderer) renderValue(index int) (*renderedAccess, error) {
	return r.render(index, modeValue)
}

func (r *expressionRenderer) renderIterable(index int) (*renderedAccess, error) {
	return r.render(index, modeIterable)
}

func (r *expressionRenderer) render(index int, mode expressionMode) (*renderedAccess, error) {
	if index >= 0 && index < len(r.exprs) && len(r.exprs[index].Steps) > 0 {
		return r.renderFromSteps(index, mode)
	}

	return r.renderFromCEL(index)
}

func (r *expressionRenderer) renderFromSteps(index int, mode expressionMode) (*renderedAccess, error) {
	expr := r.exprs[index]
	if len(expr.Steps) == 0 {
		panic(fmt.Sprintf("explang expression %d has no steps", index))
	}

	rootName := expr.Steps[0].Identifier

	goName, ok := r.scope.lookup(rootName)
	if !ok {
		panic(fmt.Sprintf("identifier %q is not available in the current scope", rootName))
	}

	plan := &renderedAccess{}
	valueVar := goName
	steps := expr.Steps[1:]

	if len(steps) > 0 {
		tmp := r.nextTempVar("tmp")
		plan.Setup = append(plan.Setup, fmt.Sprintf("%s := %s", tmp, valueVar))
		valueVar = tmp
	}

	hasSafe := false

	for _, step := range steps {
		if step.Safe {
			hasSafe = true
			break
		}
	}

	if hasSafe && len(steps) == 0 {
		tmp := r.nextTempVar("tmp")
		plan.Setup = append(plan.Setup, fmt.Sprintf("%s := %s", tmp, valueVar))
		valueVar = tmp
	}

	if hasSafe {
		plan.ValidVar = r.nextTempVar("ok")
		plan.Setup = append(plan.Setup, plan.ValidVar+" := true")
	}

	plan.ValueVar = valueVar

	for _, step := range steps {
		exprStr := renderStepExpression(valueVar, step)

		if step.Safe {
			if plan.ValidVar == "" {
				plan.ValidVar = r.nextTempVar("ok")
				plan.Setup = append(plan.Setup, plan.ValidVar+" := true")
			}

			plan.Setup = append(plan.Setup, fmt.Sprintf("if %s {", plan.ValidVar))

			condition := renderSafeCondition(valueVar, step)
			if condition != "" {
				plan.Setup = append(plan.Setup, fmt.Sprintf("\tif %s {", condition))
				plan.Setup = append(plan.Setup, fmt.Sprintf("\t\t%s = %s", valueVar, exprStr))
				plan.Setup = append(plan.Setup, "\t} else {")
				plan.Setup = append(plan.Setup, fmt.Sprintf("\t\t%s = false", plan.ValidVar))
				plan.Setup = append(plan.Setup, "\t}")
			} else {
				plan.Setup = append(plan.Setup, fmt.Sprintf("\t%s = %s", valueVar, exprStr))
			}

			plan.Setup = append(plan.Setup, "}")

			continue
		}

		if plan.ValidVar != "" {
			plan.Setup = append(plan.Setup, fmt.Sprintf("if %s {", plan.ValidVar))
			plan.Setup = append(plan.Setup, fmt.Sprintf("\t%s = %s", valueVar, exprStr))
			plan.Setup = append(plan.Setup, "}")
		} else {
			plan.Setup = append(plan.Setup, fmt.Sprintf("%s = %s", valueVar, exprStr))
		}
	}

	return plan, nil
}

func (r *expressionRenderer) renderFromCEL(index int) (*renderedAccess, error) {
	if index < 0 || index >= len(r.celExprs) {
		panic(fmt.Sprintf("expression index %d out of range", index))
	}

	cel := r.celExprs[index]

	identifier := strings.TrimSpace(cel.Expression)
	if identifier == "" {
		panic(fmt.Sprintf("expression %d is empty", index))
	}

	if strings.ContainsAny(identifier, ".[]") {
		panic(fmt.Sprintf("expression '%s' requires explang steps; regenerate intermediate", cel.Expression))
	}

	goName, ok := r.scope.lookup(identifier)
	if !ok {
		panic(fmt.Sprintf("identifier %q is not available in the current scope", identifier))
	}

	return &renderedAccess{ValueVar: goName}, nil
}

func (r *expressionRenderer) nextTempVar(prefix string) string {
	name := fmt.Sprintf("%s%d", prefix, r.tempCounter)
	r.tempCounter++

	return name
}

func renderStepExpression(current string, step intermediate.Expressions) string {
	switch step.Kind {
	case intermediate.StepMember:
		return fmt.Sprintf("%s.%s", current, toExportedIdentifier(step.Property))
	case intermediate.StepIndex:
		return fmt.Sprintf("%s[%d]", current, step.Index)
	default:
		panic(fmt.Sprintf("unsupported step kind %d", step.Kind))
	}
}

func renderSafeCondition(current string, step intermediate.Expressions) string {
	switch step.Kind {
	case intermediate.StepMember:
		return current + " != nil"
	case intermediate.StepIndex:
		return fmt.Sprintf("len(%s) > %d", current, step.Index)
	default:
		return ""
	}
}

func toExportedIdentifier(name string) string {
	if name == "" {
		return ""
	}

	if !strings.Contains(name, "_") {
		return capitalizeFirst(name)
	}

	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = capitalizeFirst(part)
	}

	return strings.Join(parts, "")
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)

	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}

	return string(runes)
}
