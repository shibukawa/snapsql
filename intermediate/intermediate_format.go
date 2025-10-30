package intermediate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
)

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
	IsNullable bool   `json:"is_nullable,omitempty"`
	MaxLength  *int   `json:"max_length,omitempty"`
	Precision  *int   `json:"precision,omitempty"`
	Scale      *int   `json:"scale,omitempty"`
	// HierarchyKeyLevel: 0=非PK, 1=ルートPK, 2=第一階層子PK, 3=第二階層子PK ...
	// a__b__c のような多段 prefix に対応する将来拡張を想定
	// 設定タイミング: SELECT 解析 Processor (未実装) が prefix 分解とスキーマ主キー照合で決定する予定
	HierarchyKeyLevel int `json:"hierarchy_key_level,omitempty"`
	// Internal only: precise source origin (not exported to final intermediate JSON)
	SourceTable  string `json:"-"`
	SourceColumn string `json:"-"`
}

// ImplicitParameter represents a parameter that should be obtained from context/TLS
type ImplicitParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

// EnvVar represents a variable available within a CEL environment level.
type EnvVar struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TableReferenceInfo represents a table reference in the query (including CTEs, subqueries, and joins)
type TableReferenceInfo struct {
	// Canonical identifier for the referenced object (physical table, CTE, or subquery)
	Name string `json:"name"`

	// Physical table name when it can be resolved via schema metadata
	TableName string `json:"table_name,omitempty"`

	// Alias used in the SQL text (if different from Name)
	Alias string `json:"alias,omitempty"`

	// Owning derived query (CTE/subquery) when the reference is defined inside it
	QueryName string `json:"query_name,omitempty"`

	// Context where this table is used ("main", "join", "cte", "subquery")
	Context string `json:"context,omitempty"`
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

	// StatementType describes the root SQL statement (select/insert/update/delete)
	StatementType string `json:"statement_type,omitempty"`

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

	// Warning messages emitted during intermediate generation (e.g., type inference degradation)
	Warnings []string `json:"warnings,omitempty"`

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

	// Table references used in the query (including CTEs, subqueries, and joins)
	TableReferences []TableReferenceInfo `json:"table_references,omitempty"`

	// MockTestCases stores parsed test cases for mock generation / WithMock integration
	MockTestCases []snapsql.MockTestCase `json:"test_cases,omitempty"`

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
func marshalCompact(v any) (json.RawMessage, error) {
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
