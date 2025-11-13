package gogen

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// sqlBuilderData represents SQL building code generation data
type sqlBuilderData struct {
	IsStatic             bool     // true if SQL can be built as a static string
	StaticSQL            string   // static SQL string if IsStatic is true
	BuilderCode          []string // code lines for dynamic SQL building
	HasArguments         bool     // true if the query has parameters
	ArgumentExprs        []argumentExpr
	ArgumentSystemFields []string // system field names aligned with argument blocks (empty string for non-system)
	HasSystemArguments   bool     // true if Arguments includes system parameters
	NeedsRowLockClause   bool     // true if SQL expects a runtime row-lock clause appended
	HasFallbackGuard     bool     // true if FALLBACK_CONDITION instructions are present
	FallbackVarName      string   // name of the boolean flag tracking fallback usage
}

type argumentExpr struct {
	Lines []string
}

type fallbackExprPlan struct {
	Lines   []string
	BoolVar string
}

var _ = snapsqlgo.Truthy

func indentLines(lines []string, level int) []string {
	if level <= 0 {
		return append([]string(nil), lines...)
	}

	prefix := strings.Repeat("\t", level)

	result := make([]string, len(lines))
	for i, line := range lines {
		if line == "" {
			result[i] = ""
			continue
		}

		result[i] = prefix + line
	}

	return result
}

func buildArgumentLines(plan *renderedAccess) []string {
	lines := make([]string, 0, len(plan.Setup)+4)
	lines = append(lines, plan.Setup...)

	appendLine := fmt.Sprintf("args = append(args, snapsqlgo.NormalizeNullableTimestamp(%s))", plan.ValueVar)
	if plan.ValidVar != "" {
		lines = append(lines, fmt.Sprintf("if %s {", plan.ValidVar))
		lines = append(lines, "\t"+appendLine)
		lines = append(lines, "} else {")
		lines = append(lines, "\targs = append(args, nil)")
		lines = append(lines, "}")
	} else {
		lines = append(lines, appendLine)
	}

	return lines
}

func buildConditionLines(plan *renderedAccess, condVar string) []string {
	lines := make([]string, 0, len(plan.Setup)+3)
	lines = append(lines, plan.Setup...)

	lines = append(lines, fmt.Sprintf("%s := %s", condVar, plan.ValueVar))
	if plan.ValidVar != "" {
		lines = append(lines, fmt.Sprintf("if !%s {", plan.ValidVar))
		lines = append(lines, fmt.Sprintf("\t%s = nil", condVar))
		lines = append(lines, "}")
	}

	return lines
}

// processSQLBuilderWithDialect processes instructions and generates SQL building code for a specific dialect
func processSQLBuilderWithDialect(format *intermediate.IntermediateFormat, dialect, functionName string) (*sqlBuilderData, error) {
	// Require dialect to be specified
	if dialect == "" {
		return nil, snapsql.ErrDialectMustBeSpecified
	}

	// Use intermediate package's optimization with dialect filtering
	optimizedInstructions, err := codegenerator.OptimizeInstructions(format.Instructions, snapsql.Dialect(dialect))
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Check if we need dynamic building
	needsDynamic := codegenerator.HasDynamicInstructions(optimizedInstructions)

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

	// Check keywords in order: longer keywords first to avoid partial matches
	// e.g., check "ORDER" before "OR" to prevent "ORDER BY" from matching "OR"
	for _, kw := range []string{"WHERE", "JOIN", "ORDER", "GROUP", "AND", "OR", "ON"} {
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
func generateStaticSQLFromOptimized(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat) (*sqlBuilderData, error) {
	var (
		sqlParts             []string
		argumentExprs        []argumentExpr
		argumentSystemFields []string
		hasSystemArguments   bool
		parameterIndex       = 1
	)

	scope := newExpressionScope(format.Parameters)
	renderer := newExpressionRenderer(format, scope)

	needsRowLockClause := slices.ContainsFunc(instructions, func(inst codegenerator.OptimizedInstruction) bool {
		return inst.Op == codegenerator.OpEmitSystemFor
	})

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
				plan, err := renderer.renderValue(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				argumentExprs = append(argumentExprs, argumentExpr{Lines: indentLines(buildArgumentLines(plan), 1)})
				argumentSystemFields = append(argumentSystemFields, "")
			}
		case "ADD_SYSTEM_PARAM":
			line := fmt.Sprintf("args = append(args, snapsqlgo.NormalizeNullableTimestamp(systemValues[%q]))", inst.SystemField)
			argumentExprs = append(argumentExprs, argumentExpr{Lines: []string{"\t" + line}})
			argumentSystemFields = append(argumentSystemFields, inst.SystemField)
			hasSystemArguments = true
		case codegenerator.OpEmitSystemFor:
			// handled at runtime via rowLockClause
		}
	}

	staticSQL := strings.Join(sqlParts, "")

	return &sqlBuilderData{
		IsStatic:             true,
		StaticSQL:            staticSQL,
		HasArguments:         len(argumentExprs) > 0,
		ArgumentExprs:        argumentExprs,
		ArgumentSystemFields: argumentSystemFields,
		HasSystemArguments:   hasSystemArguments,
		NeedsRowLockClause:   needsRowLockClause,
	}, nil
}

