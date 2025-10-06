package query

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
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
	instructions    []intermediate.Instruction
	expressions     []intermediate.CELExpression
	systemFields    map[string]intermediate.SystemFieldInfo
	implicitMap     map[string]intermediate.ImplicitParameter
	generated       map[string]any
	dialect         string
	celEnv          *cel.Env
	loopBoundaries  map[int]int
	loopBoundaryErr error
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

	generator := &SQLGenerator{
		instructions: instructions,
		expressions:  expressions,
		systemFields: systemFields,
		implicitMap:  implicitMap,
		generated:    make(map[string]any),
		dialect:      dialect,
		celEnv:       env,
	}

	boundaries, err := computeLoopBoundaries(generator.instructions)
	generator.loopBoundaries = boundaries
	generator.loopBoundaryErr = err

	return generator
}

// Generate generates SQL and parameters from the instructions
func (g *SQLGenerator) Generate(params map[string]interface{}) (string, []interface{}, error) {
	if g.loopBoundaryErr != nil {
		return "", nil, g.loopBoundaryErr
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	// Reset generated cache per invocation to avoid leaking values across calls.
	g.generated = make(map[string]any)

	var result strings.Builder

	sqlParams := make([]interface{}, 0)
	state := &generationState{
		builder:   &result,
		sqlParams: &sqlParams,
	}
	state.boundaryEnabled = containsBoundary(g.instructions)

	conditionStack := make([]bool, 0)

	if err := g.processInstructions(state, params, &conditionStack, 0, len(g.instructions)); err != nil {
		return "", nil, err
	}

	if len(conditionStack) > 0 {
		return "", nil, fmt.Errorf("%w: unmatched IF statement", ErrUnsupportedOperation)
	}

	return result.String(), sqlParams, nil
}

type generationState struct {
	builder         *strings.Builder
	sqlParams       *[]interface{}
	lastChar        byte
	hasLast         bool
	boundaryEnabled bool
	boundaryNeeded  bool
}

func (s *generationState) appendSQL(text string) {
	if len(text) == 0 {
		return
	}

	s.builder.WriteString(text)
	s.lastChar = text[len(text)-1]
	s.hasLast = true
}

func (s *generationState) appendSpaceIfNeeded() {
	if s.hasLast && isWhitespace(s.lastChar) {
		return
	}

	s.builder.WriteByte(' ')
	s.lastChar = ' '
	s.hasLast = true
}

func (s *generationState) trimTrailingComma() {
	if s.builder.Len() == 0 {
		s.hasLast = false
		s.lastChar = 0

		return
	}

	current := s.builder.String()
	end := len(current)

	idx := end - 1
	for idx >= 0 && isWhitespaceByte(current[idx]) {
		idx--
	}

	if idx < 0 || current[idx] != ',' {
		return
	}

	prefix := current[:idx]
	suffix := current[idx+1:]

	s.builder.Reset()
	s.builder.WriteString(prefix)
	s.builder.WriteString(suffix)

	newLen := len(prefix) + len(suffix)
	if newLen == 0 {
		s.hasLast = false
		s.lastChar = 0

		return
	}

	if len(suffix) > 0 {
		s.lastChar = suffix[len(suffix)-1]
		s.hasLast = true

		return
	}

	s.lastChar = prefix[len(prefix)-1]
	s.hasLast = true
}

func isWhitespace(b byte) bool {
	switch b {
	case ' ', '\n', '\t', '\r':
		return true
	default:
		return false
	}
}

func (g *SQLGenerator) processInstructions(state *generationState, params map[string]interface{}, conditionStack *[]bool, start, end int) error {
	for i := start; i < end; i++ {
		instr := g.instructions[i]

		// Skip instructions in inactive conditional branches, but keep control-flow markers in sync.
		if len(*conditionStack) > 0 && !(*conditionStack)[len(*conditionStack)-1] {
			switch instr.Op {
			case intermediate.OpEnd, intermediate.OpElse, intermediate.OpElseIf:
				// Still need to evaluate control flow markers.
			default:
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
						if state.hasLast && !isWhitespace(state.lastChar) {
							state.appendSQL(" ")
						}
					}
				}
			}

			state.appendSQL(value)

			if state.boundaryEnabled && strings.TrimSpace(value) != "" {
				state.boundaryNeeded = true
			}

		case intermediate.OpEmitEval:
			if instr.ExprIndex == nil {
				return fmt.Errorf("%w: instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return fmt.Errorf("%w: instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			value, err := g.evaluateExpression(expr.Expression, params)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			state.appendSQL("?")

			*state.sqlParams = append(*state.sqlParams, value)
			if state.boundaryEnabled {
				state.boundaryNeeded = true
			}

		case intermediate.OpIf:
			if instr.ExprIndex == nil {
				return fmt.Errorf("%w: IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return fmt.Errorf("%w: IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			condition, err := g.evaluateCondition(expr.Expression, params)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			*conditionStack = append(*conditionStack, condition)

		case intermediate.OpElseIf:
			if len(*conditionStack) == 0 {
				return fmt.Errorf("%w: ELSE_IF without matching IF", ErrUnsupportedOperation)
			}

			if (*conditionStack)[len(*conditionStack)-1] {
				(*conditionStack)[len(*conditionStack)-1] = false
				continue
			}

			if instr.ExprIndex == nil {
				return fmt.Errorf("%w: ELSE_IF instruction %d has no expression index", ErrInvalidExpressionIndex, i)
			}

			if *instr.ExprIndex < 0 || *instr.ExprIndex >= len(g.expressions) {
				return fmt.Errorf("%w: ELSE_IF instruction %d has invalid expression index %d", ErrInvalidExpressionIndex, i, *instr.ExprIndex)
			}

			expr := g.expressions[*instr.ExprIndex]

			condition, err := g.evaluateCondition(expr.Expression, params)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
			}

			(*conditionStack)[len(*conditionStack)-1] = condition

		case intermediate.OpElse:
			if len(*conditionStack) == 0 {
				return fmt.Errorf("%w: ELSE without matching IF", ErrUnsupportedOperation)
			}

			(*conditionStack)[len(*conditionStack)-1] = !(*conditionStack)[len(*conditionStack)-1]

		case intermediate.OpEnd:
			if len(*conditionStack) == 0 {
				return fmt.Errorf("%w: END without matching IF", ErrUnsupportedOperation)
			}

			*conditionStack = (*conditionStack)[:len(*conditionStack)-1]

		case intermediate.OpEmitUnlessBoundary:
			if state.boundaryEnabled {
				token := padBoundaryToken(instr.Value)
				if state.boundaryNeeded {
					state.appendSQL(token)
				}

				state.boundaryNeeded = true
			} else {
				state.appendSQL(instr.Value)
			}

		case intermediate.OpBoundary:
			if state.boundaryEnabled {
				state.boundaryNeeded = false
			}

		case intermediate.OpLoopStart:
			endIndex, err := g.handleLoop(state, params, conditionStack, i, instr)
			if err != nil {
				return err
			}

			i = endIndex

		case intermediate.OpLoopEnd:
			// Loop end is handled when the corresponding start is processed.

		case intermediate.OpIfSystemLimit:
			*conditionStack = append(*conditionStack, g.shouldEmitSystemClause(params, "limit"))
		case intermediate.OpIfSystemOffset:
			*conditionStack = append(*conditionStack, g.shouldEmitSystemClause(params, "offset"))
		case intermediate.OpEmitSystemLimit:
			limitLiteral := g.resolveSystemNumeric(instr.DefaultValue, "limit")
			state.appendSQL(limitLiteral)
		case intermediate.OpEmitSystemOffset:
			offsetLiteral := g.resolveSystemNumeric(instr.DefaultValue, "offset")
			state.appendSQL(offsetLiteral)
		case intermediate.OpEmitSystemValue:
			value, err := g.resolveSystemValue(instr.SystemField, params)
			if err != nil {
				return err
			}

			state.appendSQL("?")

			*state.sqlParams = append(*state.sqlParams, value)
			if state.boundaryEnabled {
				state.boundaryNeeded = true
			}

		case intermediate.OpEmitIfDialect:
			if shouldEmitForDialect(g.dialect, instr.Dialects) {
				state.appendSQL(instr.SqlFragment)
			}

		default:
			return fmt.Errorf("%w: %s", ErrUnsupportedOperation, instr.Op)
		}
	}

	return nil
}

func (g *SQLGenerator) handleLoop(state *generationState, params map[string]interface{}, conditionStack *[]bool, startIndex int, instr intermediate.Instruction) (int, error) {
	if instr.Variable == "" {
		return 0, fmt.Errorf("%w: loop variable missing at instruction %d", ErrUnsupportedOperation, startIndex)
	}

	endIndex, ok := g.loopBoundaries[startIndex]
	if !ok {
		return 0, fmt.Errorf("%w: LOOP_START at instruction %d has no matching LOOP_END", ErrUnsupportedOperation, startIndex)
	}

	collection, err := g.evaluateLoopCollection(instr, startIndex, params)
	if err != nil {
		return 0, err
	}

	prevValue, hadPrev := params[instr.Variable]
	initialStackLen := len(*conditionStack)

	for _, element := range collection {
		params[instr.Variable] = element
		if err := g.processInstructions(state, params, conditionStack, startIndex+1, endIndex); err != nil {
			return 0, err
		}

		*conditionStack = (*conditionStack)[:initialStackLen]
	}

	if len(collection) > 0 {
		state.trimTrailingComma()
	}

	if hadPrev {
		params[instr.Variable] = prevValue
	} else {
		delete(params, instr.Variable)
	}

	return endIndex, nil
}

func (g *SQLGenerator) evaluateLoopCollection(instr intermediate.Instruction, index int, params map[string]interface{}) ([]interface{}, error) {
	var raw interface{}

	if instr.CollectionExprIndex != nil {
		exprIndex := *instr.CollectionExprIndex
		if exprIndex < 0 || exprIndex >= len(g.expressions) {
			return nil, fmt.Errorf("%w: LOOP_START instruction %d has invalid collection expression index %d", ErrInvalidExpressionIndex, index, exprIndex)
		}

		expr := g.expressions[exprIndex]

		value, err := g.evaluateExpression(expr.Expression, params)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrExpressionEvaluation, err)
		}

		raw = value
	} else if instr.Collection != "" {
		value, ok := params[instr.Collection]
		if !ok {
			return nil, fmt.Errorf("%w: collection %s not found for instruction %d", ErrParameterNotFound, instr.Collection, index)
		}

		raw = value
	} else {
		return nil, fmt.Errorf("%w: loop collection missing for instruction %d", ErrUnsupportedOperation, index)
	}

	return normalizeCollectionValue(raw)
}

