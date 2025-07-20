package parsercommon

import (
	"fmt"
	"maps"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/uuid"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
	"github.com/shopspring/decimal"
)

type frame struct {
	variable cel.EnvOption
	values   map[string]any
	env      *cel.Env
}

type Namespace struct {
	fd               *FunctionDefinition
	frames           []frame
	currentEnv       *cel.Env
	currentValues    map[string]any
	currentVariables cel.EnvOption
}

func NewNamespaceFromDefinition(fd *FunctionDefinition) (*Namespace, error) {
	var vars []*decls.VariableDecl
	for key, val := range fd.Parameters {
		vars = append(vars, decls.NewVariable(key, snapSqlToCel(val)))
	}

	root := cel.VariableDecls(vars...)
	current, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		root,
	)
	if err != nil {
		return nil, err
	}
	result := &Namespace{
		fd:               fd,
		currentVariables: root,
		currentEnv:       current,
		currentValues:    fd.DummyData().(map[string]any),
	}

	return result, nil
}

func NewNamespaceFromConstants(constants map[string]any) (*Namespace, error) {
	var consts []*decls.VariableDecl
	for key, val := range constants {
		consts = append(consts, decls.NewVariable(key, snapSqlToCel(inferTypeStringFromActualValues(val, nil))))
	}
	root := cel.VariableDecls(consts...)
	current, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		root,
	)
	if err != nil {
		return nil, err
	}
	result := &Namespace{
		currentVariables: root,
		currentEnv:       current,
		currentValues:    constants,
	}
	return result, nil
}

func (ns *Namespace) Eval(exp string) (value any, tp string, err error) {
	ast, issues := ns.currentEnv.Parse(exp)
	if issues != nil && issues.Err() != nil {
		return nil, "", fmt.Errorf("%w: CEL expression parse error: %v", ErrInvalidForSnapSQL, issues.Err())
	}
	checked, issues := ns.currentEnv.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, "", fmt.Errorf("%w: CEL expression check error: %v", ErrInvalidForSnapSQL, issues.Err())
	}
	prg, err := ns.currentEnv.Program(checked)
	if err != nil {
		return nil, "", fmt.Errorf("%w: CEL program creation error: %v", ErrInvalidForSnapSQL, err)
	}

	v, _, err := prg.Eval(ns.currentValues)
	if err != nil {
		return nil, "", fmt.Errorf("%w: CEL program evaluation error: %v", ErrInvalidForSnapSQL, err)
	}
	if ns.fd != nil {
		return v.Value(), inferTypeStringFromDummyValue(v.Value()), nil
	}
	if v.Type() == cel.BytesType {
		var result, _ = v.Value().([]byte)
		if len(result) == 16 {
			uuidObj, err := uuid.FromBytes(result)
			if err != nil {
				return nil, "", fmt.Errorf("%w: error converting bytes to UUID: %v", ErrInvalidForSnapSQL, err)
			}
			return uuidObj, "uuid", nil
		}
	}
	result := v.Value()
	return result, inferTypeStringFromActualValues(result, v.Type()), nil
}

// EnterLoop creates a new frame for a loop variable
// It can accept either an expression string or a slice of values
func (ns *Namespace) EnterLoop(variableName string, loopTarget any) error {
	var a []any

	// Handle different types of loop targets
	switch v := loopTarget.(type) {
	case string:
		// If the loop target is a string, evaluate it as an expression
		val, _, err := ns.Eval(v)
		if err != nil {
			return err
		}

		// Check if the result is a slice
		var ok bool
		a, ok = val.([]any)
		if !ok {
			return fmt.Errorf("%w: expected array for loop variable %s, got %T", ErrInvalidForSnapSQL, variableName, val)
		}
	case []any:
		// If the loop target is already a slice, use it directly
		a = v
	default:
		return fmt.Errorf("%w: expected array or expression for loop variable %s, got %T", ErrInvalidForSnapSQL, variableName, loopTarget)
	}

	// If the slice is empty, return an error
	if len(a) == 0 {
		return fmt.Errorf("%w: empty array for loop variable %s", ErrInvalidForSnapSQL, variableName)
	}

	// Save the current frame
	ns.frames = append(ns.frames, frame{
		variable: ns.currentVariables,
		values:   ns.currentValues,
		env:      ns.currentEnv,
	})

	// Create a new frame with the loop variable
	newVariable := cel.Variable(variableName, snapSqlToCel(inferTypeStringFromDummyValue(a[0])))
	newValues := maps.Clone(ns.currentValues)
	newValues[variableName] = a[0] // Set the first item as the loop variable

	// Create a new environment with all previous variables plus the new loop variable
	options := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}
	for _, f := range ns.frames {
		options = append(options, f.variable)
	}
	options = append(options, newVariable)
	newEnv, err := cel.NewEnv(options...)
	if err != nil {
		return fmt.Errorf("%w: error creating new CEL environment for loop: %v", ErrInvalidForSnapSQL, err)
	}

	// Update the current frame
	ns.currentEnv = newEnv
	ns.currentValues = newValues
	ns.currentVariables = newVariable
	return nil
}

