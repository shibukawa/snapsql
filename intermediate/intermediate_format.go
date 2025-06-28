package intermediate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op          string `json:"op"`
	Pos         []int  `json:"pos"`                   // Position [line, column, offset] from original template (required)
	Value       string `json:"value,omitempty"`       // For EMIT_LITERAL
	Param       string `json:"param,omitempty"`       // For EMIT_PARAM
	Exp         string `json:"exp,omitempty"`         // For EMIT_EVAL, JUMP_IF_EXP
	Placeholder string `json:"placeholder,omitempty"` // For EMIT_PARAM, EMIT_EVAL
	Target      int    `json:"target,omitempty"`      // For JUMP, JUMP_IF_EXP
	Name        string `json:"name,omitempty"`        // For LABEL
	Variable    string `json:"variable,omitempty"`    // For LOOP_START, LOOP_END
	Collection  string `json:"collection,omitempty"`  // For LOOP_START
	EndLabel    string `json:"end_label,omitempty"`   // For LOOP_START
	StartLabel  string `json:"start_label,omitempty"` // For LOOP_NEXT
	Label       string `json:"label,omitempty"`       // For LOOP_END
}

// IntermediateFormat represents the enhanced intermediate file format
type IntermediateFormat struct {
	// Source information
	Source SourceInfo `json:"source"`

	// Interface schema (optional)
	InterfaceSchema *InterfaceSchema `json:"interface_schema,omitempty"`

	// Instruction sequence
	Instructions []Instruction `json:"instructions"`

	// Variable dependencies for caching optimization
	Dependencies VariableDependencies `json:"dependencies"`

	// Metadata
	Metadata FormatMetadata `json:"metadata"`
}

// SourceInfo contains information about the original template file
type SourceInfo struct {
	File    string `json:"file"`
	Content string `json:"content"`
	Hash    string `json:"hash"` // SHA-256 hash of content
}

