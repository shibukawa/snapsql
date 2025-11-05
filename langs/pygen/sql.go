package pygen

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

// processSQLBuilder processes instructions and generates SQL building code for Python
func processSQLBuilder(format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*sqlBuilderData, error) {
	if dialect == "" {
		return nil, errors.New("dialect must be specified")
	}

	// Use intermediate package's optimization with dialect filtering
	optimizedInstructions, err := codegenerator.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Check if we need dynamic building
	needsDynamic := codegenerator.HasDynamicInstructions(optimizedInstructions)

	if !needsDynamic {
		// Generate static SQL
		return generateStaticSQL(optimizedInstructions, format, string(dialect))
	}

	// Generate dynamic SQL building code
	return generateDynamicSQL(optimizedInstructions, format, string(dialect))
}

// generateStaticSQL generates a static SQL string from optimized instructions
func generateStaticSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect string) (*sqlBuilderData, error) {
	var (
		sqlParts  []string
		arguments []string
		celEvals  []celEvaluationData
	)

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			// Just append the value as-is
			// Placeholders are already in the correct format from optimization
			sqlParts = append(sqlParts, inst.Value)

		case "EMIT_EVAL":
			// For static SQL, EMIT_EVAL means we need to evaluate a CEL expression
			// and use it as a parameter
			if inst.ExprIndex != nil && *inst.ExprIndex < len(format.CELExpressions) {
				expr := format.CELExpressions[*inst.ExprIndex]
				// Add a placeholder for the evaluated result
				sqlParts = append(sqlParts, "?")
				// The result will be computed at runtime and added to arguments
				arguments = append(arguments, fmt.Sprintf("_cel_result_%d", *inst.ExprIndex))
				// Track this CEL evaluation
				celEvals = append(celEvals, celEvaluationData{
					Index:      *inst.ExprIndex,
					Expression: expr.Expression,
					EnvIndex:   expr.EnvironmentIndex,
				})
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				// Get the expression to determine the parameter name
				if *inst.ExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.ExprIndex]
					// For simple expressions that are just parameter names, use the parameter directly
					paramName := snakeToCamelLower(expr.Expression)
					arguments = append(arguments, paramName)
				}
			}

		case "ADD_SYSTEM_PARAM":
			// System parameters come from context
			arguments = append(arguments, inst.SystemField)

		case codegenerator.OpEmitSystemFor:
			// Row lock clause - handled at runtime
		}
	}

	staticSQL := strings.Join(sqlParts, "")

	// Now convert placeholders to dialect-specific format
	staticSQL = convertPlaceholders(staticSQL, dialect)

	return &sqlBuilderData{
		IsStatic:       true,
		StaticSQL:      staticSQL,
		Args:           arguments,
		CELEvaluations: celEvals,
	}, nil
}

// convertPlaceholders converts ? placeholders to dialect-specific format
func convertPlaceholders(sql string, dialect string) string {
	if !strings.Contains(sql, "?") {
		return sql
	}

	paramIndex := 1
	result := strings.Builder{}
	result.Grow(len(sql))

	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			placeholder, _ := GetPlaceholder(dialect, paramIndex)
			result.WriteString(placeholder)

			paramIndex++
		} else {
			result.WriteByte(sql[i])
		}
	}

	return result.String()
}

