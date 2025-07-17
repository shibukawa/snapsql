package parsercommon

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
)

// Namespace manages namespace information and CEL functionality in an integrated manner

type frame struct {
	variables  map[string]any // Variables added at this scope level
	param      map[string]any // Parameter values at this scope level (for compatibility)
	loopTarget []any          // Loop target array (if this is a loop frame)
	loopIndex  int            // Current loop index
	loopVar    string         // Loop variable name
}

type Namespace struct {
	environment  map[string]any // CEL evaluation variables
	envCEL       *cel.Env       // CEL environment for environment variables
	paramBaseCEL *cel.Env       // Base CEL environment for parameters
	currentCEL   *cel.Env       // Current CEL environment with all variables
	stack        []frame        // Stack of variable scopes
}

// NewNamespace creates a new namespace with efficient CEL environment management
// If schema is nil, creates an empty InterfaceSchema
func NewNamespace(schema *FunctionDefinition, environment, param map[string]any) *Namespace {
	// Create empty schema if nil
	if schema == nil {
		schema = &FunctionDefinition{
			Parameters: make(map[string]any),
		}
	}

	if param == nil {
		param = generateDummyDataFromSchema(schema)
	}

	// Create environment CEL environment
	envOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}
	for key := range environment {
		envOptions = append(envOptions, cel.Variable(key, cel.DynType))
	}

	envCEL, err := cel.NewEnv(envOptions...)
	if err != nil {
		panic(err.Error())
	}

	// Create base CEL environment for parameters with type inference
	paramBaseOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}

	// Add typed variable declarations for parameters
	if schema != nil && schema.Parameters != nil {
		paramBaseOptions = append(paramBaseOptions, createCELVariableDeclarations(schema.Parameters)...)
	} else {
		// Fallback to dynamic type for all parameters
		for key := range param {
			paramBaseOptions = append(paramBaseOptions, cel.Variable(key, cel.DynType))
		}
	}
	paramBaseCEL, err := cel.NewEnv(paramBaseOptions...)
	if err != nil {
		panic(err.Error())
	}

	ns := &Namespace{
		environment:  environment,
		envCEL:       envCEL,
		paramBaseCEL: paramBaseCEL,
		currentCEL:   paramBaseCEL, // Initially same as base
		stack: []frame{
			{
				variables: make(map[string]any), // No additional variables at base level
				param:     param,
			},
		},
	}
	return ns
}

// rebuildCurrentCEL rebuilds the current CEL environment with all variables from the stack
func (ns *Namespace) rebuildCurrentCEL() error {
	// Start with base parameter environment options (reuse base declarations)
	options := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}

	// Add base parameter variables (these already have proper types from base env)
	baseFrame := ns.stack[0]
	for key := range baseFrame.param {
		// Reuse type inference from value or use dyn as fallback
		celType := inferCELTypeFromValue(baseFrame.param[key])
		options = append(options, cel.Variable(key, celType))
	}

	// Add variables from all stack frames (loop variables etc.)
	for _, frame := range ns.stack {
		for key := range frame.variables {
			// Loop variables and dynamic variables use dyn type
			options = append(options, cel.Variable(key, cel.DynType))
		}
	}

	// Create new environment
	currentCEL, err := cel.NewEnv(options...)
	if err != nil {
		return err
	}

	ns.currentCEL = currentCEL
	return nil
}

// EnterLoop enters a new loop scope with efficient variable management
func (ns *Namespace) EnterLoop(varName string, loopTarget []any) bool {
	if len(loopTarget) == 0 {
		return false
	}

	// Generate dummy value based on the first element type
	visited := make(map[string]bool)
	dummyValue := generateDummyValueWithPath(loopTarget[0], visited, varName)

	// Create new frame with only the loop variable
	newFrame := frame{
		variables: map[string]any{
			varName: dummyValue,
		},
		param:      make(map[string]any), // Empty for compatibility
		loopTarget: loopTarget,
		loopIndex:  0,
		loopVar:    varName,
	}
	ns.stack = append(ns.stack, newFrame)

	// Rebuild CEL environment to include the new variable
	if err := ns.rebuildCurrentCEL(); err != nil {
		// Rollback on error
		ns.stack = ns.stack[:len(ns.stack)-1]
		return false
	}

	return true
}

func (ns *Namespace) Next() bool {
	if len(ns.stack) == 0 {
		return false
	}
	last := &ns.stack[len(ns.stack)-1]
	if last.loopIndex+1 >= len(last.loopTarget) {
		return false // No more elements in loop
	}

	// Move to next element in loop
	last.loopIndex++
	last.variables[last.loopVar] = last.loopTarget[last.loopIndex]
	return true
}

func (ns *Namespace) LeaveLoop() {
	if len(ns.stack) > 1 {
		ns.stack = ns.stack[:len(ns.stack)-1]
		// Rebuild CEL environment after leaving loop
		ns.rebuildCurrentCEL()
	}
}

