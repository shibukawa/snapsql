package intermediate

// MetadataExtractor extracts function metadata and parameters
type MetadataExtractor struct{}

func (m *MetadataExtractor) Name() string {
	return "MetadataExtractor"
}

func (m *MetadataExtractor) Process(ctx *ProcessingContext) error {
	// Extract function information from the function definition
	if ctx.FunctionDef != nil {
		ctx.FunctionName = ctx.FunctionDef.FunctionName
		ctx.Description = ctx.FunctionDef.Description

		// Convert function parameters to intermediate format parameters
		ctx.Parameters = make([]Parameter, 0, len(ctx.FunctionDef.ParameterOrder))
		for _, paramName := range ctx.FunctionDef.ParameterOrder {
			// Use OriginalParameters for parameter type names (preserves common type names)
			originalParamValue := ctx.FunctionDef.OriginalParameters[paramName]

			var (
				paramType   string
				description string
				optional    bool
			)

			if originalParamValue != nil {
				// Check if this is a JSON parameter structure (has "type" field)
				if paramMap, ok := originalParamValue.(map[string]any); ok {
					if typeVal, hasType := paramMap["type"]; hasType {
						// This is a JSON parameter definition like {"type": "int", "description": "...", "optional": true}
						if typeStr, ok := typeVal.(string); ok {
							paramType = typeStr
						}

						if descVal, hasDesc := paramMap["description"]; hasDesc {
							if descStr, ok := descVal.(string); ok {
								description = descStr
							}
						}

						if optVal, hasOpt := paramMap["optional"]; hasOpt {
							if optBool, ok := optVal.(bool); ok {
								optional = optBool
							}
						}
					} else {
						// Regular parameter structure, extract type
						paramType = extractParameterTypeFromOriginal(originalParamValue)
					}
				} else {
					// For common types, use the original type name (e.g., "User", "Department[]")
					paramType = extractParameterTypeFromOriginal(originalParamValue)
				}
			} else {
				// Fallback to normalized type if original is not available
				paramValue := ctx.FunctionDef.Parameters[paramName]
				paramType = extractParameterType(paramValue)
			}

			// Add the parameter
			ctx.Parameters = append(ctx.Parameters, Parameter{
				Name:        paramName,
				Type:        paramType,
				Description: description,
				Optional:    optional,
			})
		}
	}

	return nil
}
