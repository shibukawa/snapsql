package pygen

import (
	"fmt"
	"strings"
)

// processParameters processes function parameters from intermediate format
// and generates parameter data for the Python template
func (g *Generator) processParameters() ([]parameterData, error) {
	if g.Format == nil || len(g.Format.Parameters) == 0 {
		return []parameterData{}, nil
	}

	params := make([]parameterData, 0, len(g.Format.Parameters))

	for _, param := range g.Format.Parameters {
		// Convert type to Python type hint
		pyType, err := ConvertToPythonType(param.Type, false)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter type %s: %w", param.Type, err)
		}

		// Convert parameter name to snake_case
		paramName := toSnakeCase(param.Name)

		// Create parameter data
		paramData := parameterData{
			Name:        paramName,
			TypeHint:    pyType,
			Description: param.Description,
			HasDefault:  param.Optional,
		}

		// Set default value for optional parameters
		if param.Optional {
			paramData.Default = "None"
		}

		params = append(params, paramData)
	}

	return params, nil
}

// processImplicitParameters processes implicit parameters (system columns)
// that are obtained from context variables
func (g *Generator) processImplicitParameters() ([]implicitParamData, error) {
	if g.Format == nil || len(g.Format.ImplicitParameters) == 0 {
		return []implicitParamData{}, nil
	}

	implicitParams := make([]implicitParamData, 0, len(g.Format.ImplicitParameters))

	for _, param := range g.Format.ImplicitParameters {
		// Convert type to Python type hint
		pyType, err := ConvertToPythonType(param.Type, false)
		if err != nil {
			return nil, fmt.Errorf("failed to convert implicit parameter type %s: %w", param.Type, err)
		}

		// Convert parameter name to snake_case
		paramName := toSnakeCase(param.Name)

		// Create implicit parameter data
		implicitData := implicitParamData{
			Name:     paramName,
			TypeHint: pyType,
			Description: fmt.Sprintf("System column: %s (from context if not provided)",
				paramName),
		}

		// Set default value if provided
		if param.Default != nil {
			implicitData.HasDefaultValue = true
			implicitData.DefaultValue = formatDefaultValue(param.Default, param.Type)
		}

		implicitParams = append(implicitParams, implicitData)
	}

	return implicitParams, nil
}

// processValidations generates parameter validation code
func (g *Generator) processValidations(params []parameterData) []validationData {
	validations := make([]validationData, 0)

	for _, param := range params {
		// Skip optional parameters (they can be None)
		if param.HasDefault {
			continue
		}

		// Add validation for required parameters
		validation := validationData{
			Condition: param.Name + " is None",
			Message:   fmt.Sprintf("Required parameter '%s' cannot be None", param.Name),
			ParamName: param.Name,
		}
		validations = append(validations, validation)
	}

	return validations
}

// formatDefaultValue formats a default value for Python code
func formatDefaultValue(value any, typeName string) string {
	if value == nil {
		return "None"
	}

	switch v := value.(type) {
	case string:
		// Check if it's a code expression (contains parentheses or dots)
		// If so, use as-is without quoting
		if strings.Contains(v, "(") || strings.Contains(v, ".") {
			return v
		}
		// Otherwise, quote it as a string literal
		return fmt.Sprintf("%q", v)
	case int, int32, int64, float32, float64:
		// For numeric types, use as-is
		return fmt.Sprintf("%v", v)
	case bool:
		// Python uses True/False (capitalized)
		if v {
			return "True"
		}

		return "False"
	default:
		// For other types, try to format as string
		return fmt.Sprintf("%v", v)
	}
}
