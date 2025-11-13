package pygen

import (
    "github.com/shibukawa/snapsql/intermediate"
)

type explangExpressionData struct {
    ID    string
    Steps []explangStepData
}

type explangStepData struct {
    Kind       string
    Identifier string
    Property   string
    Index      int
    Safe       bool
}

func buildExplangExpressionData(format *intermediate.IntermediateFormat) []explangExpressionData {
    if format == nil || len(format.Expressions) == 0 {
        return nil
    }

    data := make([]explangExpressionData, 0, len(format.Expressions))
    for _, expr := range format.Expressions {
        stepList := make([]explangStepData, 0, len(expr.Steps))
        for _, step := range expr.Steps {
            stepList = append(stepList, explangStepData{
                Kind:       stepKindName(step.Kind),
                Identifier: step.Identifier,
                Property:   step.Property,
                Index:      step.Index,
                Safe:       step.Safe,
            })
        }

        data = append(data, explangExpressionData{
            ID:    expr.ID,
            Steps: stepList,
        })
    }

    return data
}

func stepKindName(kind intermediate.ExpressionKind) string {
    switch kind {
    case intermediate.StepIdentifier:
        return "identifier"
    case intermediate.StepMember:
        return "member"
    case intermediate.StepIndex:
        return "index"
    default:
        return "identifier"
    }
}
