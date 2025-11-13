package pygen

import (
	"fmt"
	"strconv"
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
func generateWhereGuardCode(funcName string, mutationKind string, whereMeta *whereClauseMetaData) string {
	if mutationKind == "" || whereMeta == nil {
		return ""
	}

	var code strings.Builder

	// Generate WHERE clause metadata initialization
	code.WriteString("# WHERE clause safety check\n")
	code.WriteString("where_meta = {\n")
	code.WriteString(fmt.Sprintf("    'status': %q,\n", whereMeta.Status))

	if len(whereMeta.RemovalCombos) > 0 {
		code.WriteString("    'removal_combos': [\n")

		for _, combo := range whereMeta.RemovalCombos {
			comboStr := "["

			var comboStrSb109 strings.Builder

			for i, lit := range combo {
				if i > 0 {
					comboStrSb109.WriteString(", ")
				}

				comboStrSb109.WriteString(fmt.Sprintf("{'expr_index': %d, 'when': %v}", lit.ExprIndex, lit.When))
			}

			comboStr += comboStrSb109.String()

			comboStr += "],"
			code.WriteString("        " + comboStr + "\n")
		}

		code.WriteString("    ],\n")
	}

	if len(whereMeta.ExpressionRefs) > 0 {
		refsStr := "["

		var refsStrSb123 strings.Builder

		for i, ref := range whereMeta.ExpressionRefs {
			if i > 0 {
				refsStrSb123.WriteString(", ")
			}

			refsStrSb123.WriteString(strconv.Itoa(ref))
		}

		refsStr += refsStrSb123.String()

		refsStr += "]"
		code.WriteString(fmt.Sprintf("    'expression_refs': %s,\n", refsStr))
	}

	if len(whereMeta.DynamicConditions) > 0 {
		code.WriteString("    'dynamic_conditions': [\n")

		for _, cond := range whereMeta.DynamicConditions {
			condStr := fmt.Sprintf("{'expr_index': %d, 'negated_when_empty': %v, 'has_else': %v",
				cond.ExprIndex, cond.NegatedWhenEmpty, cond.HasElse)
			if cond.Description != "" {
				condStr += fmt.Sprintf(", 'description': %q", cond.Description)
			}

			condStr += "},"
			code.WriteString("        " + condStr + "\n")
		}

		code.WriteString("    ],\n")
	}

	if whereMeta.RawText != "" {
		code.WriteString(fmt.Sprintf("    'raw_text': %q,\n", whereMeta.RawText))
	}

	code.WriteString("}\n\n")

	// Generate enforcement call
	code.WriteString("# Enforce WHERE clause for " + strings.ToLower(mutationKind[8:]) + "\n")
	code.WriteString(fmt.Sprintf("enforce_non_empty_where_clause(ctx, %q, %q, where_meta, sql)\n",
		funcName, strings.ToLower(mutationKind[8:])))

	return strings.TrimSuffix(code.String(), "\n")
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
