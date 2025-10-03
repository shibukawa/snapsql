package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// sqlBuilderData represents SQL building code generation data
type sqlBuilderData struct {
	IsStatic              bool     // true if SQL can be built as a static string
	StaticSQL             string   // static SQL string if IsStatic is true
	BuilderCode           []string // code lines for dynamic SQL building
	HasArguments          bool     // true if the query has parameters
	Arguments             []int    // expression indices for arguments (in order, -1 for system params)
	ParameterNames        []string // parameter names for static SQL (legacy use)
	ArgumentSystemFields  []string // system field names aligned with Arguments slice (empty string for non-system)
	HasSystemArguments    bool     // true if Arguments includes system parameters
	HasNonSystemArguments bool     // true if Arguments includes regular expressions
}

// processSQLBuilderWithDialect processes instructions and generates SQL building code for a specific dialect
func processSQLBuilderWithDialect(format *intermediate.IntermediateFormat, dialect, functionName string) (*sqlBuilderData, error) {
	// Require dialect to be specified
	if dialect == "" {
		return nil, snapsql.ErrDialectMustBeSpecified
	}

	// Use intermediate package's optimization with dialect filtering
	optimizedInstructions, err := intermediate.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Check if we need dynamic building
	needsDynamic := intermediate.HasDynamicInstructions(optimizedInstructions)

	if !needsDynamic {
		// Generate static SQL
		return generateStaticSQLFromOptimized(optimizedInstructions, format)
	}

	// Generate dynamic SQL building code
	return generateDynamicSQLFromOptimized(optimizedInstructions, format, functionName)
}

func ensureSpaceBeforePlaceholders(s string) string {
	if len(s) == 0 {
		return s
	}

	var builder strings.Builder
	builder.Grow(len(s) + len(s)/4)

	for i := range len(s) {
		ch := s[i]
		if (ch == '?' || ch == '$') && i > 0 {
			prev := s[i-1]
			if !isWhitespaceByte(prev) && isWordCharBeforePlaceholder(prev) {
				builder.WriteByte(' ')
			}
		}

		builder.WriteByte(ch)
	}

	return builder.String()
}

func isWhitespaceByte(b byte) bool {
	switch b {
	case ' ', '\n', '\t', '\r':
		return true
	default:
		return false
	}
}

func isWordCharBeforePlaceholder(b byte) bool {
	if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
		return true
	}

	switch b {
	case '_', ')', ']', '"':
		return true
	default:
		return false
	}
}