// InterfaceSchema contains extracted parameter definitions and metadata
type InterfaceSchema struct {
	Name         string      `json:"name"`
	FunctionName string      `json:"function_name"`
	Parameters   []Parameter `json:"parameters"`
	ResultType   *ResultType `json:"result_type,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

// ResultType represents the expected result structure
type ResultType struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field represents a result field
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DatabaseTag string `json:"database_tag,omitempty"`
	// --- typeinference 型情報 ---
	BaseType   string `json:"base_type,omitempty"`
	IsNullable bool   `json:"is_nullable,omitempty"`
	MaxLength  *int   `json:"max_length,omitempty"`
	Precision  *int   `json:"precision,omitempty"`
	Scale      *int   `json:"scale,omitempty"`
}

// VariableDependencies contains variable dependency information for optimization
type VariableDependencies struct {
	// All variables referenced in the template
	AllVariables []string `json:"all_variables"`

	// Variables that affect SQL structure (if/for conditions)
	StructuralVariables []string `json:"structural_variables"`

	// Variables that only affect parameter values
	ParameterVariables []string `json:"parameter_variables"`

	// Dependency graph for advanced optimization
	DependencyGraph map[string][]string `json:"dependency_graph"`

	// Cache key template for SQL reuse
	CacheKeyTemplate string `json:"cache_key_template"`
}

// FormatMetadata contains metadata about the intermediate format
type FormatMetadata struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
	Generator   string `json:"generator"`
	SchemaURL   string `json:"schema_url"`
}

// VariableExtractor extracts variable dependencies from CEL expressions and instructions
type VariableExtractor struct {
	allVars        map[string]bool
	structuralVars map[string]bool
	parameterVars  map[string]bool
	dependencies   map[string]map[string]bool
}

// NewVariableExtractor creates a new variable extractor
func NewVariableExtractor() *VariableExtractor {
	return &VariableExtractor{
		allVars:        make(map[string]bool),
		structuralVars: make(map[string]bool),
		parameterVars:  make(map[string]bool),
		dependencies:   make(map[string]map[string]bool),
	}
}

// ExtractFromInstructions extracts variable dependencies from instruction sequence
func (ve *VariableExtractor) ExtractFromInstructions(instructions []Instruction) VariableDependencies {
	// Reset state
	ve.allVars = make(map[string]bool)
	ve.structuralVars = make(map[string]bool)
	ve.parameterVars = make(map[string]bool)
	ve.dependencies = make(map[string]map[string]bool)

	for _, inst := range instructions {
		switch inst.Op {
		case "JUMP_IF_EXP":
			// Structural variables - affect SQL structure
			vars := ve.extractVariablesFromCEL(inst.Exp)
			for _, v := range vars {
				ve.allVars[v] = true
				ve.structuralVars[v] = true
			}
		case "LOOP_START":
			// Collection variables - affect SQL structure
			vars := ve.extractVariablesFromCEL(inst.Collection)
			for _, v := range vars {
				ve.allVars[v] = true
				ve.structuralVars[v] = true
			}
		case "EMIT_PARAM":
			// Parameter variables - only affect values
			if inst.Param != "" {
				vars := ve.extractVariablesFromPath(inst.Param)
				for _, v := range vars {
					ve.allVars[v] = true
					ve.parameterVars[v] = true
				}
			}
		case "EMIT_EVAL":
			// CEL expression variables - only affect values
			vars := ve.extractVariablesFromCEL(inst.Exp)
			for _, v := range vars {
				ve.allVars[v] = true
				ve.parameterVars[v] = true
			}
		}
	}

	return VariableDependencies{
		AllVariables:        ve.mapKeysToSlice(ve.allVars),
		StructuralVariables: ve.mapKeysToSlice(ve.structuralVars),
		ParameterVariables:  ve.mapKeysToSlice(ve.parameterVars),
		DependencyGraph:     ve.buildDependencyGraph(),
		CacheKeyTemplate:    ve.generateCacheKeyTemplate(),
	}
}

// extractVariablesFromCEL extracts variable names from CEL expressions
func (ve *VariableExtractor) extractVariablesFromCEL(expression string) []string {
	if expression == "" {
		return nil
	}

	// Simple variable extraction - can be enhanced with proper CEL parsing
	vars := make(map[string]bool)

	// Handle negation
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "!") {
		expr = strings.TrimPrefix(expr, "!")
		if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
			expr = strings.TrimPrefix(strings.TrimSuffix(expr, ")"), "(")
		}
	}

	// Handle logical operators
	for _, op := range []string{" || ", " && "} {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			for _, part := range parts {
				subVars := ve.extractVariablesFromCEL(strings.TrimSpace(part))
				for _, v := range subVars {
					vars[v] = true
				}
			}
			return ve.mapKeysToSlice(vars)
		}
	}

	// Extract dot notation variables
	if strings.Contains(expr, ".") {
		// Extract root variable (e.g., "filters" from "filters.active")
		parts := strings.Split(expr, ".")
		if len(parts) > 0 {
			rootVar := strings.TrimSpace(parts[0])
			if isValidVariableName(rootVar) {
				vars[rootVar] = true
			}
		}
	} else {
		// Simple variable name
		if isValidVariableName(expr) {
			vars[expr] = true
		}
	}

	return ve.mapKeysToSlice(vars)
}

// extractVariablesFromPath extracts variables from dot notation paths
func (ve *VariableExtractor) extractVariablesFromPath(path string) []string {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	if len(parts) > 0 {
		rootVar := strings.TrimSpace(parts[0])
		if isValidVariableName(rootVar) {
			return []string{rootVar}
		}
	}

	return nil
}

// isValidVariableName checks if a string is a valid variable name
func isValidVariableName(name string) bool {
	if name == "" {
		return false
	}

	// Simple validation - starts with letter or underscore, contains only alphanumeric and underscore
	first := name[0]
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}

	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}

	return true
}

// mapKeysToSlice converts map keys to sorted slice
func (ve *VariableExtractor) mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildDependencyGraph builds a dependency graph between variables
func (ve *VariableExtractor) buildDependencyGraph() map[string][]string {
	graph := make(map[string][]string)

	// For now, create a simple graph where structural variables depend on themselves
	// This can be enhanced with more sophisticated dependency analysis
	for structVar := range ve.structuralVars {
		graph[structVar] = []string{structVar}
	}

	return graph
}

// generateCacheKeyTemplate generates a template for cache key generation
func (ve *VariableExtractor) generateCacheKeyTemplate() string {
	if len(ve.structuralVars) == 0 {
		return "static" // No structural variables, SQL is static
	}

	// Create cache key template based on structural variables
	vars := ve.mapKeysToSlice(ve.structuralVars)
	return strings.Join(vars, ",")
}

// GenerateCacheKey generates a cache key for given parameter values
func (vd *VariableDependencies) GenerateCacheKey(params map[string]any) string {
	if vd.CacheKeyTemplate == "static" {
		return "static"
	}

	// Extract values for structural variables
	keyParts := make([]string, 0)
	for _, varName := range vd.StructuralVariables {
		value := extractValueFromParams(params, varName)
		keyParts = append(keyParts, fmt.Sprintf("%s=%v", varName, value))
	}

	// Generate hash of the key parts
	keyString := strings.Join(keyParts, "&")
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 characters
}

// extractValueFromParams extracts a value from parameters using dot notation
func extractValueFromParams(params map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = params

	for _, part := range parts {
		if currentMap, ok := current.(map[string]any); ok {
			current = currentMap[part]
		} else {
			return nil
		}
	}

	return current
}

// ToJSON converts the intermediate format to JSON
func (f *IntermediateFormat) ToJSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// FromJSON creates an intermediate format from JSON
func FromJSON(data []byte) (*IntermediateFormat, error) {
	var format IntermediateFormat
	err := json.Unmarshal(data, &format)
	return &format, err
}
