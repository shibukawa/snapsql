package pygen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

// processSQLBuilder processes instructions and generates SQL building code for Python
func processSQLBuilder(format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*sqlBuilderData, error) {
	optimizedInstructions, err := codegenerator.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	needsDynamic := codegenerator.HasDynamicInstructions(optimizedInstructions)
	if !needsDynamic {
		return generateStaticSQL(optimizedInstructions, format, dialect)
	}

	return generateDynamicSQL(optimizedInstructions, format, dialect)
}

func generateStaticSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*sqlBuilderData, error) {
	var (
		sqlBuilder strings.Builder
		arguments  []string
		needsMap   bool
	)

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			sqlBuilder.WriteString(inst.Value)
		case "EMIT_EVAL":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				sqlBuilder.WriteString("?")

				arguments = append(arguments, fmt.Sprintf("_eval_explang_expression(%d, param_map)", *inst.ExprIndex))
				needsMap = true
			}
		case "ADD_PARAM":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				arguments = append(arguments, fmt.Sprintf("_eval_explang_expression(%d, param_map)", *inst.ExprIndex))
				needsMap = true
			}
		case "ADD_SYSTEM_PARAM":
			arguments = append(arguments, inst.SystemField)
		case codegenerator.OpEmitSystemFor:
		}
	}

	staticSQL := convertPlaceholders(sqlBuilder.String(), dialect)

	return &sqlBuilderData{
		IsStatic:      true,
		StaticSQL:     staticSQL,
		Args:          arguments,
		NeedsParamMap: needsMap,
	}, nil
}

func convertPlaceholders(sql string, dialect snapsql.Dialect) string {
	if !strings.Contains(sql, "?") {
		return sql
	}

	paramIndex := 1
	result := strings.Builder{}
	result.Grow(len(sql))

	for i := range len(sql) {
		if sql[i] == '?' {
			result.WriteString(GetPlaceholder(dialect, paramIndex))
			paramIndex++
		} else {
			result.WriteByte(sql[i])
		}
	}

	return result.String()
}

func generateDynamicSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*sqlBuilderData, error) {
	var code strings.Builder

	code.WriteString("# Build SQL dynamically\n")
	code.WriteString("sql_parts = []\n")
	code.WriteString("args = []\n")

	if len(format.Parameters) > 0 {
		code.WriteString("param_map = {\n")

		for _, param := range format.Parameters {
			paramName := pythonIdentifier(param.Name)

			code.WriteString("    ")
			code.WriteString(fmt.Sprintf("'%s': %s,\n", param.Name, paramName))
		}

		code.WriteString("}\n")
	} else {
		code.WriteString("param_map = {}\n")
	}

	type controlFrame struct {
		typ     string
		loopVar string
	}

	var controlStack []controlFrame

	indentLevel := 0
	paramIndex := 1

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			value := inst.Value

			placeholderCount := strings.Count(value, "?")
			if placeholderCount > 0 {
				for range placeholderCount {
					placeholder := GetPlaceholder(dialect, paramIndex)
					value = strings.Replace(value, "?", placeholder, 1)
					paramIndex++
				}
			}

			if indentLevel > 0 {
				code.WriteString(strings.Repeat("    ", indentLevel))
			}

			code.WriteString(fmt.Sprintf("sql_parts.append(%q)\n", value))

		case "EMIT_EVAL":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				placeholder := GetPlaceholder(dialect, paramIndex)

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("sql_parts.append(%q)\n", placeholder))

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("args.append(_eval_explang_expression(%d, param_map))\n", *inst.ExprIndex))

				paramIndex++
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("args.append(_eval_explang_expression(%d, param_map))\n", *inst.ExprIndex))
			}

		case "ADD_SYSTEM_PARAM":
			if indentLevel > 0 {
				code.WriteString(strings.Repeat("    ", indentLevel))
			}

			code.WriteString(fmt.Sprintf("args.append(%s)\n", pythonIdentifier(inst.SystemField)))

		case "IF":
			if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("cond_value = _eval_explang_expression(%d, param_map)\n", *inst.ExprIndex))

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString("if _truthy(cond_value):\n")

				controlStack = append(controlStack, controlFrame{typ: "if"})
				indentLevel++
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				indentLevel--
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				if inst.ExprIndex != nil && hasExplangExpression(format, *inst.ExprIndex) {
					code.WriteString(fmt.Sprintf("cond_value = _eval_explang_expression(%d, param_map)\n", *inst.ExprIndex))

					if indentLevel > 0 {
						code.WriteString(strings.Repeat("    ", indentLevel))
					}

					code.WriteString("elif _truthy(cond_value):\n")
				}

				indentLevel++
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				indentLevel--
				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

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
				loopVar := pythonIdentifier(inst.Variable)
				collectionVar := loopVar + "_collection"

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("%s = _eval_explang_expression(%d, param_map)\n", collectionVar, *inst.CollectionExprIndex))

				if indentLevel > 0 {
					code.WriteString(strings.Repeat("    ", indentLevel))
				}

				code.WriteString(fmt.Sprintf("for %s in _as_iterable(%s):\n", loopVar, collectionVar))
				code.WriteString(strings.Repeat("    ", indentLevel+1))
				code.WriteString(fmt.Sprintf("param_map['%s'] = %s\n", inst.Variable, loopVar))
				controlStack = append(controlStack, controlFrame{typ: "for", loopVar: inst.Variable})
				indentLevel++
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "for" {
				indentLevel--

				frame := controlStack[len(controlStack)-1]
				if frame.loopVar != "" {
					if indentLevel > 0 {
						code.WriteString(strings.Repeat("    ", indentLevel))
					}

					code.WriteString(fmt.Sprintf("param_map.pop('%s', None)\n", frame.loopVar))
				}

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
