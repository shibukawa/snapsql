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

	// Track control flow stack with loop variable names
	type controlFrame struct {
		typ     string // "if" or "for"
		loopVar string // loop variable name (for "for" frames only)
	}

	controlStack := []controlFrame{}

	needsBoundaryTracking := slices.ContainsFunc(instructions, func(inst intermediate.OptimizedInstruction) bool {
		return inst.Op == "EMIT_UNLESS_BOUNDARY" || inst.Op == "BOUNDARY" || inst.Op == "LOOP_START"
	})

	// Check if boundaryNeeded variable is actually used (i.e., EMIT_UNLESS_BOUNDARY outside loops)
	needsBoundaryNeededVar := false

	if needsBoundaryTracking {
		loopDepth := 0

		for _, inst := range instructions {
			switch inst.Op {
			case "LOOP_START":
				loopDepth++
			case "LOOP_END":
				loopDepth--
			case "EMIT_UNLESS_BOUNDARY", "BOUNDARY":
				if loopDepth == 0 {
					// EMIT_UNLESS_BOUNDARY or BOUNDARY outside a loop - need boundaryNeeded variable
					needsBoundaryNeededVar = true
				}
			}
		}
	}

	// Add boundary tracking variables
	if needsBoundaryNeededVar {
		code = append(code, "var boundaryNeeded bool")
	}

	// Add parameter map for loop variables
	code = append(code, "paramMap := map[string]any{")
	for _, param := range format.Parameters {
		code = append(code, fmt.Sprintf("    %q: %s,", param.Name, snakeToCamelLower(param.Name)))
	}

	code = append(code, "}")

	for i, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			// WHERE/RETURNING の前にスペースを強制する
			val := inst.Value
			val = strings.ReplaceAll(val, "WHERE", " WHERE")
			val = strings.ReplaceAll(val, "RETURNING", " RETURNING")
			val = ensureSpaceBeforePlaceholders(val)
			val = ensureKeywordSpacing(val)

			// Normal static content processing
			inLoop := slices.ContainsFunc(controlStack, func(f controlFrame) bool { return f.typ == "for" })
			{
				// Normal static content processing
				frag := fmt.Sprintf("%q", val)
				// 直前に出力がある場合のみワンスペースを追加する
				code = append(code, fmt.Sprintf(`{ // append static fragment
	_frag := %s
	if builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	builder.WriteString(_frag)
}`, frag))

				if needsBoundaryNeededVar && !inLoop {
					// Only set boundaryNeeded = true outside of loops
					// Check if the next instruction is EMIT_UNLESS_BOUNDARY
					// If so, don't set boundaryNeeded to true (let EMIT_UNLESS_BOUNDARY handle it)
					nextIsEmitUnlessBoundary := false
					if i+1 < len(instructions) && instructions[i+1].Op == "EMIT_UNLESS_BOUNDARY" {
						nextIsEmitUnlessBoundary = true
					}

					if !nextIsEmitUnlessBoundary {
						code = append(code, "boundaryNeeded = true")
					}
				}
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				resVar := fmt.Sprintf("evalRes%d", evalCounter)
				evalCounter++

				code = append(code, fmt.Sprintf("// Evaluate expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("%s, _, err := %sPrograms[%d].Eval(paramMap)",
					resVar, toLowerCamel(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return \"\", nil, fmt.Errorf(\"%s: failed to evaluate expression: %%w\", err)", functionName))
				code = append(code, "}")
				code = append(code, fmt.Sprintf("args = append(args, snapsqlgo.NormalizeNullableTimestamp(%s))", resVar))
				hasArguments = true
			}

		case "ADD_SYSTEM_PARAM":
			code = append(code, "// Add system parameter: "+inst.SystemField)
			code = append(code, fmt.Sprintf("args = append(args, snapsqlgo.NormalizeNullableTimestamp(systemValues[%q]))", inst.SystemField))
			hasArguments = true
			hasSystemArguments = true

		case "EMIT_UNLESS_BOUNDARY":
			if needsBoundaryTracking {
				// Check if we're inside a loop
				var loopVar string

				for j := len(controlStack) - 1; j >= 0; j-- {
					if controlStack[j].typ == "for" {
						loopVar = controlStack[j].loopVar
						break
					}
				}

				if loopVar != "" {
					// Inside a loop: emit delimiter only if NOT the last iteration
					padded := padBoundaryToken(inst.Value)

					code = append(code, fmt.Sprintf("if !%sIsLast {", loopVar))
					code = append(code, fmt.Sprintf("    builder.WriteString(%q)", padded))
					code = append(code, "}")
				} else {
					// Outside a loop (e.g., in IF blocks): emit conditionally based on boundaryNeeded
					// Check the next instruction to determine if we should skip
					shouldSkip := false

					if i+1 < len(instructions) {
						nextInst := instructions[i+1]
						// Skip only if next token starts with ')' or if it's the last token in the clause
						if nextInst.Op == "EMIT_STATIC" && len(nextInst.Value) > 0 {
							firstChar := strings.TrimSpace(nextInst.Value)[0:1]
							if firstChar == ")" {
								shouldSkip = true
							}
						} else if nextInst.Op == "END" || nextInst.Op == "BOUNDARY" {
							// Last token in the clause
							shouldSkip = true
						}
					} else {
						// Last instruction overall
						shouldSkip = true
					}

					if !shouldSkip {
						padded := padBoundaryToken(inst.Value)

						code = append(code, "if boundaryNeeded {")
						code = append(code, fmt.Sprintf("    builder.WriteString(%q)", padded))
						code = append(code, "}")
					}
					// Don't set boundaryNeeded = true here, as this is a boundary delimiter
				}
			}

		case "BOUNDARY":
			if needsBoundaryNeededVar {
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
				controlStack = append(controlStack, controlFrame{typ: "if"})
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				code = append(code, "} else if snapsqlgo.Truthy(condResult) {")
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
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

				// Use AsIterableAnyWithLast to get both the item and isLast flag for boundary tracking
				if needsBoundaryTracking {
					code = append(code, fmt.Sprintf("for %sLoopVar, %sIsLast := range snapsqlgo.AsIterableAnyWithLast(collectionResult%d.Value()) {",
						inst.Variable, inst.Variable, *inst.CollectionExprIndex))
				} else {
					code = append(code, fmt.Sprintf("for %sLoopVar := range snapsqlgo.AsIterableAny(collectionResult%d.Value()) {",
						inst.Variable, *inst.CollectionExprIndex))
				}

				code = append(code, fmt.Sprintf("    paramMap[%q] = %sLoopVar", inst.Variable, inst.Variable))

				controlStack = append(controlStack, controlFrame{typ: "for", loopVar: inst.Variable})
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "for" {
				// Find the corresponding LOOP_START to get the variable name
				var loopVar string

				for j := len(instructions) - 1; j >= 0; j-- {
					if instructions[j].Op == "LOOP_START" && instructions[j].EnvIndex == inst.EnvIndex {
						loopVar = instructions[j].Variable
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
				controlType := controlStack[len(controlStack)-1].typ
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