// generateDynamicSQL generates dynamic SQL building code from optimized instructions
func generateDynamicSQL(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, dialect string) (*sqlBuilderData, error) {
	var code []string

	// Track control flow stack
	type controlFrame struct {
		typ     string // "if" or "for"
		loopVar string // loop variable name (for "for" frames only)
	}

	controlStack := []controlFrame{}

	// Check if we need boundary tracking
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
					needsBoundaryNeededVar = true
				}
			}
		}
	}

	// Initialize SQL building
	code = append(code, "# Build SQL dynamically")
	code = append(code, "sql_parts = []")
	code = append(code, "args = []")

	if needsBoundaryNeededVar {
		code = append(code, "boundary_needed = False")
	}

	// Track parameter index for placeholders
	paramIndex := 1

	for i, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			value := inst.Value

			// Convert placeholders to dialect-specific format
			placeholderCount := strings.Count(value, "?")
			if placeholderCount > 0 {
				for j := 0; j < placeholderCount; j++ {
					placeholder, err := GetPlaceholder(dialect, paramIndex)
					if err != nil {
						return nil, err
					}

					value = strings.Replace(value, "?", placeholder, 1)
					paramIndex++
				}
			}

			// Check if we're inside a loop
			inLoop := slices.ContainsFunc(controlStack, func(f controlFrame) bool { return f.typ == "for" })

			code = append(code, fmt.Sprintf("sql_parts.append(%q)", value))

			if needsBoundaryNeededVar && !inLoop {
				// Check if the next instruction is EMIT_UNLESS_BOUNDARY
				nextIsEmitUnlessBoundary := false
				if i+1 < len(instructions) && instructions[i+1].Op == "EMIT_UNLESS_BOUNDARY" {
					nextIsEmitUnlessBoundary = true
				}

				if !nextIsEmitUnlessBoundary {
					code = append(code, "boundary_needed = True")
				}
			}

		case "EMIT_EVAL":
			// Evaluate CEL expression and emit the result
			if inst.ExprIndex != nil {
				code = append(code, fmt.Sprintf("# Evaluate CEL expression %d", *inst.ExprIndex))
				if *inst.ExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.ExprIndex]
					// Generate CEL evaluation code
					evalCode := generateCELEvaluation(*inst.ExprIndex, expr, format)
					code = append(code, evalCode...)

					// Get placeholder for this parameter
					placeholder, err := GetPlaceholder(dialect, paramIndex)
					if err != nil {
						return nil, err
					}

					code = append(code, fmt.Sprintf("sql_parts.append(%q)", placeholder))
					code = append(code, fmt.Sprintf("args.append(_cel_result_%d)", *inst.ExprIndex))
					paramIndex++
				}
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				code = append(code, fmt.Sprintf("# Add parameter from expression %d", *inst.ExprIndex))
				if *inst.ExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.ExprIndex]
					paramName := snakeToCamelLower(expr.Expression)
					code = append(code, fmt.Sprintf("args.append(%s)", paramName))
				}
			}

		case "ADD_SYSTEM_PARAM":
			code = append(code, "# Add system parameter: "+inst.SystemField)
			code = append(code, fmt.Sprintf("args.append(%s)", inst.SystemField))

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
					code = append(code, fmt.Sprintf("if not %s_is_last:", loopVar))
					code = append(code, fmt.Sprintf("    sql_parts.append(%q)", inst.Value))
				} else {
					// Outside a loop: emit conditionally based on boundary_needed
					shouldSkip := false

					if i+1 < len(instructions) {
						nextInst := instructions[i+1]
						if nextInst.Op == "EMIT_STATIC" && len(nextInst.Value) > 0 {
							firstChar := strings.TrimSpace(nextInst.Value)
							if len(firstChar) > 0 && firstChar[0:1] == ")" {
								shouldSkip = true
							}
						} else if nextInst.Op == "END" || nextInst.Op == "BOUNDARY" {
							shouldSkip = true
						}
					} else {
						shouldSkip = true
					}

					if !shouldSkip {
						code = append(code, "if boundary_needed:")
						code = append(code, fmt.Sprintf("    sql_parts.append(%q)", inst.Value))
					}
				}
			}

		case "BOUNDARY":
			if needsBoundaryNeededVar {
				code = append(code, "boundary_needed = False")
			}

		case "IF":
			if inst.ExprIndex != nil {
				code = append(code, fmt.Sprintf("# IF condition: expression %d", *inst.ExprIndex))
				if *inst.ExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.ExprIndex]
					// Check if this is a simple expression or needs CEL evaluation
					if isSimpleCELExpression(expr.Expression) {
						// For simple boolean expressions, evaluate directly
						condExpr := convertCELExpressionToPython(expr.Expression)
						code = append(code, fmt.Sprintf("if %s:", condExpr))
					} else {
						// For complex expressions, use CEL evaluation
						evalCode := generateCELEvaluation(*inst.ExprIndex, expr, format)
						code = append(code, evalCode...)
						code = append(code, fmt.Sprintf("if _cel_result_%d:", *inst.ExprIndex))
					}

					controlStack = append(controlStack, controlFrame{typ: "if"})
				}
			}

		case "ELSEIF":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				if inst.ExprIndex != nil && *inst.ExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.ExprIndex]
					if isSimpleCELExpression(expr.Expression) {
						condExpr := convertCELExpressionToPython(expr.Expression)
						code = append(code, fmt.Sprintf("elif %s:", condExpr))
					} else {
						evalCode := generateCELEvaluation(*inst.ExprIndex, expr, format)
						code = append(code, evalCode...)
						code = append(code, fmt.Sprintf("elif _cel_result_%d:", *inst.ExprIndex))
					}
				}
			}

		case "ELSE":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "if" {
				code = append(code, "else:")
			}

		case "LOOP_START":
			if inst.CollectionExprIndex != nil {
				code = append(code, fmt.Sprintf("# FOR loop: collection expression %d", *inst.CollectionExprIndex))
				if *inst.CollectionExprIndex < len(format.CELExpressions) {
					expr := format.CELExpressions[*inst.CollectionExprIndex]

					var collectionExpr string
					if isSimpleCELExpression(expr.Expression) {
						collectionExpr = convertCELExpressionToPython(expr.Expression)
					} else {
						// For complex expressions, use CEL evaluation
						evalCode := generateCELEvaluation(*inst.CollectionExprIndex, expr, format)
						code = append(code, evalCode...)
						collectionExpr = fmt.Sprintf("_cel_result_%d", *inst.CollectionExprIndex)
					}

					if needsBoundaryTracking {
						// Use enumerate to track if it's the last item
						code = append(code, fmt.Sprintf("_collection = list(%s)", collectionExpr))
						code = append(code, fmt.Sprintf("for %s_idx, %s in enumerate(_collection):", inst.Variable, inst.Variable))
						code = append(code, fmt.Sprintf("    %s_is_last = (%s_idx == len(_collection) - 1)", inst.Variable, inst.Variable))
					} else {
						code = append(code, fmt.Sprintf("for %s in %s:", inst.Variable, collectionExpr))
					}

					controlStack = append(controlStack, controlFrame{typ: "for", loopVar: inst.Variable})
				}
			}

		case "LOOP_END":
			if len(controlStack) > 0 && controlStack[len(controlStack)-1].typ == "for" {
				// No explicit end needed in Python (indentation-based)
				controlStack = controlStack[:len(controlStack)-1]
			}

		case "END":
			if len(controlStack) > 0 {
				controlType := controlStack[len(controlStack)-1].typ
				controlStack = controlStack[:len(controlStack)-1]

				if controlType == "if" {
					// No explicit end needed in Python (indentation-based)
				}
			}

		case codegenerator.OpEmitSystemFor:
			// Row lock clause - handled at runtime
		}
	}

	// Join SQL parts
	code = append(code, "")
	code = append(code, "# Join SQL parts")
	code = append(code, "sql = ' '.join(sql_parts)")

	return &sqlBuilderData{
		IsStatic:    false,
		DynamicCode: strings.Join(code, "\n"),
	}, nil
}

