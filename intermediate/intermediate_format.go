package intermediate

import (
	"encoding/json"
	"fmt"
)

// Instruction operation types
const (
	// Basic output instructions
	OpEmitLiteral = "EMIT_LITERAL" // Output literal text
	OpEmitParam   = "EMIT_PARAM"   // Output parameter value
	OpEmitEval    = "EMIT_EVAL"    // Output evaluated expression

	// Control flow instructions
	OpJump        = "JUMP"          // Unconditional jump
	OpJumpIfExp   = "JUMP_IF_EXP"   // Conditional jump based on complex expression
	OpJumpIfParam = "JUMP_IF_PARAM" // Conditional jump based on simple parameter
	OpLabel       = "LABEL"         // Jump target label

	// Loop instructions
	OpLoopStartParam = "LOOP_START_PARAM" // Start of loop block with simple parameter collection
	OpLoopStartExp   = "LOOP_START_EXP"   // Start of loop block with complex expression collection
	OpLoopNext       = "LOOP_NEXT"        // Next iteration
	OpLoopEnd        = "LOOP_END"         // End of loop block

	// System directive instructions
	OpEmitExplain      = "EMIT_EXPLAIN"       // Output EXPLAIN clause
	OpJumpIfForceLimit = "JUMP_IF_FORCE_LIMIT" // Jump if force limit is set
	OpJumpIfForceOffset = "JUMP_IF_FORCE_OFFSET" // Jump if force offset is set
	OpEmitSystemFields = "EMIT_SYSTEM_FIELDS" // Output system fields
	OpEmitSystemValues = "EMIT_SYSTEM_VALUES" // Output system values
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op          string `json:"op"`
	Pos         []int  `json:"pos,omitempty"`          // Position [line, column, offset] from original template
	Value       string `json:"value,omitempty"`        // For EMIT_LITERAL
	Param       string `json:"param,omitempty"`        // For EMIT_PARAM, JUMP_IF_PARAM
	ExpIndex    int    `json:"exp_index,omitempty"`    // Index into pre-compiled expressions
	EnvLevel    int    `json:"env_level,omitempty"`    // Environment level for CEL evaluation
	Placeholder string `json:"placeholder,omitempty"`  // For EMIT_PARAM, EMIT_EVAL
	Target      int    `json:"target,omitempty"`       // For JUMP, JUMP_IF_EXP, JUMP_IF_PARAM
	Name        string `json:"name,omitempty"`         // For LABEL
	Variable    string `json:"variable,omitempty"`     // For LOOP_START_PARAM, LOOP_START_EXP
	Collection  string `json:"collection,omitempty"`   // For LOOP_START_PARAM
	CollectionExpIndex int `json:"collection_exp_index,omitempty"` // Index into pre-compiled expressions for collection
	EndLabel    string `json:"end_label,omitempty"`    // For LOOP_START_PARAM, LOOP_START_EXP
	StartLabel  string `json:"start_label,omitempty"`  // For LOOP_NEXT
	Label       string `json:"label,omitempty"`        // For LOOP_END
	Analyze     bool   `json:"analyze,omitempty"`      // For EMIT_EXPLAIN
	Fields      []string `json:"fields,omitempty"`     // For EMIT_SYSTEM_FIELDS, EMIT_SYSTEM_VALUES
}

// InterfaceSchema contains extracted parameter definitions and metadata
type InterfaceSchema struct {
	Name         string      `json:"name"`
	FunctionName string      `json:"function_name"`
	Parameters   []Parameter `json:"parameters"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

// ResponseType represents the expected result structure
type ResponseType struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
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

// IntermediateFormat represents the enhanced intermediate file format
type IntermediateFormat struct {
	// Interface schema (input parameters)
	InterfaceSchema *InterfaceSchema `json:"interface_schema,omitempty"`

	// Response type information
	ResponseType *ResponseType `json:"response_type,omitempty"`
	
	// Response affinity (database type mapping)
	ResponseAffinity string `json:"response_affinity,omitempty"`

	// Instruction sequence
	Instructions []Instruction `json:"instructions"`
	
	// Complex CEL expressions
	CELExpressions []string `json:"cel_expressions,omitempty"`
	
	// Simple variables
	SimpleVars []string `json:"simple_vars,omitempty"`
	
	// Environment variables by level
	Envs [][]EnvVar `json:"envs,omitempty"`
}

// ToJSON serializes the intermediate format to JSON
func (f *IntermediateFormat) ToJSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
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