// generateStaticSQLFromOptimized generates a static SQL string from optimized instructions
func generateStaticSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat) (*sqlBuilderData, error) {
	var (
		sqlParts              []string
		arguments             []int
		argumentSystemFields  []string
		hasSystemArguments    bool
		hasNonSystemArguments bool
		parameterIndex        = 1
	)

	// helper: 直前のパーツの末尾と次のパーツの先頭が共に単語/識別子の場合は間に空白を入れる
	needsSpaceBetween := func(left, right string) bool {
		if left == "" || right == "" {
			return false
		}
		// 左の末尾が英数字/アンダースコア/閉じ括弧
		le := left[len(left)-1]
		// 先頭の空白をスキップ
		k := 0
		for k < len(right) && (right[k] == ' ' || right[k] == '\n' || right[k] == '\t') {
			k++
		}

		if k >= len(right) {
			return false
		}

		rs := right[k]
		isWordTail := (le >= 'A' && le <= 'Z') || (le >= 'a' && le <= 'z') || (le >= '0' && le <= '9') || le == '_' || le == ')'
		isWordHead := (rs >= 'A' && rs <= 'Z') || (rs >= 'a' && rs <= 'z') || rs == '_' || rs == '(' || rs == '$'

		return isWordTail && isWordHead
	}

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			// Convert ? placeholders to PostgreSQL format
			value := inst.Value
			for strings.Contains(value, "?") {
				value = strings.Replace(value, "?", fmt.Sprintf("$%d", parameterIndex), 1)
				parameterIndex++
			}
			// WHERE/RETURNING の前に必ずスペースを入れる（既にスペースがある場合はそのまま）
			value = strings.ReplaceAll(value, "WHERE", " WHERE")
			value = strings.ReplaceAll(value, "RETURNING", " RETURNING")
			value = ensureSpaceBeforePlaceholders(value)
			// 直前のパーツと単語が連結してしまう場合のスペース付与
			if len(sqlParts) > 0 && needsSpaceBetween(sqlParts[len(sqlParts)-1], value) {
				sqlParts[len(sqlParts)-1] = sqlParts[len(sqlParts)-1] + " "
			}

			sqlParts = append(sqlParts, value)
		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				arguments = append(arguments, *inst.ExprIndex)
				argumentSystemFields = append(argumentSystemFields, "")
				hasNonSystemArguments = true
			}
		case "ADD_SYSTEM_PARAM":
			arguments = append(arguments, -1) // Use -1 to indicate system parameter
			argumentSystemFields = append(argumentSystemFields, inst.SystemField)
			hasSystemArguments = true
		}
	}

	staticSQL := strings.Join(sqlParts, "")

	// Convert expression indices and system fields to parameter names for static SQL
	var parameterNames []string

	for idx, exprIndex := range arguments {
		if exprIndex == -1 {
			if idx < len(argumentSystemFields) {
				parameterNames = append(parameterNames, argumentSystemFields[idx])
			}
		} else if exprIndex < len(format.CELExpressions) {
			expr := format.CELExpressions[exprIndex]
			// For simple expressions that are just parameter names, use the parameter directly
			paramName := snakeToCamelLower(expr.Expression)
			parameterNames = append(parameterNames, paramName)
		}
	}

	return &sqlBuilderData{
		IsStatic:              true,
		StaticSQL:             staticSQL,
		HasArguments:          len(arguments) > 0,
		Arguments:             arguments,
		ParameterNames:        parameterNames,
		ArgumentSystemFields:  argumentSystemFields,
		HasSystemArguments:    hasSystemArguments,
		HasNonSystemArguments: hasNonSystemArguments,
	}, nil
}