// convertCELExpressionToPython converts a simple CEL expression to Python
// This is a simplified converter for common cases
func convertCELExpressionToPython(celExpr string) string {
	// Handle simple cases
	expr := strings.TrimSpace(celExpr)

	// Convert null checks first (before other operators)
	expr = strings.ReplaceAll(expr, " == null", " is None")
	expr = strings.ReplaceAll(expr, " != null", " is not None")
	expr = strings.ReplaceAll(expr, "== null", "is None")
	expr = strings.ReplaceAll(expr, "!= null", "is not None")

	// Convert CEL operators to Python
	expr = strings.ReplaceAll(expr, "&&", " and ")
	expr = strings.ReplaceAll(expr, "||", " or ")

	// Handle negation carefully - only replace ! at the start or after whitespace
	if strings.HasPrefix(expr, "!") {
		expr = "not " + expr[1:]
	}

	expr = strings.ReplaceAll(expr, " !", " not ")

	// Convert parameter names from snake_case to camelCase
	// This is a simple heuristic - for complex expressions, we'd need proper parsing
	words := strings.Fields(expr)
	for i, word := range words {
		// Skip Python keywords and operators
		if word == "and" || word == "or" || word == "not" || word == "is" || word == "None" {
			continue
		}
		// Convert identifiers
		if strings.Contains(word, "_") {
			words[i] = snakeToCamelLower(word)
		}
	}

	expr = strings.Join(words, " ")

	return expr
}

