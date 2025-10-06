package query

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/uuid"
	"github.com/shibukawa/snapsql/intermediate"
)

// Error definitions for SQL generation
var (
	ErrInvalidExpressionIndex = errors.New("invalid expression index")
	ErrParameterNotFound      = errors.New("parameter not found")
	ErrExpressionEvaluation   = errors.New("expression evaluation failed")
	ErrUnsupportedOperation   = errors.New("unsupported operation")
)

var randomSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// SQLGenerator generates SQL from intermediate format instructions
type SQLGenerator struct {
	instructions []intermediate.Instruction
	expressions  []intermediate.CELExpression
	systemFields map[string]intermediate.SystemFieldInfo
	implicitMap  map[string]intermediate.ImplicitParameter
	generated    map[string]any
	dialect      string
	celEnv       *cel.Env
}

// NewSQLGenerator creates a new SQL generator
func NewSQLGenerator(format *intermediate.IntermediateFormat, dialect string) *SQLGenerator {
	var (
		instructions []intermediate.Instruction
		expressions  []intermediate.CELExpression
		systemFields map[string]intermediate.SystemFieldInfo
		implicitMap  map[string]intermediate.ImplicitParameter
	)

	if format != nil {
		instructions = format.Instructions
		expressions = format.CELExpressions

		if len(format.SystemFields) > 0 {
			systemFields = make(map[string]intermediate.SystemFieldInfo, len(format.SystemFields))
			for _, field := range format.SystemFields {
				systemFields[strings.ToLower(field.Name)] = field
			}
		}

		if len(format.ImplicitParameters) > 0 {
			implicitMap = make(map[string]intermediate.ImplicitParameter, len(format.ImplicitParameters))
			for _, ip := range format.ImplicitParameters {
				implicitMap[strings.ToLower(ip.Name)] = ip
			}
		}
	}

	if systemFields == nil {
		systemFields = make(map[string]intermediate.SystemFieldInfo)
	}

	if implicitMap == nil {
		implicitMap = make(map[string]intermediate.ImplicitParameter)
	}

	// Create CEL environment
	env, _ := cel.NewEnv(
		cel.Variable("params", cel.MapType(cel.StringType, cel.AnyType)),
	)

	return &SQLGenerator{
		instructions: instructions,
		expressions:  expressions,
		systemFields: systemFields,
		implicitMap:  implicitMap,
		generated:    make(map[string]any),
		dialect:      dialect,
		celEnv:       env,
	}
}

