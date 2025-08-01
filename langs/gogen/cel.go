package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// celTypeMap maps SnapSQL types to CEL type names
var celTypeMap = map[string]string{
	"int":       "IntType",
	"int32":     "IntType",
	"int64":     "IntType",
	"string":    "StringType",
	"bool":      "BoolType",
	"float":     "DoubleType",
	"float32":   "DoubleType",
	"float64":   "DoubleType",
	"decimal":   "DoubleType", // Using double for decimal
	"timestamp": "TimestampType",
	"date":      "TimestampType",
	"time":      "TimestampType",
	"bytes":     "BytesType",
	"any":       "AnyType",
}

// goTypeMap maps SnapSQL types to Go types
var goTypeMap = map[string]string{
	"int":       "int",
	"int32":     "int32",
	"int64":     "int64",
	"string":    "string",
	"bool":      "bool",
	"float":     "float64",
	"float32":   "float32",
	"float64":   "float64",
	"decimal":   "decimal.Decimal",
	"timestamp": "time.Time",
	"date":      "time.Time",
	"time":      "time.Time",
	"bytes":     "[]byte",
	"any":       "interface{}",
}

// celEnvironmentData represents a CEL environment for code generation
type celEnvironmentData struct {
	Index     int
	Variables []celVariableData
	Imports   map[string]struct{} // Track required imports
}

// celVariableData represents a CEL variable for code generation
type celVariableData struct {
	Name     string
	Type     string
	CelType  string
	GoType   string
	IsArray  bool
	IsObject bool
}

// processCELEnvironments processes CEL environments and returns template data
func processCELEnvironments(format *intermediate.IntermediateFormat) ([]celEnvironmentData, error) {
	envs := make([]celEnvironmentData, len(format.CELEnvironments))

	for i, env := range format.CELEnvironments {
		envData := celEnvironmentData{
			Index:     env.Index,
			Variables: make([]celVariableData, 0),
			Imports:   make(map[string]struct{}),
		}

		// Process variables from parameters for environment 0
		if i == 0 {
			for _, param := range format.Parameters {
				varData, err := processCELVariable(intermediate.CELVariableInfo{
					Name: param.Name,
					Type: param.Type,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to process parameter %s: %w", param.Name, err)
				}

				// Track imports
				if strings.Contains(varData.GoType, ".") {
					pkg := strings.Split(varData.GoType, ".")[0]
					envData.Imports[pkg] = struct{}{}
				}

				envData.Variables = append(envData.Variables, varData)
			}
		}

		// Process additional variables
		for _, v := range env.AdditionalVariables {
			varData, err := processCELVariable(v)
			if err != nil {
				return nil, fmt.Errorf("failed to process variable %s: %w", v.Name, err)
			}

			// Track imports
			if strings.Contains(varData.GoType, ".") {
				pkg := strings.Split(varData.GoType, ".")[0]
				envData.Imports[pkg] = struct{}{}
			}

			envData.Variables = append(envData.Variables, varData)
		}

		envs[i] = envData
	}

	return envs, nil
}

// processCELVariable converts a SnapSQL variable to CEL variable data
func processCELVariable(v intermediate.CELVariableInfo) (celVariableData, error) {
	baseType := strings.TrimSuffix(strings.TrimSuffix(v.Type, "[]"), "*")

	celType, ok := celTypeMap[strings.ToLower(baseType)]
	if !ok {
		return celVariableData{}, fmt.Errorf("unsupported type: %s", v.Type)
	}

	goType, ok := goTypeMap[strings.ToLower(baseType)]
	if !ok {
		return celVariableData{}, fmt.Errorf("unsupported type: %s", v.Type)
	}

	// Handle arrays and pointers
	isArray := strings.HasSuffix(v.Type, "[]")
	if isArray {
		goType = "[]" + goType
		celType = "ListType(cel." + celType + ")"
	}
	if strings.HasSuffix(v.Type, "*") {
		goType = "*" + goType
	}

	return celVariableData{
		Name:     v.Name,
		Type:     v.Type,
		CelType:  celType,
		GoType:   goType,
		IsArray:  isArray,
		IsObject: strings.HasPrefix(v.Type, "."), // Types starting with . are objects
	}, nil
}

// generateCELPrograms generates CEL program initialization code
func generateCELPrograms(format *intermediate.IntermediateFormat, envs []celEnvironmentData) ([]celProgramData, error) {
	programs := make([]celProgramData, len(format.CELExpressions))

	for i, expr := range format.CELExpressions {
		program := celProgramData{
			Index:          i,
			ID:             expr.ID,
			Expression:     expr.Expression,
			EnvironmentIdx: expr.EnvironmentIndex,
		}
		programs[i] = program
	}

	return programs, nil
}

// celProgramData represents a CEL program for code generation
type celProgramData struct {
	Index          int
	ID             string
	Expression     string
	EnvironmentIdx int
}
