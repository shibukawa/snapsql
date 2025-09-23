package intermediate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Instruction operation types
const (
	// Basic output instructions
	OpEmitStatic = "EMIT_STATIC" // Output static text
	OpEmitEval   = "EMIT_EVAL"   // Output evaluated expression

	// Boundary instructions for conditional delimiter handling
	OpEmitUnlessBoundary = "EMIT_UNLESS_BOUNDARY" // Output text unless followed by boundary
	OpBoundary           = "BOUNDARY"             // Mark boundary for delimiter removal

	// Control flow instructions
	OpIf     = "IF"      // Start of if block
	OpElseIf = "ELSE_IF" // Else if condition
	OpElse   = "ELSE"    // Else block
	OpEnd    = "END"     // End of control block (if, for)

	// Loop instructions
	OpLoopStart = "LOOP_START" // Start of for loop block
	OpLoopEnd   = "LOOP_END"   // End of for loop block

	// System directive instructions
	OpIfSystemLimit    = "IF_SYSTEM_LIMIT"    // Conditional based on system limit
	OpIfSystemOffset   = "IF_SYSTEM_OFFSET"   // Conditional based on system offset
	OpEmitSystemLimit  = "EMIT_SYSTEM_LIMIT"  // Output system limit value
	OpEmitSystemOffset = "EMIT_SYSTEM_OFFSET" // Output system offset value
	OpEmitSystemValue  = "EMIT_SYSTEM_VALUE"  // Output system value for specific field

	// Database dialect instructions
	OpEmitIfDialect = "EMIT_IF_DIALECT" // Output SQL fragment if current dialect matches
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op                  string `json:"op"`
	Pos                 string `json:"pos,omitempty"`                   // Position "line:column" from original template
	Value               string `json:"value,omitempty"`                 // For EMIT_STATIC
	Param               string `json:"param,omitempty"`                 // For EMIT_PARAM (deprecated, use ExprIndex)
	ExprIndex           *int   `json:"expr_index,omitempty"`            // Index into expressions array
	Condition           string `json:"condition,omitempty"`             // For IF, ELSE_IF (deprecated, use ExprIndex)
	Variable            string `json:"variable,omitempty"`              // For FOR
	Collection          string `json:"collection,omitempty"`            // For FOR (deprecated, use CollectionExprIndex)
	CollectionExprIndex *int   `json:"collection_expr_index,omitempty"` // Index into expressions array for collection
	EnvIndex            *int   `json:"env_index,omitempty"`             // Environment index for LOOP_START/LOOP_END
	DefaultValue        string `json:"default_value,omitempty"`         // For EMIT_SYSTEM_LIMIT, EMIT_SYSTEM_OFFSET
	SystemField         string `json:"system_field,omitempty"`          // For EMIT_SYSTEM_VALUE - system field name

	// Database dialect fields
	SqlFragment string   `json:"sql_fragment,omitempty"` // For EMIT_IF_DIALECT - SQL fragment to output
	Dialects    []string `json:"dialects,omitempty"`     // For EMIT_IF_DIALECT - target database dialects
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

// Response represents a result field
type Response struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	BaseType   string `json:"base_type,omitempty"`
	IsNullable bool   `json:"is_nullable,omitempty"`
	MaxLength  *int   `json:"max_length,omitempty"`
	Precision  *int   `json:"precision,omitempty"`
	Scale      *int   `json:"scale,omitempty"`
}

// ImplicitParameter represents a parameter that should be obtained from context/TLS
type ImplicitParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

// SystemFieldInfo represents system field configuration in intermediate format
type SystemFieldInfo struct {
	// Field name
	Name string `json:"name"`

	// Whether to exclude this field from SELECT statements by default
	ExcludeFromSelect bool `json:"exclude_from_select,omitempty"`

	// Configuration for INSERT operations
	OnInsert *SystemFieldOperationInfo `json:"on_insert,omitempty"`

	// Configuration for UPDATE operations
	OnUpdate *SystemFieldOperationInfo `json:"on_update,omitempty"`
}