func normalizeCollectionValue(value interface{}) ([]interface{}, error) {
	if value == nil {
		return []interface{}{}, nil
	}

	switch v := value.(type) {
	case []interface{}:
		return append([]interface{}{}, v...), nil
	case traits.Lister:
		iter := v.Iterator()

		result := make([]interface{}, 0)
		for hasNext(iter.HasNext()) {
			result = append(result, iter.Next().Value())
		}

		return result, nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		length := rv.Len()

		result := make([]interface{}, length)
		for i := range length {
			result[i] = rv.Index(i).Interface()
		}

		return result, nil
	}

	return nil, fmt.Errorf("%w: loop collection must be array or list, got %T", ErrUnsupportedOperation, value)
}

func hasNext(val ref.Val) bool {
	switch v := val.(type) {
	case types.Bool:
		return bool(v)
	default:
		if b, ok := val.Value().(bool); ok {
			return b
		}
	}

	return false
}

func shouldEmitForDialect(current string, targets []string) bool {
	if len(targets) == 0 {
		return true
	}

	current = strings.ToLower(strings.TrimSpace(current))
	for _, dialect := range targets {
		if current == strings.ToLower(strings.TrimSpace(dialect)) {
			return true
		}
	}

	return false
}

func computeLoopBoundaries(instructions []intermediate.Instruction) (map[int]int, error) {
	boundaries := make(map[int]int)
	stack := make([]int, 0)

	for idx, instr := range instructions {
		switch instr.Op {
		case intermediate.OpLoopStart:
			stack = append(stack, idx)
		case intermediate.OpLoopEnd:
			if len(stack) == 0 {
				return nil, fmt.Errorf("%w: LOOP_END without matching LOOP_START at instruction %d", ErrUnsupportedOperation, idx)
			}

			start := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			boundaries[start] = idx
		}
	}

	if len(stack) > 0 {
		return nil, fmt.Errorf("%w: LOOP_START without matching LOOP_END", ErrUnsupportedOperation)
	}

	return boundaries, nil
}

func containsBoundary(instructions []intermediate.Instruction) bool {
	for _, inst := range instructions {
		switch inst.Op {
		case intermediate.OpEmitUnlessBoundary, intermediate.OpBoundary:
			return true
		}
	}

	return false
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
