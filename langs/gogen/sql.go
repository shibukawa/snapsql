package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// sqlBuilderData represents SQL building code generation data
type sqlBuilderData struct {
	IsStatic       bool     // true if SQL can be built as a static string
	StaticSQL      string   // static SQL string if IsStatic is true
	BuilderCode    []string // code lines for dynamic SQL building
	HasArguments   bool     // true if the query has parameters
	Arguments      []int    // expression indices for arguments (in order)
	ParameterNames []string // parameter names for static SQL (in order)
}

// processSQLBuilderWithDialect processes instructions and generates SQL building code for a specific dialect
func processSQLBuilderWithDialect(format *intermediate.IntermediateFormat, dialect string) (*sqlBuilderData, error) {
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
	return generateDynamicSQLFromOptimized(optimizedInstructions, format)
}

// generateStaticSQLFromOptimized generates a static SQL string from optimized instructions
func generateStaticSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat) (*sqlBuilderData, error) {
	var (
		sqlParts  []string
		arguments []int
	)

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			sqlParts = append(sqlParts, inst.Value)
		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				arguments = append(arguments, *inst.ExprIndex)
			}
		}
	}

	staticSQL := strings.Join(sqlParts, "")

	// Convert expression indices to parameter names for static SQL
	var parameterNames []string

	for _, exprIndex := range arguments {
		if exprIndex < len(format.CELExpressions) {
			expr := format.CELExpressions[exprIndex]
			// For simple expressions that are just parameter names, use the parameter directly
			paramName := snakeToCamelLower(expr.Expression)
			parameterNames = append(parameterNames, paramName)
		}
	}

	return &sqlBuilderData{
		IsStatic:       true,
		StaticSQL:      staticSQL,
		HasArguments:   len(arguments) > 0,
		Arguments:      arguments,
		ParameterNames: parameterNames,
	}, nil
}

// generateDynamicSQLFromOptimized generates dynamic SQL building code from optimized instructions
func generateDynamicSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat) (*sqlBuilderData, error) {
	var code []string

	hasArguments := false

	// Track control flow stack
	controlStack := []string{}

	// Add boundary tracking variables
	code = append(code, "var boundaryNeeded bool")

	// Add parameter map for loop variables
	code = append(code, "paramMap := map[string]interface{}{")
	for _, param := range format.Parameters {
		code = append(code, fmt.Sprintf("    %q: %s,", param.Name, snakeToCamelLower(param.Name)))
	}

	code = append(code, "}")

	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			code = append(code, fmt.Sprintf("builder.WriteString(%q)", inst.Value))
			code = append(code, "boundaryNeeded = true")

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				code = append(code, fmt.Sprintf("// Evaluate expression %d", *inst.ExprIndex))
				code = append(code, fmt.Sprintf("result, err := %sPrograms[%d].Eval(paramMap)",
					strings.ToLower(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, "    return result, fmt.Errorf(\"failed to evaluate expression: %w\", err)")
				code = append(code, "}")
				code = append(code, "args = append(args, result.Value())")
				hasArguments = true
			}

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
				code = append(code, fmt.Sprintf("condResult, err := %sPrograms[%d].Eval(paramMap)",
					strings.ToLower(format.FunctionName), *inst.ExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, "    return result, fmt.Errorf(\"failed to evaluate condition: %w\", err)")
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
				code = append(code, fmt.Sprintf("collectionResult%d, err := %sPrograms[%d].Eval(paramMap)",
					*inst.CollectionExprIndex, strings.ToLower(format.FunctionName), *inst.CollectionExprIndex))
				code = append(code, "if err != nil {")
				code = append(code, "    return result, fmt.Errorf(\"failed to evaluate collection: %w\", err)")
				code = append(code, "}")
				code = append(code, fmt.Sprintf("collection%d := collectionResult%d.Value().([]interface{})",
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
