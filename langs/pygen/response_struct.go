package pygen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// processResponseStruct processes response fields and generates dataclass data
// It returns all structs (children first) plus the main struct pointer for downstream consumers.
func processResponseStruct(format *intermediate.IntermediateFormat) ([]responseStructData, *responseStructData, error) {
	if len(format.Responses) == 0 {
		// No response fields - this is normal for INSERT/UPDATE/DELETE statements
		return nil, nil, ErrNoResponseFields
	}

	// Check for hierarchical structure
	hierarchicalGroups, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	if len(hierarchicalGroups) > 0 {
		// This is a hierarchical response - use hierarchical processing
		childStructs, mainStruct, err := generateHierarchicalStructs(format.FunctionName, hierarchicalGroups, rootFields)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate hierarchical structs: %w", err)
		}

		allStructs := make([]responseStructData, 0, len(childStructs)+1)
		allStructs = append(allStructs, childStructs...)
		allStructs = append(allStructs, *mainStruct)

		return allStructs, mainStruct, nil
	}

	// Regular flat structure
	className := generateClassName(format.FunctionName)

	fields := make([]responseFieldData, len(format.Responses))

	for i, response := range format.Responses {
		pyType, err := ConvertToPythonType(response.Type, response.IsNullable)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert response field %s type: %w", response.Name, err)
		}

		// Convert field name to snake_case (Python convention)
		fieldName := toSnakeCase(response.Name)

		// Determine if field has default value (for Optional fields)
		hasDefault := response.IsNullable
		defaultValue := "None"

		fields[i] = responseFieldData{
			Name:       fieldName,
			TypeHint:   pyType,
			HasDefault: hasDefault,
			Default:    defaultValue,
		}
	}

	fields = reorderResponseFields(fields)

	mainStruct := &responseStructData{
		ClassName: className,
		Fields:    fields,
	}

	return []responseStructData{*mainStruct}, mainStruct, nil
}

func reorderResponseFields(fields []responseFieldData) []responseFieldData {
	if len(fields) == 0 {
		return fields
	}

	required := make([]responseFieldData, 0, len(fields))
	optional := make([]responseFieldData, 0)

	for _, f := range fields {
		if f.HasDefault {
			optional = append(optional, f)
			continue
		}

		required = append(required, f)
	}

	return append(required, optional...)
}

// generateClassName generates a Python class name from a function name
// Example: "get_user_by_id" -> "GetUserByIdResult"
func generateClassName(functionName string) string {
	// Convert snake_case to PascalCase
	parts := strings.Split(functionName, "_")
	result := make([]string, len(parts))

	for i, part := range parts {
		if part == "" {
			continue
		}
		// Capitalize first letter
		result[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(result, "") + "Result"
}
