package parsercommon

import (
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
)

// Namespace manages namespace information and CEL functionality in an integrated manner

type frame struct {
	cel        *cel.Env
	param      map[string]any
	loopTarget []any
	loopIndex  int
	loopVar    string
}
type Namespace struct {
	environment map[string]any // CEL evaluation variables
	envCEL      *cel.Env       // CEL environment for environment variables
	stack       []frame        // CEL environment for parameters
}

// NewNamespace creates a new namespace
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

	// Register environment constants directly as CEL variables (no prefix)
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

	paramOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}
	for key := range param {
		paramOptions = append(paramOptions, cel.Variable(key, cel.DynType))
	}
	paramCEL, err := cel.NewEnv(paramOptions...)
	if err != nil {
		panic(err.Error())
	}

	ns := &Namespace{
		environment: environment,
		stack: []frame{
			{
				cel:   paramCEL,
				param: param,
			},
		},
		envCEL: envCEL,
	}
	return ns
}

// Copy creates a copy of the namespace
func (ns *Namespace) EnterLoop(varName string, loopTarget []any) bool {
	if len(loopTarget) == 0 {
		return false
	}
	last := ns.stack[len(ns.stack)-1]
	newParam := maps.Clone(last.param)

	// Generate dummy value based on the first element type
	visited := make(map[string]bool)
	dummyValue := generateDummyValueWithPath(loopTarget[0], visited, varName)
	newParam[varName] = dummyValue // Set dummy value for loop variable

	paramOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}
	for key := range newParam {
		paramOptions = append(paramOptions, cel.Variable(key, cel.DynType))
	}
	paramCEL, err := cel.NewEnv(paramOptions...)
	if err != nil {
		return false
	}

	newFrame := frame{
		cel:        paramCEL,
		loopTarget: loopTarget,
		loopIndex:  0,
		loopVar:    varName,
		param:      newParam,
	}
	ns.stack = append(ns.stack, newFrame)
	return true
}

func (ns *Namespace) Next() bool {
	if len(ns.stack) == 0 {
		return false
	}
	last := ns.stack[len(ns.stack)-1]
	if last.loopIndex+1 >= len(last.loopTarget) {
		return false // No more elements in loop
	}

	// Move to next element in loop
	last.loopIndex++
	last.param[last.loopVar] = last.loopTarget[last.loopIndex]
	return true
}

func (ns *Namespace) LeaveLoop() {
	if len(ns.stack) > 1 {
		ns.stack = ns.stack[:len(ns.stack)-1]
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

// EvaluateEnvironmentExpression evaluates environment constant expressions (/*@ */)
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

// EvaluateParameterExpression evaluates parameter expressions in dummy data environment
func (ns *Namespace) EvaluateParameterExpression(expression string) (any, error) {
	// Compile CEL expression
	last := ns.stack[len(ns.stack)-1]
	ast, issues := last.cel.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create program
	program, err := last.cel.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Evaluate using dummy data
	result, _, err := program.Eval(last.param)
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

// generateDummyDataFromSchema generates dummy data environment from schema (function style)
func generateDummyDataFromSchema(schema *FunctionDefinition) map[string]any {
	if schema == nil || schema.Parameters == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)
	visited := make(map[string]bool) // Prevent circular references using path-based tracking
	for key, typeInfo := range schema.Parameters {
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