// generateDynamicSQLFromOptimized generates dynamic SQL building code from optimized instructions
func generateDynamicSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat, functionName string) (*sqlBuilderData, error) {
	var code []string

	hasArguments := false
	evalCounter := 0

	// Track control flow stack
	controlStack := []string{}

	// Add boundary tracking variables
	code = append(code, "var boundaryNeeded bool")

	// Add parameter map for loop variables
	code = append(code, "paramMap := map[string]any{")
	for _, param := range format.Parameters {
		code = append(code, fmt.Sprintf("    %q: %s,", param.Name, snakeToCamelLower(param.Name)))
	}

	code = append(code, "}")

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			// WHERE/RETURNING の前にスペースを強制する
			val := inst.Value
			val = strings.ReplaceAll(val, "WHERE", " WHERE")
			val = strings.ReplaceAll(val, "RETURNING", " RETURNING")
			val = ensureSpaceBeforePlaceholders(val)
			frag := fmt.Sprintf("%q", val)
			// ひとつ前が単語で終わり、今回が単語で始まるなら、直前にスペースを追記する
			code = append(code, fmt.Sprintf(`{ // safe append static with spacing
	_frag := %s
	if builder.Len() > 0 {
		_b := builder.String()
		_last := _b[len(_b)-1]
		// determine if last char is word char
		_endsWord := (_last >= 'A' && _last <= 'Z') || (_last >= 'a' && _last <= 'z') || (_last >= '0' && _last <= '9') || _last == '_' || _last == ')'
		// skip leading spaces in _frag
		_k := 0
		for _k < len(_frag) && (_frag[_k] == ' ' || _frag[_k] == '\n' || _frag[_k] == '\t') { _k++ }
		_startsWord := false
		if _k < len(_frag) { _c := _frag[_k]; _startsWord = (_c >= 'A' && _c <= 'Z') || (_c >= 'a' && _c <= 'z') || _c == '_' || _c == '(' || _c == '$' }
		if _endsWord && _startsWord { builder.WriteByte(' ') }
	}
	builder.WriteString(_frag)
}`, frag))
			code = append(code, "boundaryNeeded = true")

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				resVar := fmt.Sprintf("evalRes%d", evalCounter)
				evalCounter++

				code = append(code, fmt.Sprintf("// Evaluate expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("%s, _, err := %sPrograms[%d].Eval(paramMap)",
					resVar, strings.ToLower(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate expression: %%w\", err)", functionName))
				code = append(code, "}")
				code = append(code, fmt.Sprintf("args = append(args, %s.Value())", resVar))
				hasArguments = true
			}

		case "ADD_SYSTEM_PARAM":
			code = append(code, "// Add system parameter: "+inst.SystemField)
			code = append(code, fmt.Sprintf("args = append(args, systemValues[%q])", inst.SystemField))
			hasArguments = true

		case "EMIT_UNLESS_BOUNDARY":
			code = append(code, "if boundaryNeeded {")
			code = append(code, fmt.Sprintf("    builder.WriteString(%q)", inst.Value))
			code = append(code, "}")
			code = append(code, "boundaryNeeded = true")

		case "BOUNDARY":
			code = append(code, "boundaryNeeded = false")

		case "IF":
			if inst.ExprIndex != nil {
				code = append(code, fmt.Sprintf("// IF condition: expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("condResult, _, err := %sPrograms[%d].Eval(paramMap)",
					strings.ToLower(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate condition: %%w\", err)", functionName))
				code = append(code, "}")
				code = append(code, "if condResult.Value().(bool) {")
				controlStack = append(controlStack, "if")
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1] == "if" {
				code = append(code, "} else if condResult.Value().(bool) {")
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1] == "if" {
				code = append(code, "} else {")
			}

		case "LOOP_START":
			if inst.CollectionExprIndex != nil {
				code = append(code, fmt.Sprintf("// FOR loop: evaluate collection expression %d", *inst.CollectionExprIndex))
				code = append(code, fmt.Sprintf("collectionResult%d, _, err := %sPrograms[%d].Eval(paramMap)",
					*inst.CollectionExprIndex, strings.ToLower(format.FunctionName), *inst.CollectionExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate collection: %%w\", err)", functionName))
				code = append(code, "}")
				code = append(code, fmt.Sprintf("collection%d := collectionResult%d.Value().([]any)",
					*inst.CollectionExprIndex, *inst.CollectionExprIndex))
				code = append(code, fmt.Sprintf("for _, %sLoopVar := range collection%d {",
					inst.Variable, *inst.CollectionExprIndex))
				code = append(code, fmt.Sprintf("    paramMap[%q] = %sLoopVar", inst.Variable, inst.Variable))
				controlStack = append(controlStack, "for")
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1] == "for" {
				// Find the corresponding LOOP_START to get the variable name
				var loopVar string

				for i := len(instructions) - 1; i >= 0; i-- {
					if instructions[i].Op == "LOOP_START" && instructions[i].EnvIndex == inst.EnvIndex {
						loopVar = instructions[i].Variable
						break
					}
				}

				if loopVar != "" {
					code = append(code, fmt.Sprintf("    delete(paramMap, %q)", loopVar))
				}

				code = append(code, "}")
				controlStack = controlStack[:len(controlStack)-1]
			}

		case "END":
			if len(controlStack) > 0 {
				controlType := controlStack[len(controlStack)-1]
				controlStack = controlStack[:len(controlStack)-1]

				switch controlType {
				case "if":
					code = append(code, "}")
				}
			}
		}
	}

	return &sqlBuilderData{
		IsStatic:     false,
		BuilderCode:  code,
		HasArguments: hasArguments,
	}, nil
}