// valueToLiteral converts values to SQL literals
func (ns *Namespace) valueToLiteral(value any) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes in strings
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []any:
		// Convert array to comma-separated literals
		var literals []string
		for _, item := range v {
			literals = append(literals, ns.valueToLiteral(item))
		}
		return strings.Join(literals, ", ")
	case []string:
		// Special handling for string arrays
		var literals []string
		for _, item := range v {
			literals = append(literals, fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''")))
		}
		return strings.Join(literals, ", ")
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// EvaluateEnvironmentExpression evaluates environment constant expressions (/*# */)
func (ns *Namespace) EvaluateEnvironmentExpression(expression string) (any, error) {
	if ns.envCEL == nil {
		return nil, ErrEnvironmentCELNotInit
	}

	// Compile CEL expression
	ast, issues := ns.envCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create program
	program, err := ns.envCEL.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Evaluate expression
	result, _, err := program.Eval(ns.environment)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation error: %w", err)
	}

	return result.Value(), nil
}

// ParameterDefinition represents parameter definition
type ParameterDefinition struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// getCELTypeFromValue gets CEL type from value
func (ns *Namespace) getCELTypeFromValue(value any) *cel.Type {
	if value == nil {
		return cel.AnyType
	}

	switch v := value.(type) {
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case float32, float64:
		return cel.DoubleType
	case bool:
		return cel.BoolType
	case []string:
		return cel.ListType(cel.StringType)
	case []int:
		return cel.ListType(cel.IntType)
	case []int64:
		return cel.ListType(cel.IntType)
	case []float64:
		return cel.ListType(cel.DoubleType)
	case []bool:
		return cel.ListType(cel.BoolType)
	case []any:
		// Infer element type
		if len(v) > 0 {
			elementType := ns.getCELTypeFromValue(v[0])
			if elementType != nil {
				return cel.ListType(elementType)
			}
		}
		return cel.ListType(cel.AnyType)
	case map[string]any:
		// For nested objects, use MapType (complex object type definition is difficult in CEL)
		return cel.MapType(cel.StringType, cel.AnyType)
	default:
		// Use reflection for more detailed type inference
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return cel.ListType(cel.AnyType)
		case reflect.Map:
			return cel.MapType(cel.StringType, cel.AnyType)
		default:
			return cel.AnyType
		}
	}
}

// EvaluateParameterExpression evaluates parameter expressions using current CEL environment
func (ns *Namespace) EvaluateParameterExpression(expression string) (any, error) {
	// Compile CEL expression using current environment
	ast, issues := ns.currentCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create program
	program, err := ns.currentCEL.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Collect all variables from stack
	allVars := make(map[string]any)

	// Add base parameters
	baseFrame := ns.stack[0]
	for k, v := range baseFrame.param {
		allVars[k] = v
	}

	// Add variables from all stack frames
	for _, frame := range ns.stack {
		for k, v := range frame.variables {
			allVars[k] = v
		}
	}

	// Evaluate using collected variables
	result, _, err := program.Eval(allVars)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation error: %w", err)
	}

	return result.Value(), nil
}

// extractElementFromList extracts element value and type from list result
func (ns *Namespace) extractElementFromList(listResult any) (elementValue any, elementType string, err error) {
	switch list := listResult.(type) {
	case []any:
		if len(list) > 0 {
			element := list[0]
			return element, ns.inferTypeFromValue(element), nil
		}
		return "", "str", nil // Default for empty list
	case []string:
		if len(list) > 0 {
			return list[0], "str", nil
		}
		return "", "str", nil
	case []int:
		if len(list) > 0 {
			return list[0], "int", nil
		}
		return 0, "int", nil
	case []int64:
		if len(list) > 0 {
			return list[0], "int", nil
		}
		return 0, "int", nil
	case []float64:
		if len(list) > 0 {
			return list[0], "float", nil
		}
		return 0.0, "float", nil
	case []bool:
		if len(list) > 0 {
			return list[0], "bool", nil
		}
		return false, "bool", nil
	default:
		// Error if not a list
		return nil, "", ErrExpressionNotList
	}
}

// inferTypeFromValue infers type from value
func (ns *Namespace) inferTypeFromValue(value any) string {
	switch value.(type) {
	case string:
		return "str"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "float"
	case bool:
		return "bool"
	case map[string]any:
		// For nested objects, return as object type
		return "object"
	case []any:
		return "list"
	default:
		return "any"
	}
}

// generateDummyDataFromSchema generates dummy data environment from schema excluding nullable parameters
func generateDummyDataFromSchema(schema *FunctionDefinition) map[string]any {
	if schema == nil || schema.Parameters == nil {
		return make(map[string]any)
	}

	// Use createCleanParameterMap to exclude nullable parameters
	cleanParams := createCleanParameterMap(schema.Parameters)

	result := make(map[string]any)
	visited := make(map[string]bool) // Prevent circular references using path-based tracking
	for key, typeInfo := range cleanParams {
		result[key] = generateDummyValueWithPath(typeInfo, visited, key)
	}
	return result
}

