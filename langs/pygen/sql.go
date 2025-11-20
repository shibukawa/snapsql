package pygen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

func processSQLBuilder(format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*sqlBuilderData, error) {
	optimized, err := codegenerator.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	scope := newExpressionScope(format)
	renderer := newPythonExpressionRenderer(format, scope)

	if !codegenerator.HasDynamicInstructions(optimized) {
		return generateStaticSQL(optimized, format, dialect, renderer)
	}

	return generateDynamicSQL(optimized, format, dialect, renderer, scope)
}

func generateStaticSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect snapsql.Dialect, renderer *pythonExpressionRenderer) (*sqlBuilderData, error) {
	var (
		sqlBuilder strings.Builder
		arguments  []string
	)

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			sqlBuilder.WriteString(inst.Value)
		case "EMIT_EVAL":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				valueExpr, err := renderer.render(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				sqlBuilder.WriteString("?")
				arguments = append(arguments, valueExpr)
			}
		case "ADD_PARAM":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				valueExpr, err := renderer.render(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, valueExpr)
			}
		case "ADD_SYSTEM_PARAM":
			arguments = append(arguments, inst.SystemField)
		case codegenerator.OpEmitSystemFor:
		}
	}

	staticSQL := convertPlaceholders(sqlBuilder.String(), dialect)

	return &sqlBuilderData{
		IsStatic:  true,
		StaticSQL: staticSQL,
		Args:      arguments,
	}, nil
}

func convertPlaceholders(sql string, dialect snapsql.Dialect) string {
	if !strings.Contains(sql, "?") {
		return sql
	}

	paramIndex := 1
	result := strings.Builder{}
	result.Grow(len(sql))

	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			result.WriteString(GetPlaceholder(dialect, paramIndex))
			paramIndex++
		} else {
			result.WriteByte(sql[i])
		}
	}

	return result.String()
}

func generateDynamicSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect snapsql.Dialect, renderer *pythonExpressionRenderer, scope *expressionScope) (*sqlBuilderData, error) {
	var code strings.Builder

	code.WriteString("# Build SQL dynamically\n")
	code.WriteString("sql_parts = []\n")
	code.WriteString("args = []\n")

	type controlFrame struct {
		typ     string
		loopVar string
	}

	controlStack := []controlFrame{}
	indentLevel := 0
	paramIndex := 1

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			value := inst.Value
			placeholderCount := strings.Count(value, "?")
			for range placeholderCount {
				placeholder := GetPlaceholder(dialect, paramIndex)
				value = strings.Replace(value, "?", placeholder, 1)
				paramIndex++
			}

			if indentLevel > 0 {
				code.WriteString(strings.Repeat("    ", indentLevel))
			}
			code.WriteString(fmt.Sprintf("sql_parts.append(%q)\n", value))

		case "EMIT_EVAL":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				exprStr, err := renderer.render(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				placeholder := GetPlaceholder(dialect, paramIndex)
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}
				code.WriteString(fmt.Sprintf("sql_parts.append(%q)\n", placeholder))
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}
				code.WriteString(fmt.Sprintf("args.append(%s)\n", exprStr))
				paramIndex++
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				exprStr, err := renderer.render(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}
				code.WriteString(fmt.Sprintf("args.append(%s)\n", exprStr))
			}

		case "ADD_SYSTEM_PARAM":
			if indentLevel > 0 {
				code.WriteString(strings.Repeat("    ", indentLevel))
			}
			code.WriteString(fmt.Sprintf("args.append(%s)\n", pythonIdentifier(inst.SystemField)))

		case "IF":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				exprStr, err := renderer.render(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				indent := strings.Repeat("    ", indentLevel)
				code.WriteString(fmt.Sprintf("%scond_value = %s\n", indent, exprStr))
				code.WriteString(fmt.Sprintf("%sif cond_value:\n", indent))

				controlStack = append(controlStack, controlFrame{typ: "if"})
				indentLevel++
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				indentLevel--
				if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
					exprStr, err := renderer.render(*inst.ExprIndex)
					if err != nil {
						return nil, err
					}

					indent := strings.Repeat("    ", indentLevel)
					code.WriteString(fmt.Sprintf("%scond_value = %s\n", indent, exprStr))
					code.WriteString(fmt.Sprintf("%selif cond_value:\n", indent))
				}
				indentLevel++
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				indentLevel--
				code.WriteString(strings.Repeat("    ", indentLevel))
				code.WriteString("else:\n")
				indentLevel++
			}

		case "END":
			if len(controlStack) > 0 {
				indentLevel--
				controlStack = controlStack[:len(controlStack)-1]
			}

		case "LOOP_START":
			if inst.CollectionExprIndex != nil && hasExplangExpression(format, *inst.CollectionExprIndex) {
				exprStr, err := renderer.render(*inst.CollectionExprIndex)
				if err != nil {
					return nil, err
				}

				loopVar := pythonIdentifier(inst.Variable)
				collectionVar := loopVar + "_collection"
				iterableVar := loopVar + "_iterable"

				indent := strings.Repeat("    ", indentLevel)
				code.WriteString(fmt.Sprintf("%s%s = %s\n", indent, collectionVar, exprStr))
				code.WriteString(fmt.Sprintf("%sif %s is None:\n", indent, collectionVar))
				code.WriteString(fmt.Sprintf("%s    %s = []\n", indent, iterableVar))
				code.WriteString(fmt.Sprintf("%selif isinstance(%s, (list, tuple)):\n", indent, collectionVar))
				code.WriteString(fmt.Sprintf("%s    %s = list(%s)\n", indent, iterableVar, collectionVar))
				code.WriteString(fmt.Sprintf("%selse:\n", indent))
				code.WriteString(fmt.Sprintf("%s    %s = [%s]\n", indent, iterableVar, collectionVar))
				code.WriteString(fmt.Sprintf("%sfor %s in %s:\n", indent, loopVar, iterableVar))
				scope.pushSingle(inst.Variable, loopVar)
				controlStack = append(controlStack, controlFrame{typ: "for", loopVar: inst.Variable})
				indentLevel++
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "for" {
				scope.pop()
				indentLevel--
				controlStack = controlStack[:len(controlStack)-1]
			}

		case codegenerator.OpEmitSystemFor:
		case "FALLBACK_CONDITION", "BOUNDARY", "EMIT_UNLESS_BOUNDARY":
		}
	}

	code.WriteByte('\n')
	code.WriteString("# Join SQL parts\n")
	code.WriteString("sql = ''.join(sql_parts)\n")

	return &sqlBuilderData{
		IsStatic:    false,
		StaticSQL:   "",
		DynamicCode: strings.TrimSuffix(code.String(), "\n"),
	}, nil
}

func pythonIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	var builder strings.Builder

	for i, r := range s {
		switch {
		case r == '.':
			builder.WriteRune('_')
		case r == '-' || r == ' ':
			builder.WriteRune('_')
		case unicode.IsLetter(r):
			builder.WriteRune(unicode.ToLower(r))
		case r == '_':
			builder.WriteRune('_')
		case unicode.IsDigit(r):
			if i == 0 {
				builder.WriteRune('_')
			}
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	result := builder.String()
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	return result
}

func hasExplangExpression(format *intermediate.IntermediateFormat, index int) bool {
	if format == nil {
		return false
	}

	return index >= 0 && index < len(format.Expressions)
}
