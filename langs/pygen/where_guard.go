package pygen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// whereClauseMetaData represents WHERE clause metadata for code generation
type whereClauseMetaData struct {
	Status            string
	RemovalCombos     [][]removalLiteralData
	ExpressionRefs    []int
	DynamicConditions []whereDynamicConditionData
	RawText           string
}

// removalLiteralData represents a single boolean requirement controlling WHERE removal
type removalLiteralData struct {
	ExprIndex int
	When      bool
}

// whereDynamicConditionData describes a conditional construct that may remove the WHERE clause
type whereDynamicConditionData struct {
	ExprIndex        int
	NegatedWhenEmpty bool
	HasElse          bool
	Description      string
}

// convertWhereMeta converts intermediate WHERE clause metadata to template data
func convertWhereMeta(meta *intermediate.WhereClauseMeta) *whereClauseMetaData {
	if meta == nil {
		return nil
	}

	result := &whereClauseMetaData{
		Status:         meta.Status,
		ExpressionRefs: append([]int(nil), meta.ExpressionRefs...),
		RawText:        meta.RawText,
	}

	// Convert removal combos
	if len(meta.RemovalCombos) > 0 {
		result.RemovalCombos = make([][]removalLiteralData, len(meta.RemovalCombos))
		for i, combo := range meta.RemovalCombos {
			result.RemovalCombos[i] = make([]removalLiteralData, len(combo))
			for j, lit := range combo {
				result.RemovalCombos[i][j] = removalLiteralData{
					ExprIndex: lit.ExprIndex,
					When:      lit.When,
				}
			}
		}
	}

	// Convert dynamic conditions
	if len(meta.DynamicConditions) > 0 {
		result.DynamicConditions = make([]whereDynamicConditionData, len(meta.DynamicConditions))
		for i, cond := range meta.DynamicConditions {
			result.DynamicConditions[i] = whereDynamicConditionData{
				ExprIndex:        cond.ExprIndex,
				NegatedWhenEmpty: cond.NegatedWhenEmpty,
				HasElse:          cond.HasElse,
				Description:      cond.Description,
			}
		}
	}

	return result
}

// getMutationKind determines the mutation kind from statement type
func getMutationKind(statementType string) string {
	switch strings.ToLower(statementType) {
	case "update":
		return "MutationUpdate"
	case "delete":
		return "MutationDelete"
	default:
		return ""
	}
}

// isMutationStatement checks if the statement is a mutation (UPDATE/DELETE)
func isMutationStatement(statementType string) bool {
	return getMutationKind(statementType) != ""
}

// generateWhereGuardCode generates Python code for WHERE clause enforcement
func generateWhereGuardCode(funcName string, mutationKind string, whereMeta *whereClauseMetaData, expressions []intermediate.CELExpression) string {
	if mutationKind == "" || whereMeta == nil {
		return ""
	}

	operation := mutationKindToLower(mutationKind)
	if operation == "" {
		return ""
	}

	var guardBuilder strings.Builder

	seen := make(map[string]struct{})
	guardCount := 0

	addGuard := func(condition, hint string) {
		condition = strings.TrimSpace(condition)
		if condition == "" {
			return
		}

		if _, ok := seen[condition]; ok {
			return
		}

		seen[condition] = struct{}{}

		if guardCount == 0 {
			guardBuilder.WriteString("    # WHERE clause guard for " + operation + "\n")
		}

		if hint == "" {
			hint = condition
		}

		guardBuilder.WriteString(fmt.Sprintf("    if %s:\n", condition))
		guardBuilder.WriteString("        raise UnsafeQueryError(\n")
		guardBuilder.WriteString(fmt.Sprintf("            message=%q,\n",
			fmt.Sprintf("%s %s attempted without WHERE clause (%s)",
				strings.ToUpper(operation),
				funcName,
				hint)))
		guardBuilder.WriteString(fmt.Sprintf("            func_name=%q,\n", funcName))
		guardBuilder.WriteString("            query=sql,\n")
		guardBuilder.WriteString(fmt.Sprintf("            mutation_kind=%q\n", operation))
		guardBuilder.WriteString("        )\n\n")

		guardCount++
	}

	if strings.EqualFold(whereMeta.Status, "fullscan") {
		addGuard("True", "template omits WHERE clause")
	}

	for _, cond := range whereMeta.DynamicConditions {
		if cond.HasElse {
			continue
		}

		expr := cond.Description
		if expr == "" {
			expr = expressionString(cond.ExprIndex, expressions)
		}

		if expr == "" {
			continue
		}

		condition := expr
		if cond.NegatedWhenEmpty {
			condition = fmt.Sprintf("not (%s)", expr)
		}

		addGuard(condition, expr)
	}

	for _, combo := range whereMeta.RemovalCombos {
		if len(combo) == 0 {
			addGuard("True", "WHERE clause removed")
			continue
		}

		var (
			parts []string
			hints []string
		)

		for _, literal := range combo {
			expr := expressionString(literal.ExprIndex, expressions)
			if expr == "" {
				expr = fmt.Sprintf("expr[%d]", literal.ExprIndex)
			}

			part := expr
			if !literal.When {
				part = fmt.Sprintf("not (%s)", expr)
			}

			parts = append(parts, part)
			hints = append(hints, fmt.Sprintf("%s=%v", expr, literal.When))
		}

		if len(parts) == 0 {
			continue
		}

		addGuard(strings.Join(parts, " and "), strings.Join(hints, " and "))
	}

	if guardCount == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("ctx = get_snapsql_context()\n")
	builder.WriteString("if not ctx.allow_unsafe_mutations:\n")
	builder.WriteString(guardBuilder.String())

	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func expressionString(exprIndex int, expressions []intermediate.CELExpression) string {
	if expressions == nil {
		return ""
	}

	if exprIndex < 0 || exprIndex >= len(expressions) {
		return ""
	}

	return expressions[exprIndex].Expression
}

// describeDynamicConditions generates a description of dynamic conditions
func describeDynamicConditions(conds []whereDynamicConditionData, filterRemovable bool) string {
	if len(conds) == 0 {
		return ""
	}

	var labels []string

	for _, cond := range conds {
		if filterRemovable && (!cond.NegatedWhenEmpty || cond.HasElse) {
			continue
		}

		label := fmt.Sprintf("expr[%d]", cond.ExprIndex)
		if cond.Description != "" {
			label += " " + cond.Description
		}

		labels = append(labels, label)
	}

	return strings.Join(labels, ", ")
}
