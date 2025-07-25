package intermediate

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Instruction operation types
const (
	// Basic output instructions
	OpEmitStatic = "EMIT_STATIC" // Output static text
	OpEmitEval   = "EMIT_EVAL"   // Output evaluated expression

	// Control flow instructions
	OpIf     = "IF"      // Start of if block
	OpElseIf = "ELSE_IF" // Else if condition
	OpElse   = "ELSE"    // Else block
	OpEnd    = "END"     // End of control block (if, for)

	// Loop instructions
	OpLoopStart = "LOOP_START" // Start of for loop block
	OpLoopEnd   = "LOOP_END"   // End of for loop block

	// System directive instructions
	OpEmitSystemExplain = "EMIT_SYSTEM_EXPLAIN" // Output EXPLAIN clause
	OpIfSystemLimit     = "IF_SYSTEM_LIMIT"     // Conditional based on system limit
	OpIfSystemOffset    = "IF_SYSTEM_OFFSET"    // Conditional based on system offset
	OpEmitSystemLimit   = "EMIT_SYSTEM_LIMIT"   // Output system limit value
	OpEmitSystemOffset  = "EMIT_SYSTEM_OFFSET"  // Output system offset value
	OpEmitSystemFields  = "EMIT_SYSTEM_FIELDS"  // Output system fields
	OpEmitSystemValues  = "EMIT_SYSTEM_VALUES"  // Output system values
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op                  string   `json:"op"`
	Pos                 string   `json:"pos,omitempty"`                   // Position "line:column" from original template
	Value               string   `json:"value,omitempty"`                 // For EMIT_STATIC
	Param               string   `json:"param,omitempty"`                 // For EMIT_PARAM (deprecated, use ExprIndex)
	ExprIndex           *int     `json:"expr_index,omitempty"`            // Index into expressions array
	Condition           string   `json:"condition,omitempty"`             // For IF, ELSE_IF (deprecated, use ExprIndex)
	Variable            string   `json:"variable,omitempty"`              // For FOR
	Collection          string   `json:"collection,omitempty"`            // For FOR (deprecated, use CollectionExprIndex)
	CollectionExprIndex *int     `json:"collection_expr_index,omitempty"` // Index into expressions array for collection
	DefaultValue        string   `json:"default_value,omitempty"`         // For EMIT_SYSTEM_LIMIT, EMIT_SYSTEM_OFFSET
	Fields              []string `json:"fields,omitempty"`                // For EMIT_SYSTEM_FIELDS, EMIT_SYSTEM_VALUES
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

// Field represents a result field
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DatabaseTag string `json:"database_tag,omitempty"`
	BaseType    string `json:"base_type,omitempty"`
	IsNullable  bool   `json:"is_nullable,omitempty"`
	MaxLength   *int   `json:"max_length,omitempty"`
	Precision   *int   `json:"precision,omitempty"`
	Scale       *int   `json:"scale,omitempty"`
}

// Response represents the expected result structure
type Response struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// IntermediateFormat represents the enhanced intermediate file format
type IntermediateFormat struct {
	// Format version
	FormatVersion string `json:"format_version"`

	// Query name
	Name string `json:"name,omitempty"`

	// Function name for code generation
	FunctionName string `json:"function_name,omitempty"`

	// Parameters for the query
	Parameters []Parameter `json:"parameters,omitempty"`

	// Response definitions
	Responses []Response `json:"responses,omitempty"`

	// Response affinity (database type mapping)
	ResponseAffinity string `json:"response_affinity,omitempty"`

	// Instruction sequence
	Instructions []Instruction `json:"instructions"`

	// CEL expressions
	Expressions []string `json:"expressions,omitempty"`

	// Environment variables by level
	Envs [][]EnvVar `json:"envs,omitempty"`

	// Cache keys for frequently evaluated expressions
	CacheKeys []string `json:"cache_keys,omitempty"`
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

// ToJSON serializes the intermediate format to JSON
func (f *IntermediateFormat) ToJSON() ([]byte, error) {
	// Use MarshalIndent for pretty printing
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}

	// Make arrays more compact
	data = compactArrays(data)

	return data, nil
}

// compactArrays makes arrays more compact in the JSON output
func compactArrays(data []byte) []byte {
	// This is a simplified implementation
	// In a real implementation, we would use a more sophisticated approach

	// Replace arrays of simple values with compact format
	data = bytes.ReplaceAll(data, []byte("[\n      "), []byte("["))
	data = bytes.ReplaceAll(data, []byte(",\n      "), []byte(", "))
	data = bytes.ReplaceAll(data, []byte("\n    ]"), []byte("]"))

	return data
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