func (ns *Namespace) ExitLoop() error {
	if len(ns.frames) == 0 {
		return fmt.Errorf("%w: no loop to exit", ErrInvalidForSnapSQL)
	}
	lastFrame := ns.frames[len(ns.frames)-1]
	ns.frames = ns.frames[:len(ns.frames)-1]
	ns.currentEnv = lastFrame.env
	ns.currentValues = lastFrame.values
	ns.currentVariables = lastFrame.variable
	return nil
}

func snapSqlToCel(val any) *cel.Type {
	switch val {
	// --- Primitive and alias types ---
	case "string":
		return cel.StringType
	case "int":
		return cel.IntType
	case "int32":
		return cel.IntType
	case "int16":
		return cel.IntType
	case "int8":
		return cel.IntType
	case "float":
		return cel.DoubleType
	case "decimal":
		return snapsqlgo.DecimalType
	case "bool":
		return cel.BoolType
	// --- Special types ---
	case "date":
		return cel.StringType
	case "datetime", "timestamp", "time":
		return cel.TimestampType
	case "email":
		return cel.StringType
	case "uuid":
		return cel.StringType
	case "json":
		return cel.MapType(cel.StringType, cel.DynType)
	case "list":
		return cel.ListType(cel.DynType)
	case "any", "map":
		return cel.DynType
	default:
		switch val.(type) {
		case []any:
			return cel.ListType(cel.DynType)
		case map[string]any:
			return cel.DynType
		}
	}
	panic(fmt.Sprintf("Unsupported type for CEL conversion: %T of %v", val, val))
}

func inferTypeStringFromActualValues(v any, rt ref.Type) string {
	switch v2 := v.(type) {
	case int, int64, int32, int16, int8, uint64, uint32, uint16, uint8:
		return "int"
	case string:
		return "string"
	case bool:
		return "bool"
	case float64, float32:
		return "float"
	case uuid.UUID, [16]byte:
		return "uuid"
	case []byte:
		if rt == cel.BytesType {
			if len(v2) == 16 {
				return "uuid"
			}
		}
		return "string"
	case []any:
		return "list"
	case map[string]any:
		return "map"
	case time.Time:
		return "timestamp"
	case *snapsqlgo.Decimal, decimal.Decimal:
		return "decimal"
	default:
		return "unknown"
	}
}

// NewNamespace creates a new Namespace from a function definition and optional environment and parameters.
// This is a convenience function that handles the different cases of creating a Namespace.
func NewNamespace(fd *FunctionDefinition, environment map[string]any, parameters map[string]any) *Namespace {
	// If no function definition is provided, create a namespace from constants
	if fd == nil {
		// Combine environment and parameters into a single map
		constants := make(map[string]any)
		for k, v := range environment {
			constants[k] = v
		}
		for k, v := range parameters {
			constants[k] = v
		}

		// Create a namespace from constants
		ns, err := NewNamespaceFromConstants(constants)
		if err != nil {
			// This should not happen with valid constants
			panic(fmt.Sprintf("Failed to create namespace from constants: %v", err))
		}
		return ns
	}

	// Create a namespace from the function definition
	ns, err := NewNamespaceFromDefinition(fd)
	if err != nil {
		// This should not happen with a valid function definition
		panic(fmt.Sprintf("Failed to create namespace from function definition: %v", err))
	}

	// Override dummy data with provided parameters if any
	if parameters != nil {
		for k, v := range parameters {
			ns.currentValues[k] = v
		}
	}

	// Add environment variables
	for k, v := range environment {
		ns.currentValues[k] = v
	}

	return ns
}

// GetLoopVariableType returns the type of a loop variable and whether it exists
func (ns *Namespace) GetLoopVariableType(variableName string) (string, bool) {
	if val, ok := ns.currentValues[variableName]; ok {
		return inferTypeStringFromActualValues(val, nil), true
	}
	return "", false
}