// snakeToCamelLower converts snake_case to camelCase (lower first letter)
func snakeToCamelLower(s string) string {
	// If already in camelCase or simple identifier, return as-is
	if !strings.Contains(s, "_") {
		return s
	}

	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}

	// First part stays lowercase
	result := strings.ToLower(parts[0])

	// Capitalize first letter of remaining parts
	var resultSb440 strings.Builder
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			resultSb440.WriteString(strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:]))
		}
	}
	result += resultSb440.String()

	return result
}

// generateCELEvaluation generates Python code to evaluate a CEL expression
func generateCELEvaluation(exprIndex int, expr intermediate.CELExpression, format *intermediate.IntermediateFormat) []string {
	var code []string

	// Build the context dictionary for CEL evaluation
	code = append(code, "# Evaluate CEL expression: "+expr.Expression)
	code = append(code, "_cel_context = {")

	// Add parameters to context
	for _, param := range format.Parameters {
		paramName := snakeToCamelLower(param.Name)
		code = append(code, fmt.Sprintf("    %q: %s,", param.Name, paramName))
	}

	// Add additional variables from the CEL environment
	if expr.EnvironmentIndex < len(format.CELEnvironments) {
		env := format.CELEnvironments[expr.EnvironmentIndex]
		for _, v := range env.AdditionalVariables {
			varName := snakeToCamelLower(v.Name)
			code = append(code, fmt.Sprintf("    %q: %s,", v.Name, varName))
		}
	}

	code = append(code, "}")

	// Evaluate the CEL program
	code = append(code, fmt.Sprintf("_cel_result_%d = _cel_program_%d.evaluate(_cel_context)", exprIndex, exprIndex))

	return code
}

// isSimpleCELExpression checks if a CEL expression is simple enough to convert directly to Python
// Simple expressions are those that only use basic operators and don't require CEL evaluation
func isSimpleCELExpression(expr string) bool {
	// Simple expressions typically:
	// - Are just variable names
	// - Use basic comparison operators (==, !=, <, >, <=, >=)
	// - Use basic logical operators (&&, ||, !)
	// - Don't use CEL-specific functions

	// Check for CEL-specific functions that require evaluation
	celFunctions := []string{
		".size()", ".length()", ".contains(", ".startsWith(", ".endsWith(",
		".matches(", ".split(", ".join(", ".replace(",
		"has(", "type(", "int(", "string(", "double(",
	}

	for _, fn := range celFunctions {
		if strings.Contains(expr, fn) {
			return false
		}
	}

	// If it contains method calls or complex expressions, use CEL evaluation
	if strings.Contains(expr, ".") && !strings.Contains(expr, " . ") {
		// Check if it's a method call (not just a decimal number)
		if strings.Contains(expr, "(") {
			return false
		}
	}

	// Otherwise, it's simple enough to convert directly
	return true
}