// SystemFieldOperationInfo represents the configuration for a system field in a specific operation
type SystemFieldOperationInfo struct {
	// Default value (if specified, this field gets this default value)
	// Can be any type: string, int, bool, nil for SQL NULL, etc.
	Default any `json:"default,omitempty"`

	// Parameter configuration (how this field should be handled as a parameter)
	// Values: "explicit", "implicit", "error", ""
	Parameter string `json:"parameter,omitempty"`
}

// IntermediateFormat represents the enhanced intermediate file format
type IntermediateFormat struct {
	// Format version
	FormatVersion string `json:"format_version"`

	// Query name
	Name string `json:"name,omitempty"`

	// Description of the query/function
	Description string `json:"description,omitempty"`

	// Function name for code generation
	FunctionName string `json:"function_name,omitempty"`

	// Parameters for the query
	Parameters []Parameter `json:"parameters,omitempty"`

	// Response fields (simplified structure)
	Responses []Response `json:"responses,omitempty"`

	// Response affinity (database type mapping)
	ResponseAffinity string `json:"response_affinity,omitempty"`

	// Instruction sequence
	Instructions []Instruction `json:"instructions"`

	// Enhanced CEL expressions with metadata
	CELExpressions []CELExpression `json:"cel_expressions"`

	// CEL environments with variable definitions
	CELEnvironments []CELEnvironment `json:"cel_environments"`

	// Environment variables by level
	Envs [][]EnvVar `json:"envs,omitempty"`

	// Cache keys for frequently evaluated expressions
	CacheKeys []string `json:"cache_keys,omitempty"`

	// System fields configuration
	SystemFields []SystemFieldInfo `json:"system_fields,omitempty"`

	// Implicit parameters that should be obtained from context/TLS
	ImplicitParameters []ImplicitParameter `json:"implicit_parameters,omitempty"`

	// Indicates whether the main statement guarantees ordered results via ORDER BY
	HasOrderedResult bool `json:"has_ordered_result,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for IntermediateFormat
func (f *IntermediateFormat) MarshalJSON() ([]byte, error) {
	// Create a custom struct for marshaling
	type Alias IntermediateFormat

	// Marshal the base fields
	baseJSON, err := json.Marshal((*Alias)(f))
	if err != nil {
		return nil, err
	}

	// Unmarshal into a map for manipulation
	var result map[string]json.RawMessage
	if err := json.Unmarshal(baseJSON, &result); err != nil {
		return nil, err
	}

	// Custom marshal for Parameters
	if len(f.Parameters) > 0 {
		params, err := marshalCompact(f.Parameters)
		if err != nil {
			return nil, err
		}

		result["parameters"] = params
	}

	// Custom marshal for Instructions
	if len(f.Instructions) > 0 {
		instructions, err := marshalCompact(f.Instructions)
		if err != nil {
			return nil, err
		}

		result["instructions"] = instructions
	}

	// Custom marshal for Responses
	if len(f.Responses) > 0 {
		responses, err := marshalCompact(f.Responses)
		if err != nil {
			return nil, err
		}

		result["responses"] = responses
	}

	// Custom marshal for Envs
	if len(f.Envs) > 0 {
		envs, err := marshalCompact(f.Envs)
		if err != nil {
			return nil, err
		}

		result["envs"] = envs
	}

	// Custom marshal for CELExpressions
	if len(f.CELExpressions) > 0 {
		celExpressions, err := marshalCompact(f.CELExpressions)
		if err != nil {
			return nil, err
		}

		result["cel_expressions"] = celExpressions
	}

	// Custom marshal for CELEnvironments
	if len(f.CELEnvironments) > 0 {
		celEnvironments, err := marshalCompact(f.CELEnvironments)
		if err != nil {
			return nil, err
		}

		result["cel_environments"] = celEnvironments
	}

	if f.HasOrderedResult {
		ordered, err := json.Marshal(f.HasOrderedResult)
		if err != nil {
			return nil, err
		}

		result["has_ordered_result"] = ordered
	}

	// Marshal the modified map back to JSON
	return json.Marshal(result)
}

// marshalCompact marshals an array in a compact format
func marshalCompact(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// For testing purposes, we'll keep the indentation for now
	// In a real implementation, we would use a more compact format
	return data, nil
}

// ToJSON serializes the intermediate format to JSON with improved formatting
func (f *IntermediateFormat) ToJSON() ([]byte, error) {
	// Use standard JSON formatting with custom array compacting
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intermediate format: %w", err)
	}

	// Apply custom formatting to make arrays more compact
	formatted := compactArraysInJSON(string(data))

	return []byte(formatted), nil
}

// compactArraysInJSON makes simple objects in arrays more compact
func compactArraysInJSON(jsonStr string) string {
	lines := strings.Split(jsonStr, "\n")

	var result []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if this line starts an array with objects
		if strings.Contains(line, ": [") && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "{" {
			// Look for simple objects in the array
			arrayStart := i
			arrayIndent := getIndentLevel(line)

			// Process array elements
			j := i + 1

			var arrayElements []string

			currentElement := []string{}

			for j < len(lines) {
				currentLine := lines[j]
				currentIndent := getIndentLevel(currentLine)

				// End of array
				if strings.TrimSpace(currentLine) == "]" || strings.TrimSpace(currentLine) == "]," {
					if len(currentElement) > 0 {
						if isSimpleObject(currentElement) {
							arrayElements = append(arrayElements, compactObject(currentElement, arrayIndent+2))
						} else {
							arrayElements = append(arrayElements, strings.Join(currentElement, "\n"))
						}
					}

					// Reconstruct the array
					result = append(result, line)
					result = append(result, arrayElements...)

					result = append(result, currentLine)
					i = j

					break
				}

				// Start of new object
				if strings.TrimSpace(currentLine) == "{" && currentIndent > arrayIndent {
					if len(currentElement) > 0 {
						if isSimpleObject(currentElement) {
							arrayElements = append(arrayElements, compactObject(currentElement, arrayIndent+2))
						} else {
							arrayElements = append(arrayElements, strings.Join(currentElement, "\n"))
						}
					}

					currentElement = []string{currentLine}
				} else {
					currentElement = append(currentElement, currentLine)
				}

				j++
			}

			if j >= len(lines) {
				// Fallback: add original lines if we couldn't process the array
				for k := arrayStart; k < len(lines); k++ {
					result = append(result, lines[k])
				}

				break
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// getIndentLevel returns the indentation level of a line
func getIndentLevel(line string) int {
	count := 0

	for _, char := range line {
		switch char {
		case ' ':
			count++
		case '\t':
			count += 4 // Treat tab as 4 spaces
		default:
			return count
		}
	}

	return count
}

// isSimpleObject checks if an object is simple enough to be compacted
func isSimpleObject(lines []string) bool {
	if len(lines) > 6 { // Don't compact large objects
		return false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check for nested objects or arrays
		if strings.Contains(trimmed, "{") && trimmed != "{" && trimmed != "}," && trimmed != "}" {
			return false
		}

		if strings.Contains(trimmed, "[") && trimmed != "[]" {
			return false
		}
	}

	return true
}

// compactObject converts a multi-line object to a single line
func compactObject(lines []string, indent int) string {
	var parts []string

	indentStr := strings.Repeat(" ", indent)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	if len(parts) == 0 {
		return indentStr + "{}"
	}

	// Join parts and clean up spacing
	compact := strings.Join(parts, " ")
	compact = strings.ReplaceAll(compact, "{ ", "{")
	compact = strings.ReplaceAll(compact, " }", "}")
	compact = strings.ReplaceAll(compact, " ,", ",")

	return indentStr + compact
}

// FromJSON deserializes the intermediate format from JSON
func FromJSON(data []byte) (*IntermediateFormat, error) {
	var format IntermediateFormat

	err := json.Unmarshal(data, &format)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intermediate format: %w", err)
	}

	return &format, nil
}

// CEL-related type definitions

// CELExpression represents a CEL expression with its metadata
type CELExpression struct {
	ID               string   `json:"id"`
	Expression       string   `json:"expression"`
	EnvironmentIndex int      `json:"environment_index"`
	Position         Position `json:"position,omitempty"`
}

// CELEnvironment represents a CEL environment with variable definitions
type CELEnvironment struct {
	Index               int               `json:"index"`
	AdditionalVariables []CELVariableInfo `json:"additional_variables"`
}

// CELVariableInfo represents information about a CEL variable
type CELVariableInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Position represents the position of an expression in the source
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}