// generateDummyValueWithPath generates dummy value while detecting circular references by path
func generateDummyValueWithPath(typeInfo any, visited map[string]bool, path string) any {
	// Check for circular reference by path
	if visited[path] {
		return "" // Return default value for circular reference
	}
	visited[path] = true
	defer delete(visited, path) // Remove from visited after processing

	switch t := typeInfo.(type) {
	case string:
		return generateDummyValueFromString(t)
	case []any:
		// For array types, use first element as template
		if len(t) > 0 {
			elementTemplate := t[0]
			elementValue := generateDummyValueWithPath(elementTemplate, visited, path+"[0]")
			// Return appropriate array type based on element type
			switch elementTemplate {
			case "str", "string":
				return []string{"dummy"}
			case "int", "integer":
				return []int{0}
			case "float", "double":
				return []float64{0.0}
			case "bool", "boolean":
				return []bool{false}
			default:
				return []any{elementValue}
			}
		}
		return []any{}
	case map[string]any:
		// For object types, process recursively
		result := make(map[string]any)
		for key, value := range t {
			childPath := path + "." + key
			result[key] = generateDummyValueWithPath(value, visited, childPath)
		}
		return result
	default:
		return ""
	}
}

// generateDummyValueFromString generates dummy value from string type definition
func generateDummyValueFromString(typeStr string) any {
	switch typeStr {
	case "str", "string":
		return ""
	case "int", "integer":
		return 0
	case "float", "double":
		return 0.0
	case "bool", "boolean":
		return false
	case "list[str]", "[]string":
		return []string{"dummy"}
	case "list[int]", "[]int":
		return []int{0}
	case "list[float]", "[]float":
		return []float64{0.0}
	case "list[bool]", "[]bool":
		return []bool{false}
	case "map[str]", "map[string]any":
		return map[string]any{"": ""}
	default:
		// Parse list[T] pattern
		if len(typeStr) > 5 && typeStr[:5] == "list[" && typeStr[len(typeStr)-1] == ']' {
			elementType := typeStr[5 : len(typeStr)-1]
			elementValue := generateDummyValueFromString(elementType)
			return []any{elementValue}
		}
		// Default to empty string
		return ""
	}
}

// inferCELTypeFromValue infers CEL type from Go value
func inferCELTypeFromValue(value any) *cel.Type {
	if value == nil {
		return cel.DynType
	}

	switch value.(type) {
	case string:
		return cel.StringType
	case int, int8, int16, int32, int64:
		return cel.IntType
	case float32, float64:
		return cel.DoubleType
	case bool:
		return cel.BoolType
	case time.Time:
		return cel.TimestampType
	default:
		return cel.DynType
	}
}

// inferCELTypeFromStringType infers CEL type from parameter type string
func inferCELTypeFromStringType(typeStr string) *cel.Type {
	switch typeStr {
	case "str", "string":
		return cel.StringType
	case "int", "integer":
		return cel.IntType
	case "float", "double":
		return cel.DoubleType
	case "bool", "boolean":
		return cel.BoolType
	case "timestamp", "time":
		return cel.TimestampType
	default:
		return cel.DynType
	}
}

// isNullableKey checks if parameter key ends with '?' for nullable
func isNullableKey(key string) (string, bool) {
	if strings.HasSuffix(key, "?") {
		return key[:len(key)-1], true
	}
	return key, false
}

// createCELVariableDeclarations creates cel.VariableDecls from parameter map with type inference
func createCELVariableDeclarations(parameters map[string]any) []cel.EnvOption {
	var options []cel.EnvOption

	for key, value := range parameters {
		cleanKey, isNullable := isNullableKey(key)

		var celType *cel.Type
		if isNullable {
			// Nullable parameters use dyn type
			celType = cel.DynType
		} else {
			// Infer type from value or type string
			if typeStr, ok := value.(string); ok {
				celType = inferCELTypeFromStringType(typeStr)
			} else {
				celType = inferCELTypeFromValue(value)
			}
		}

		options = append(options, cel.Variable(cleanKey, celType))
	}

	return options
}

// createCleanParameterMap creates parameter map excluding nullable keys for dummy data
func createCleanParameterMap(parameters map[string]any) map[string]any {
	clean := make(map[string]any)

	for key, value := range parameters {
		cleanKey, isNullable := isNullableKey(key)
		if !isNullable {
			clean[cleanKey] = value
		}
	}

	return clean
}

// GetLoopVariableType returns the type of a loop variable if it exists in the current stack
// Returns the type string and true if found, empty string and false if not found
func (ns *Namespace) GetLoopVariableType(variableName string) (string, bool) {
	// Check from current frame to outer frames
	for i := len(ns.stack) - 1; i >= 0; i-- {
		frame := ns.stack[i]
		if frame.loopVar == variableName {
			// Found as loop variable - infer type from loop target
			if frame.loopTarget != nil && len(frame.loopTarget) > 0 {
				return ns.inferLoopVariableType(frame.loopTarget), true
			}
			// Default to string if no loop target info
			return "string", true
		}
	}
	return "", false
}

// inferLoopVariableType infers type from loop target (usually an array element)
func (ns *Namespace) inferLoopVariableType(loopTarget []any) string {
	if len(loopTarget) == 0 {
		return "string"
	}

	// Use the existing inferTypeFromValue method
	return ns.inferTypeFromValue(loopTarget[0])
}
