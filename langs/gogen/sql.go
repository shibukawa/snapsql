package gogen

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

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

func ensureKeywordSpacing(val string) string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return val
	}

	start := strings.IndexFunc(val, func(r rune) bool { return !unicode.IsSpace(r) })
	if start == -1 {
		return val
	}

	upperTrimmed := strings.ToUpper(trimmed)
	for _, kw := range []string{"AND", "OR", "WHERE", "JOIN", "ON"} {
		if !strings.HasPrefix(upperTrimmed, kw) {
			continue
		}

		if start > 0 && !unicode.IsSpace(rune(val[start-1])) {
			val = val[:start] + " " + val[start:]
			start++
		}

		after := start + len(kw)
		if after < len(val) {
			next := rune(val[after])
			if !unicode.IsSpace(next) {
				val = val[:after] + " " + val[after:]
			}
		} else {
			val += " "
		}

		break
	}

	return val
}

func padBoundaryToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}

	switch strings.ToUpper(trimmed) {
	case "AND", "OR":
		return " " + trimmed + " "
	default:
		return value
	}
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
			value = ensureKeywordSpacing(value)
			// 直前のパーツと単語が連結してしまう場合のスペース付与
			if len(sqlParts) > 0 && needsSpaceBetween(sqlParts[len(sqlParts)-1], value) {
				sqlParts[len(sqlParts)-1] = sqlParts[len(sqlParts)-1] + " "
			}

			sqlParts = append(sqlParts, value)
		case "EMIT_UNLESS_BOUNDARY":
			sqlParts = append(sqlParts, padBoundaryToken(inst.Value))
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
	hasSystemArguments := false
	condVarDeclared := false
	evalCounter := 0

	// Track control flow stack
	controlStack := []string{}

	needsBoundaryTracking := slices.ContainsFunc(instructions, func(inst intermediate.OptimizedInstruction) bool {
		return inst.Op == "EMIT_UNLESS_BOUNDARY" || inst.Op == "BOUNDARY"
	})

	// Add boundary tracking variables
	if needsBoundaryTracking {
		code = append(code, "var boundaryNeeded bool")
	}

	temporalParams := make(map[string]bool, len(format.Parameters))
	for _, param := range format.Parameters {
		if normalizeTemporalAlias(param.Type) == "timestamp" {
			temporalParams[param.Name] = true
		}
	}

	// Add parameter map for loop variables
	code = append(code, "paramMap := map[string]any{")
	for _, param := range format.Parameters {
		valueExpr := snakeToCamelLower(param.Name)
		if temporalParams[param.Name] {
			valueExpr = fmt.Sprintf("snapsqlgo.NormalizeNullableTimestamp(%s)", valueExpr)
		}
		code = append(code, fmt.Sprintf("    %q: %s,", param.Name, valueExpr))
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
			val = ensureKeywordSpacing(val)
			frag := fmt.Sprintf("%q", val)
			// 直前に出力がある場合のみワンスペースを追加する
			code = append(code, fmt.Sprintf(`{ // append static fragment
	_frag := %s
	if builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	builder.WriteString(_frag)
}`, frag))
			if needsBoundaryTracking {
				code = append(code, "boundaryNeeded = true")
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				resIndex := evalCounter
				resVar := fmt.Sprintf("evalRes%d", resIndex)
				evalCounter++

				code = append(code, fmt.Sprintf("// Evaluate expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("%s, _, err := %sPrograms[%d].Eval(paramMap)",
					resVar, toLowerCamel(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate expression: %%w\", err)", functionName))
				code = append(code, "}")
				argVar := fmt.Sprintf("argValue%d", resIndex)
				code = append(code, fmt.Sprintf("%s := snapsqlgo.NormalizeNullableTimestamp(%s)", argVar, resVar))
				code = append(code, fmt.Sprintf("args = append(args, %s)", argVar))
				hasArguments = true
			}

		case "ADD_SYSTEM_PARAM":
			code = append(code, "// Add system parameter: "+inst.SystemField)
			code = append(code, fmt.Sprintf("args = append(args, systemValues[%q])", inst.SystemField))
			hasArguments = true
			hasSystemArguments = true

		case "EMIT_UNLESS_BOUNDARY":
			if needsBoundaryTracking {
				padded := padBoundaryToken(inst.Value)

				code = append(code, "if boundaryNeeded {")
				code = append(code, fmt.Sprintf("    builder.WriteString(%q)", padded))
				code = append(code, "}")
				code = append(code, "boundaryNeeded = true")
			}

		case "BOUNDARY":
			if needsBoundaryTracking {
				code = append(code, "boundaryNeeded = false")
			}

		case "IF":
			if inst.ExprIndex != nil {
				assignOp := ":="
				if condVarDeclared {
					assignOp = "="
				} else {
					condVarDeclared = true
				}

				code = append(code, fmt.Sprintf("// IF condition: expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("condResult, _, err %s %sPrograms[%d].Eval(paramMap)",
					assignOp, toLowerCamel(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate condition: %%w\", err)", functionName))
				code = append(code, "}")
				code = append(code, "if snapsqlgo.Truthy(condResult) {")
				controlStack = append(controlStack, "if")
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1] == "if" {
				code = append(code, "} else if snapsqlgo.Truthy(condResult) {")
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1] == "if" {
				code = append(code, "} else {")
			}

		case "LOOP_START":
			if inst.CollectionExprIndex != nil {
				code = append(code, fmt.Sprintf("// FOR loop: evaluate collection expression %d", *inst.CollectionExprIndex))
				code = append(code, fmt.Sprintf("collectionResult%d, _, err := %sPrograms[%d].Eval(paramMap)",
					*inst.CollectionExprIndex, toLowerCamel(format.FunctionName), *inst.CollectionExprIndex))
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
		IsStatic:           false,
		BuilderCode:        code,
		HasArguments:       hasArguments,
		HasSystemArguments: hasSystemArguments,
	}, nil
}