// Generate generates SQL and parameters from the instructions
func (g *SQLGenerator) Generate(params map[string]interface{}) (string, []interface{}, error) {
	if params == nil {
		params = make(map[string]interface{})
	}

	var (
		result   strings.Builder
		lastChar byte
	)

	hasLastChar := false
	isWhitespace := func(b byte) bool {
		switch b {
		case ' ', '\n', '\t', '\r':
			return true
		default:
			return false
		}
	}
	appendSQL := func(s string) {
		if len(s) == 0 {
			return
		}

		result.WriteString(s)
		lastChar = s[len(s)-1]
		hasLastChar = true
	}
	appendSpaceIfNeeded := func() {
		if hasLastChar && isWhitespace(lastChar) {
			return
		}

		result.WriteByte(' ')

		lastChar = ' '
		hasLastChar = true
	}

	sqlParams := make([]interface{}, 0) // Initialize as empty slice instead of nil

	var conditionStack []bool // Stack to track if/else conditions

	for i, instr := range g.instructions {
		// Skip instructions if we're in a false condition block
		if len(conditionStack) > 0 && !conditionStack[len(conditionStack)-1] {
			if instr.Op == intermediate.OpEnd || instr.Op == intermediate.OpElse || instr.Op == intermediate.OpElseIf {
				// These operations need to be processed even in false blocks
			} else {
				continue
			}
		}

		switch instr.Op {
		case intermediate.OpEmitStatic:
			value := instr.Value
			if len(value) > 0 && !isWhitespace(value[0]) {
				leading := strings.TrimLeft(value, " \t\r\n")
				if len(leading) > 0 {
					upper := strings.ToUpper(leading)
					if strings.HasPrefix(upper, "AND") || strings.HasPrefix(upper, "OR") || strings.HasPrefix(upper, "WHERE") || strings.HasPrefix(upper, "JOIN") || strings.HasPrefix(upper, "ON") {
						if hasLastChar && !isWhitespace(lastChar) {
							appendSQL(" ")
						}
					}
				}
			}

			appendSQL(value)

		case intermediate.OpEmitEval:
			if instr.ExprIndex == nil {
				return "", nil, fmt.Errorf("%w: instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return "", nil, fmt.Errorf("%w: instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			value, err := g.evaluateExpression(expr.Expression, params)
			if err != nil {
				return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			appendSQL("?")

			sqlParams = append(sqlParams, value)

		case intermediate.OpIf:
			if instr.ExprIndex == nil {
				return "", nil, fmt.Errorf("%w: IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return "", nil, fmt.Errorf("%w: IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			condition, err := g.evaluateCondition(expr.Expression, params)
			if err != nil {
				return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			conditionStack = append(conditionStack, condition)

		case intermediate.OpElseIf:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: ELSE_IF without matching IF", ErrUnsupportedOperation)
			}

			// If the current condition is true, set it to false (we already processed the true branch)
			if conditionStack[len(conditionStack)-1] {
				conditionStack[len(conditionStack)-1] = false
			} else {
				// Evaluate the else-if condition
				if instr.ExprIndex == nil {
					return "", nil, fmt.Errorf("%w: ELSE_IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
				}

				if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
					return "", nil, fmt.Errorf("%w: ELSE_IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
				}

				expr := g.expressions[*instr.ExprIndex]

				condition, err := g.evaluateCondition(expr.Expression, params)
				if err != nil {
					return "", nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
				}

				conditionStack[len(conditionStack)-1] = condition
			}

		case intermediate.OpElse:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: ELSE without matching IF", ErrUnsupportedOperation)
			}

			// Flip the condition for the else branch
			conditionStack[len(conditionStack)-1] = !conditionStack[len(conditionStack)-1]

		case intermediate.OpEnd:
			if len(conditionStack) == 0 {
				return "", nil, fmt.Errorf("%w: END without matching IF", ErrUnsupportedOperation)
			}

			// Pop the condition stack
			conditionStack = conditionStack[:len(conditionStack)-1]

		case intermediate.OpEmitUnlessBoundary:
			trimmed := strings.TrimSpace(instr.Value)

			upper := strings.ToUpper(trimmed)
			switch upper {
			case "AND", "OR":
				appendSpaceIfNeeded()
				appendSQL(trimmed)
				appendSQL(" ")
			default:
				appendSQL(instr.Value)
			}

		case intermediate.OpBoundary:
			// Skip boundary markers for now

		case intermediate.OpLoopStart, intermediate.OpLoopEnd:
			// Loop operations are not implemented yet
			return "", nil, fmt.Errorf("%w: loop operations not yet implemented", ErrUnsupportedOperation)

		case intermediate.OpIfSystemLimit:
			conditionStack = append(conditionStack, g.shouldEmitSystemClause(params, "limit"))
		case intermediate.OpIfSystemOffset:
			conditionStack = append(conditionStack, g.shouldEmitSystemClause(params, "offset"))
		case intermediate.OpEmitSystemLimit:
			limitLiteral := g.resolveSystemNumeric(instr.DefaultValue, "limit")
			appendSQL(limitLiteral)
		case intermediate.OpEmitSystemOffset:
			offsetLiteral := g.resolveSystemNumeric(instr.DefaultValue, "offset")
			appendSQL(offsetLiteral)
		case intermediate.OpEmitSystemValue:
			value, err := g.resolveSystemValue(instr.SystemField, params)
			if err != nil {
				return "", nil, err
			}

			appendSQL("?")

			sqlParams = append(sqlParams, value)

		default:
			return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedOperation, instr.Op)
		}
	}

	// Check for unmatched conditions
	if len(conditionStack) > 0 {
		return "", nil, fmt.Errorf("%w: unmatched IF statement", ErrUnsupportedOperation)
	}

	return result.String(), sqlParams, nil
}

func (g *SQLGenerator) shouldEmitSystemClause(params map[string]interface{}, key string) bool {
	if params == nil {
		return true
	}

	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case nil:
			return false
		case int, int32, int64:
			return toInt64(v) > 0
		case float32, float64:
			return toFloat64(v) > 0
		case string:
			return strings.TrimSpace(v) != ""
		default:
			return true
		}
	}

	return true
}

func (g *SQLGenerator) resolveSystemNumeric(defaultValue string, kind string) string {
	value := strings.TrimSpace(defaultValue)
	if value == "" {
		if kind == "limit" {
			return "1000"
		}

		return "0"
	}

	if _, err := strconv.Atoi(value); err == nil {
		return value
	}

	return value
}

func (g *SQLGenerator) resolveSystemValue(fieldName string, params map[string]interface{}) (any, error) {
	name := strings.ToLower(strings.TrimSpace(fieldName))
	if name == "" {
		return nil, fmt.Errorf("%w: system field name missing", ErrUnsupportedOperation)
	}

	if val, ok := params[name]; ok && val != nil {
		g.generated[name] = val
		return val, nil
	}

	if val, ok := g.generated[name]; ok {
		params[name] = val
		return val, nil
	}

	if implicit, ok := g.implicitMap[name]; ok {
		if implicit.Default != nil {
			value := g.normalizeSystemDefault(implicit.Default, implicit.Type, name)
			g.generated[name] = value
			params[name] = value

			return value, nil
		}

		value := g.generateFallbackValue(implicit.Type, name)
		g.generated[name] = value
		params[name] = value

		return value, nil
	}

	if field, ok := g.systemFields[name]; ok {
		if field.OnInsert != nil && field.OnInsert.Default != nil {
			value := g.normalizeSystemDefault(field.OnInsert.Default, "", name)
			g.generated[name] = value
			params[name] = value

			return value, nil
		}

		if field.OnUpdate != nil && field.OnUpdate.Default != nil {
			value := g.normalizeSystemDefault(field.OnUpdate.Default, "", name)
			g.generated[name] = value
			params[name] = value

			return value, nil
		}
	}

	value := g.generateFallbackValue("", name)
	g.generated[name] = value
	params[name] = value

	return value, nil
}

func (g *SQLGenerator) normalizeSystemDefault(raw any, typeName, fieldName string) any {
	switch v := raw.(type) {
	case nil:
		return g.generateFallbackValue(typeName, fieldName)
	case time.Time:
		return v
	case string:
		upper := strings.ToUpper(strings.TrimSpace(v))
		switch upper {
		case "NOW()", "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP()":
			return time.Now().UTC()
		case "UUID_GENERATE_V4()", "GEN_RANDOM_UUID()", "UUID()":
			return uuid.NewString()
		default:
			if num, err := strconv.Atoi(upper); err == nil {
				return num
			}

			return v
		}
	default:
		return v
	}
}

func (g *SQLGenerator) generateFallbackValue(typeName, fieldName string) any {
	lowerType := strings.ToLower(typeName)
	name := strings.ToLower(fieldName)

	if strings.Contains(lowerType, "time") || strings.HasSuffix(name, "_at") {
		return time.Now().UTC()
	}

	if strings.Contains(lowerType, "uuid") || strings.Contains(lowerType, "guid") || strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "_by") {
		return uuid.NewString()
	}

	if strings.Contains(lowerType, "int") {
		return randomSource.Int63()
	}

	if strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") || strings.Contains(lowerType, "decimal") {
		return randomSource.Float64()
	}

	if strings.Contains(lowerType, "bool") {
		return true
	}

	// Default to UUID string for other textual fields
	return uuid.NewString()
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case float64:
		return val
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// evaluateExpression evaluates a CEL expression and returns the result
func (g *SQLGenerator) evaluateExpression(expression string, params map[string]interface{}) (interface{}, error) {
	// Simple parameter lookup for now
	if value, exists := params[expression]; exists {
		return value, nil
	}

	// Try to evaluate as CEL expression
	ast, issues := g.celEnv.Compile(expression)
	if issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", expression, issues.Err())
	}

	program, err := g.celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program for expression '%s': %w", expression, err)
	}

	// Create evaluation context
	evalParams := map[string]interface{}{
		"params": params,
	}

	// Add individual parameters to the context for direct access
	for k, v := range params {
		evalParams[k] = v
	}

	result, _, err := program.Eval(evalParams)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expression, err)
	}

	return result.Value(), nil
}

// evaluateCondition evaluates a condition expression and returns a boolean result
func (g *SQLGenerator) evaluateCondition(expression string, params map[string]interface{}) (bool, error) {
	result, err := g.evaluateExpression(expression, params)
	if err != nil {
		return false, err
	}

	// Convert result to boolean (duplicated logic from previous implementation)
	switch v := result.(type) {
	case bool:
		return v, nil
	case *types.Bool:
		return bool(*v), nil
	case nil:
		return false, nil
	case int, int32, int64:
		return v != 0, nil
	case float32, float64:
		return v != 0.0, nil
	case string:
		return v != "", nil
	default:
		return true, nil // treat non-zero/non-empty as truthy
	}
}
