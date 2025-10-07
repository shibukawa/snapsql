package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
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
	"any":       "any",
}

// celEnvironmentData represents a CEL environment for code generation
type celEnvironmentData struct {
	Index     int
	Container string
	HasParent bool
	Parent    int
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

		if env.Container != "" {
			envData.Container = env.Container
		} else {
			if env.Index == 0 {
				envData.Container = "root"
			} else {
				envData.Container = fmt.Sprintf("env_%d", env.Index)
			}
		}

		if env.ParentIndex != nil {
			envData.HasParent = true
			envData.Parent = *env.ParentIndex
		} else if env.Index > 0 {
			// Fallback for older intermediate files without explicit parent information
			envData.HasParent = true
			envData.Parent = env.Index - 1
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

	// Check for custom types (file paths or relative paths)
	if strings.Contains(baseType, "/") || strings.Contains(baseType, ".") {
		// This is a custom type - extract the type name
		typeName := extractTypeNameFromPath(baseType)

		// Handle arrays and pointers
		goType := typeName
		celType := fmt.Sprintf("types.NewObjectType(\"%s\")", typeName)

		isArray := strings.HasSuffix(v.Type, "[]")
		if isArray {
			goType = "[]" + goType
			celType = "ListType(" + celType + ")"
		}

		if strings.HasSuffix(v.Type, "*") {
			goType = "*" + goType
		}

		return celVariableData{
			Name:     v.Name,
			CelType:  celType,
			GoType:   goType,
			IsArray:  isArray,
			IsObject: true,
		}, nil
	}

	normalizedBase := normalizeTemporalAlias(baseType)

	celType, ok := celTypeMap[normalizedBase]
	if !ok {
		return celVariableData{}, fmt.Errorf("%w: %s", snapsql.ErrUnsupportedType, v.Type)
	}

	goType, ok := goTypeMap[normalizedBase]
	if !ok {
		return celVariableData{}, fmt.Errorf("%w: %s", snapsql.ErrUnsupportedType, v.Type)
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

// extractTypeNameFromPath extracts the type name from a file path
func extractTypeNameFromPath(typePath string) string {
	// Handle relative paths like "./User" or "../testdata/acceptancetests/017_common_type_ok/User"
	parts := strings.Split(typePath, "/")
	return parts[len(parts)-1] // Return the last part as type name
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