// generateDynamicSQLFromOptimized generates dynamic SQL building code from optimized instructions
func generateDynamicSQLFromOptimized(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, functionName string) (*sqlBuilderData, error) {
	var code []string

	scope := newExpressionScope(format.Parameters)
	renderer := newExpressionRenderer(format, scope)

	hasArguments := false
	hasSystemArguments := false
	condCounter := 0
	loopCounter := 0
	fallbackCounter := 0
	needsRowLockClause := slices.ContainsFunc(instructions, func(inst codegenerator.OptimizedInstruction) bool {
		return inst.Op == codegenerator.OpEmitSystemFor
	})

	hasFallbackGuard := slices.ContainsFunc(instructions, func(inst codegenerator.OptimizedInstruction) bool {
		return inst.Op == codegenerator.OpFallbackCondition
	})

	fallbackGuardVar := ""
	if hasFallbackGuard {
		fallbackGuardVar = "fallbackGuardTriggered"
	}

	// Track control flow stack with loop / condition metadata
	type controlFrame struct {
		typ        string // "if" or "for"
		loopVar    string // logical loop variable name
		loopIsLast string // Go variable name for isLast flag
		condVar    string // Go variable name holding condition result
	}

	controlStack := []controlFrame{}

	needsBoundaryTracking := slices.ContainsFunc(instructions, func(inst codegenerator.OptimizedInstruction) bool {
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
				plan, err := renderer.renderValue(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				code = append(code, fmt.Sprintf("// Evaluate expression %d", *inst.ExprIndex))
				code = append(code, buildArgumentLines(plan)...)
				hasArguments = true
			}

		case "ADD_SYSTEM_PARAM":
			code = append(code, "// Add system parameter: "+inst.SystemField)
			code = append(code, fmt.Sprintf("args = append(args, snapsqlgo.NormalizeNullableTimestamp(systemValues[%q]))", inst.SystemField))
			hasArguments = true
			hasSystemArguments = true

		case codegenerator.OpEmitSystemFor:
			// Row-lock clause appended after query construction

		case "EMIT_UNLESS_BOUNDARY":
			if needsBoundaryTracking {
				// Check if we're inside a loop
				var loopIsLast string

				for j := len(controlStack) - 1; j >= 0; j-- {
					if controlStack[j].typ == "for" {
						loopIsLast = controlStack[j].loopIsLast
						break
					}
				}

				if loopIsLast != "" {
					// Inside a loop: emit delimiter only if NOT the last iteration
					padded := padBoundaryToken(inst.Value)

					code = append(code, fmt.Sprintf("if !%s {", loopIsLast))
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
				plan, err := renderer.renderValue(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				condVar := fmt.Sprintf("condValue%d", condCounter)
				condCounter++

				code = append(code, fmt.Sprintf("// IF condition: expression %d", *inst.ExprIndex))
				code = append(code, buildConditionLines(plan, condVar)...)
				code = append(code, fmt.Sprintf("if snapsqlgo.Truthy(%s) {", condVar))
				controlStack = append(controlStack, controlFrame{typ: "if", condVar: condVar})
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" && inst.ExprIndex != nil {
				plan, err := renderer.renderValue(*inst.ExprIndex)
				if err != nil {
					return nil, err
				}

				condVar := fmt.Sprintf("condValue%d", condCounter)
				condCounter++

				code = append(code, buildConditionLines(plan, condVar)...)
				code = append(code, fmt.Sprintf("} else if snapsqlgo.Truthy(%s) {", condVar))
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				code = append(code, "} else {")
			}

		case "LOOP_START":
			if inst.CollectionExprIndex != nil {
				plan, err := renderer.renderIterable(*inst.CollectionExprIndex)
				if err != nil {
					return nil, err
				}

				collectionVar := fmt.Sprintf("collectionValue%d", loopCounter)
				loopCounter++

				code = append(code, fmt.Sprintf("// FOR loop: evaluate collection expression %d", *inst.CollectionExprIndex))
				code = append(code, fmt.Sprintf("var %s any", collectionVar))
				code = append(code, plan.Setup...)

				code = append(code, fmt.Sprintf("%s = %s", collectionVar, plan.ValueVar))
				if plan.ValidVar != "" {
					code = append(code, fmt.Sprintf("if !%s {", plan.ValidVar))
					code = append(code, fmt.Sprintf("    %s = nil", collectionVar))
					code = append(code, "}")
				}

				loopItemVar := scope.pushLoopVar(inst.Variable)

				loopIsLastVar := ""
				if needsBoundaryTracking {
					loopIsLastVar = loopItemVar + "IsLast"
					code = append(code, fmt.Sprintf("for %s, %s := range snapsqlgo.AsIterableAnyWithLast(%s) {",
						loopItemVar, loopIsLastVar, collectionVar))
				} else {
					code = append(code, fmt.Sprintf("for %s := range snapsqlgo.AsIterableAny(%s) {",
						loopItemVar, collectionVar))
				}

				controlStack = append(controlStack, controlFrame{typ: "for", loopVar: loopItemVar, loopIsLast: loopIsLastVar})
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "for" {
				scope.pop()

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

		case codegenerator.OpFallbackCondition:
			fallbackCounter++
			blockLines := []string{"{"}

			fallbackVar := fmt.Sprintf("fallbackActive%d", fallbackCounter)

			exprPlans := make(map[int]*fallbackExprPlan)

			if len(inst.FallbackCombos) == 0 {
				blockLines = append(blockLines, fallbackVar+" := true")
			} else {
				blockLines = append(blockLines, fallbackVar+" := false")

				for _, combo := range inst.FallbackCombos {
					for _, literal := range combo {
						_, ok := exprPlans[literal.ExprIndex]
						if ok {
							continue
						}

						accessPlan, err := renderer.renderValue(literal.ExprIndex)
						if err != nil {
							return nil, err
						}

						lines := make([]string, 0, len(accessPlan.Setup)+3)

						lines = append(lines, accessPlan.Setup...)
						if accessPlan.ValidVar != "" {
							lines = append(lines, fmt.Sprintf("if !%s { %s = nil }", accessPlan.ValidVar, accessPlan.ValueVar))
						}

						boolVar := fmt.Sprintf("fallbackCond%d_%d", fallbackCounter, literal.ExprIndex)
						lines = append(lines, fmt.Sprintf("%s := snapsqlgo.Truthy(%s)", boolVar, accessPlan.ValueVar))
						plan := &fallbackExprPlan{Lines: lines, BoolVar: boolVar}
						exprPlans[literal.ExprIndex] = plan
						blockLines = append(blockLines, plan.Lines...)
					}
				}

				for comboIndex, combo := range inst.FallbackCombos {
					comboVar := fmt.Sprintf("comboActive%d_%d", fallbackCounter, comboIndex)
					blockLines = append(blockLines, comboVar+" := true")

					for _, literal := range combo {
						plan := exprPlans[literal.ExprIndex]
						boolVar := plan.BoolVar

						condition := boolVar
						if !literal.When {
							condition = "!" + condition
						}

						blockLines = append(blockLines, fmt.Sprintf("if !(%s) { %s = false }", condition, comboVar))
					}

					blockLines = append(blockLines, fmt.Sprintf("if %s { %s = true }", comboVar, fallbackVar))
				}
			}

			fallbackValue := inst.Value
			if fallbackValue == "" {
				fallbackValue = "1 = 1"
			}

			blockLines = append(blockLines, fmt.Sprintf("if %s {", fallbackVar))
			if strings.HasPrefix(fallbackValue, "WHERE") {
				blockLines = append(blockLines, fmt.Sprintf("    builder.WriteString(%q)", fallbackValue))
			} else {
				blockLines = append(blockLines, "    if builder.Len() > 0 { builder.WriteString(\" \") }")
				blockLines = append(blockLines, fmt.Sprintf("    builder.WriteString(%q)", fallbackValue))
			}

			if fallbackGuardVar != "" {
				blockLines = append(blockLines, fmt.Sprintf("    if %s { %s = true }", fallbackVar, fallbackGuardVar))
			}

			blockLines = append(blockLines, "}")
			blockLines = append(blockLines, "}")

			code = append(code, blockLines...)
		}
	}

	return &sqlBuilderData{
		IsStatic:           false,
		BuilderCode:        code,
		HasArguments:       hasArguments,
		HasSystemArguments: hasSystemArguments,
		NeedsRowLockClause: needsRowLockClause,
		HasFallbackGuard:   hasFallbackGuard,
		FallbackVarName:    fallbackGuardVar,
	}, nil
}
