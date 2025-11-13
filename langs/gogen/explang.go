package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

type explangExpressionData struct {
	Literal string
}

func buildExplangExpressionData(format *intermediate.IntermediateFormat) []explangExpressionData {
	if format == nil || len(format.Expressions) == 0 {
		return nil
	}

	data := make([]explangExpressionData, 0, len(format.Expressions))
	for _, expr := range format.Expressions {
		data = append(data, explangExpressionData{
			Literal: renderExplangExpression(expr),
		})
	}

	return data
}

func renderExplangExpression(expr intermediate.ExplangExpression) string {
	var builder strings.Builder
	builder.WriteString("snapsqlgo.ExplangExpression{\n")

	if expr.ID != "" {
		builder.WriteString(fmt.Sprintf("    ID: %q,\n", expr.ID))
	}

	builder.WriteString("    Expressions: []snapsqlgo.Expression{\n")

	for _, step := range expr.Steps {
		builder.WriteString(fmt.Sprintf("        {Kind: %s, Identifier: %q, Property: %q, Index: %d, Safe: %t, Position: snapsqlgo.ExpPosition{Line:%d, Column:%d, Offset:%d, Length:%d}},\n",
			stepKindConstant(step.Kind),
			step.Identifier,
			step.Property,
			step.Index,
			step.Safe,
			step.Pos.Line,
			step.Pos.Column,
			step.Pos.Offset,
			step.Pos.Length,
		))
	}

	builder.WriteString("    },\n")
	builder.WriteString("}")

	return builder.String()
}

func stepKindConstant(kind intermediate.ExpressionKind) string {
	switch kind {
	case intermediate.StepIdentifier:
		return "snapsqlgo.ExpressionIdentifier"
	case intermediate.StepMember:
		return "snapsqlgo.ExpressionMember"
	case intermediate.StepIndex:
		return "snapsqlgo.ExpressionIndex"
	default:
		return "snapsqlgo.ExpressionIdentifier"
	}
}
