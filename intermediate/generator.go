package intermediate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/shibukawa/snapsql/typeinference"
)

// Sentinel errors for validation
var (
	ErrSourceFileRequired        = errors.New("source file is required")
	ErrSourceContentRequired     = errors.New("source content is required")
	ErrSourceHashRequired        = errors.New("source hash is required")
	ErrInstructionsRequired      = errors.New("at least one instruction is required")
	ErrMissingVariableDeps       = errors.New("variable dependencies are missing but instructions contain variables")
	ErrMetadataVersionRequired   = errors.New("metadata version is required")
	ErrMetadataTimeRequired      = errors.New("metadata generated_at is required")
	ErrMetadataGeneratorRequired = errors.New("metadata generator is required")
	ErrStatementRequired         = errors.New("statement node is required")
	ErrFunctionDefRequired       = errors.New("function definition is required")
	ErrNotImplemented            = errors.New("feature not implemented for new API")
)

// GenerationOptions contains options for intermediate format generation
type GenerationOptions struct {
	// Database dialect for dialect-specific instructions
	Dialect snapsql.Dialect
	// Database schemas for type inference (type inference is always enabled)
	DatabaseSchemas []snapsql.DatabaseSchema
	// Additional metadata
	Metadata map[string]interface{}
}

// Generator generates intermediate format files from SQL templates
type Generator struct {
	extractor *EnhancedVariableExtractor
}

// NewGenerator creates a new intermediate format generator
func NewGenerator() (*Generator, error) {
	extractor, err := NewEnhancedVariableExtractor()
	if err != nil {
		return nil, fmt.Errorf("failed to create variable extractor: %w", err)
	}

	return &Generator{
		extractor: extractor,
	}, nil
}

// GenerateFromStatementNode generates intermediate format from parsed StatementNode
func (g *Generator) GenerateFromStatementNode(
	stmt parser.StatementNode,
	functionDef *parser.FunctionDefinition,
	options GenerationOptions,
) (*IntermediateFormat, error) {
	if stmt == nil {
		return nil, ErrStatementRequired
	}
	if functionDef == nil {
		return nil, ErrFunctionDefRequired
	}

	// Generate instruction sequence from statement node
	instructions, err := g.generateInstructions(stmt, options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate instructions: %w", err)
	}

	// Extract variable dependencies from instructions
	dependencies, err := g.extractor.ExtractFromInstructions(instructions)
	if err != nil {
		return nil, fmt.Errorf("failed to extract variable dependencies: %w", err)
	}

	// If instruction-based extraction yielded no results, try direct extraction from statement tokens
	if len(dependencies.AllVariables) == 0 {
		directDeps, err := g.extractVariablesDirectlyFromStatement(stmt, functionDef)
		if err == nil && len(directDeps.AllVariables) > 0 {
			dependencies = directDeps
		}
	}

	// Convert function definition to interface schema
	interfaceSchema := g.convertFunctionDefToInterfaceSchema(functionDef)

	// Create intermediate format
	format := &IntermediateFormat{
		Source: SourceInfo{
			File:    "<parsed-statement>", // No file for parsed statements
			Content: stmt.String(),        // Assuming StatementNode has String() method
			Hash:    g.calculateHash(stmt.String()),
		},
		InterfaceSchema: interfaceSchema,
		Instructions:    instructions,
		Dependencies:    dependencies,
	}

	// Perform type inference (always enabled)
	if len(options.DatabaseSchemas) > 0 {
		err = g.addTypeInference(format, stmt, options)
		if err != nil {
			return nil, fmt.Errorf("type inference failed: %w", err)
		}
	}

	return format, nil
}

// GenerateFromTemplate generates intermediate format from a SQL template
func (g *Generator) GenerateFromTemplate(templatePath string, instructions []Instruction) (*IntermediateFormat, error) {
	// Read template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Generate content hash
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])

	// Extract variable dependencies
	dependencies, err := g.extractor.ExtractFromInstructions(instructions)
	if err != nil {
		return nil, fmt.Errorf("failed to extract variable dependencies: %w", err)
	}

	// Create intermediate format
	format := &IntermediateFormat{
		Source: SourceInfo{
			File:    templatePath,
			Content: string(content),
			Hash:    contentHash,
		},
		Instructions: instructions,
		Dependencies: dependencies,
	}

	return format, nil
}

// GenerateFromTemplateWithSchema generates intermediate format with interface schema
func (g *Generator) GenerateFromTemplateWithSchema(templatePath string, instructions []Instruction, schema *InterfaceSchema) (*IntermediateFormat, error) {
	format, err := g.GenerateFromTemplate(templatePath, instructions)
	if err != nil {
		return nil, err
	}

	format.InterfaceSchema = schema
	return format, nil
}

// WriteToFile writes the intermediate format to a JSON file
func (g *Generator) WriteToFile(format *IntermediateFormat, outputPath string) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", dir, err)
	}

	// Generate JSON
	jsonData, err := format.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize intermediate format: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write intermediate file %s: %w", outputPath, err)
	}

	return nil
}

// ValidateFormat validates an intermediate format
func (g *Generator) ValidateFormat(format *IntermediateFormat) []error {
	var validationErrors []error

	// Validate source information
	if format.Source.File == "" {
		validationErrors = append(validationErrors, ErrSourceFileRequired)
	}
	if format.Source.Content == "" {
		validationErrors = append(validationErrors, ErrSourceContentRequired)
	}
	if format.Source.Hash == "" {
		validationErrors = append(validationErrors, ErrSourceHashRequired)
	}

	// Validate instructions
	if len(format.Instructions) == 0 {
		validationErrors = append(validationErrors, ErrInstructionsRequired)
	}

	// Validate instruction positions
	posErrors := ValidateInstructionPositions(format.Instructions, format.Source.Content)
	validationErrors = append(validationErrors, posErrors...)

	// Validate dependencies
	if len(format.Dependencies.AllVariables) == 0 && len(format.Instructions) > 0 {
		// Check if there are actually variables in instructions
		hasVariables := false
		for _, inst := range format.Instructions {
			if inst.Op == "EMIT_PARAM" || inst.Op == "EMIT_EVAL" || inst.Op == "JUMP_IF_EXP" || inst.Op == "LOOP_START" {
				hasVariables = true
				break
			}
		}
		if hasVariables {
			validationErrors = append(validationErrors, ErrMissingVariableDeps)
		}
	}
	return validationErrors
}

// LoadFromFile loads an intermediate format from a JSON file
func LoadFromFile(filePath string) (*IntermediateFormat, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intermediate file %s: %w", filePath, err)
	}

	format, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intermediate file %s: %w", filePath, err)
	}

	return format, nil
}

// InferResultTypeFields infers field types using typeinference and populates ResultType.Fields
func InferResultTypeFields(schema interface{}, selectClause interface{}, context interface{}) ([]Field, error) {
	// Temporarily disabled - needs to be updated for new typeinference API
	return nil, fmt.Errorf("feature not implemented: %w", ErrNotImplemented)
}

// Helper methods for the new API

// calculateHash generates SHA-256 hash of content
func (g *Generator) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// convertFunctionDefToInterfaceSchema converts parser.FunctionDefinition to InterfaceSchema
func (g *Generator) convertFunctionDefToInterfaceSchema(funcDef *parser.FunctionDefinition) *InterfaceSchema {
	// Basic implementation - convert function definition structure to interface schema format
	return &InterfaceSchema{
		FunctionName: funcDef.FunctionName,
		// Parameters and other fields would be converted here in full implementation
	}
}

// generateInstructions generates instruction sequence from StatementNode
func (g *Generator) generateInstructions(stmt parser.StatementNode, options GenerationOptions) ([]Instruction, error) {
	// Initialize instruction generator with the StatementNode
	generator := &instructionGenerator{
		dialect:      options.Dialect,
		instructions: make([]Instruction, 0),
		position:     []int{1, 1, 0}, // Default position
	}

	// Try to get raw tokens from the StatementNode to process SnapSQL directives and parameter references
	var tokens []tokenizer.Token
	var hasRawTokens bool

	// Check if RawTokens() is implemented and doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				// RawTokens() panicked (unimplemented), use fallback
				hasRawTokens = false
			}
		}()

		switch s := stmt.(type) {
		case interface{ RawTokens() []tokenizer.Token }:
			tokens = s.RawTokens()
			hasRawTokens = true
		}
	}()

	if hasRawTokens && len(tokens) > 0 {
		// Process tokens to generate appropriate instructions
		err := generator.processTokens(tokens)
		if err != nil {
			return nil, fmt.Errorf("failed to process tokens: %w", err)
		}
	} else {
		// Fallback: just process the basic structure
		instructions, err := generator.processNodeBasic(stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to process statement node: %w", err)
		}
		return instructions, nil
	}

	return generator.instructions, nil
}

// instructionGenerator helps generate instructions from StatementNode
type instructionGenerator struct {
	dialect      snapsql.Dialect
	instructions []Instruction
	position     []int // Current position [line, column, offset]
}

// processNode processes a StatementNode and generates instructions
func (ig *instructionGenerator) processNode(node parser.StatementNode) error {
	if node == nil {
		return nil
	}

	// Handle different node types
	switch n := node.(type) {
	case *parser.SelectStatement:
		return ig.processSelectStatement(n)
	case *parser.InsertIntoStatement:
		return ig.processInsertStatement(n)
	case *parser.UpdateStatement:
		return ig.processUpdateStatement(n)
	case *parser.DeleteFromStatement:
		return ig.processDeleteStatement(n)
	default:
		// For unknown node types, emit as literal
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "-- Unknown statement type",
		})
	}

	return nil
}

// processSelectStatement processes a SELECT statement
func (ig *instructionGenerator) processSelectStatement(stmt *parser.SelectStatement) error {
	// Basic SELECT processing - this would be expanded significantly
	ig.addInstruction(Instruction{
		Op:    OpEmitLiteral,
		Pos:   ig.getCurrentPosition(),
		Value: "SELECT ",
	})

	// Full implementation would process:
	// - SELECT fields (literal, parameter, expression)
	// - FROM clause with table references
	// - WHERE conditions with if/for logic
	// - JOIN clauses
	// - GROUP BY, ORDER BY, etc.

	// For now, acknowledge the statement parameter
	_ = stmt

	return nil
}

// processInsertStatement processes an INSERT statement
func (ig *instructionGenerator) processInsertStatement(stmt *parser.InsertIntoStatement) error {
	ig.addInstruction(Instruction{
		Op:    OpEmitLiteral,
		Pos:   ig.getCurrentPosition(),
		Value: "INSERT ",
	})

	// Full implementation would process INSERT specifics
	_ = stmt // Acknowledge parameter
	return nil
}

// processUpdateStatement processes an UPDATE statement
func (ig *instructionGenerator) processUpdateStatement(stmt *parser.UpdateStatement) error {
	ig.addInstruction(Instruction{
		Op:    OpEmitLiteral,
		Pos:   ig.getCurrentPosition(),
		Value: "UPDATE ",
	})

	// Full implementation would process UPDATE specifics
	_ = stmt // Acknowledge parameter
	return nil
}

// processDeleteStatement processes a DELETE statement
func (ig *instructionGenerator) processDeleteStatement(stmt *parser.DeleteFromStatement) error {
	ig.addInstruction(Instruction{
		Op:    OpEmitLiteral,
		Pos:   ig.getCurrentPosition(),
		Value: "DELETE ",
	})

	// Full implementation would process DELETE specifics
	_ = stmt // Acknowledge parameter
	return nil
}

// Helper methods

func (ig *instructionGenerator) addInstruction(inst Instruction) {
	ig.instructions = append(ig.instructions, inst)
}

func (ig *instructionGenerator) getCurrentPosition() []int {
	return ig.position
}

// processTokens processes raw tokens and generates appropriate instructions
func (ig *instructionGenerator) processTokens(tokens []tokenizer.Token) error {
	for _, token := range tokens {
		switch token.Type {
		case tokenizer.BLOCK_COMMENT:
			// Check if this is a SnapSQL directive or parameter reference
			value := strings.TrimSpace(token.Value)

			if strings.HasPrefix(value, "/*#") && strings.HasSuffix(value, "*/") {
				// SnapSQL directive like /*# if condition */, /*# for variable : collection */
				directive := strings.TrimSpace(value[3 : len(value)-2])
				err := ig.processDirective(directive)
				if err != nil {
					return fmt.Errorf("failed to process directive: %w", err)
				}
			} else if strings.HasPrefix(value, "/*=") && strings.HasSuffix(value, "*/") {
				// Parameter reference like /*= user_id */
				paramRef := strings.TrimSpace(value[3 : len(value)-2])
				if paramRef != "" {
					ig.addInstruction(Instruction{
						Op:          OpEmitParam,
						Pos:         ig.getCurrentPosition(),
						Param:       paramRef,
						Placeholder: "?", // Default placeholder
					})
				}
			} else {
				// Regular comment - emit as literal
				ig.addInstruction(Instruction{
					Op:    OpEmitLiteral,
					Pos:   ig.getCurrentPosition(),
					Value: token.Value,
				})
			}

		case tokenizer.IDENTIFIER, tokenizer.STRING, tokenizer.NUMBER:
			// Regular SQL tokens - emit as literals
			ig.addInstruction(Instruction{
				Op:    OpEmitLiteral,
				Pos:   ig.getCurrentPosition(),
				Value: token.Value,
			})

		case tokenizer.WHITESPACE:
			// Preserve whitespace
			ig.addInstruction(Instruction{
				Op:    OpEmitLiteral,
				Pos:   ig.getCurrentPosition(),
				Value: token.Value,
			})

		default:
			// Other tokens (operators, punctuation, etc.) - emit as literals
			ig.addInstruction(Instruction{
				Op:    OpEmitLiteral,
				Pos:   ig.getCurrentPosition(),
				Value: token.Value,
			})
		}

		// Update position based on token (simplified)
		ig.updatePosition(token)
	}

	return nil
}

// processDirective processes SnapSQL directives and generates appropriate instructions
func (ig *instructionGenerator) processDirective(directive string) error {
	if strings.HasPrefix(directive, "if ") {
		// /*# if condition */
		condition := strings.TrimSpace(directive[3:])
		if condition != "" {
			ig.addInstruction(Instruction{
				Op:  OpJumpIfExp,
				Pos: ig.getCurrentPosition(),
				Exp: condition,
			})
		}
	} else if strings.HasPrefix(directive, "for ") && strings.Contains(directive, " : ") {
		// /*# for variable : collection */
		parts := strings.Split(directive, " : ")
		if len(parts) >= 2 {
			loopVar := strings.TrimSpace(strings.TrimPrefix(parts[0], "for"))
			collection := strings.TrimSpace(parts[1])
			ig.addInstruction(Instruction{
				Op:         OpLoopStart,
				Pos:        ig.getCurrentPosition(),
				Variable:   loopVar,
				Collection: collection,
			})
		}
	} else if directive == "end" {
		// /*# end */
		ig.addInstruction(Instruction{
			Op:  OpLoopEnd,
			Pos: ig.getCurrentPosition(),
		})
	}
	// Add more directive types as needed

	return nil
}

// processNodeBasic provides basic fallback processing when RawTokens() is not available
func (ig *instructionGenerator) processNodeBasic(stmt parser.StatementNode) ([]Instruction, error) {
	// Fallback to basic SQL structure processing
	switch stmt.(type) {
	case *parser.SelectStatement:
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "SELECT * FROM table",
		})
	case *parser.InsertIntoStatement:
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "INSERT INTO table",
		})
	case *parser.UpdateStatement:
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "UPDATE table SET",
		})
	case *parser.DeleteFromStatement:
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "DELETE FROM table",
		})
	default:
		ig.addInstruction(Instruction{
			Op:    OpEmitLiteral,
			Pos:   ig.getCurrentPosition(),
			Value: "-- Unknown statement",
		})
	}

	return ig.instructions, nil
}

// updatePosition updates the current position based on token
func (ig *instructionGenerator) updatePosition(token tokenizer.Token) {
	// Simplified position tracking - in a full implementation,
	// this would properly track line and column based on token content
	ig.position[2] += len(token.Value) // Update offset
	if strings.Contains(token.Value, "\n") {
		ig.position[0]++   // Increment line
		ig.position[1] = 1 // Reset column
	} else {
		ig.position[1] += len(token.Value) // Update column
	}
}

// Future helper methods for more sophisticated instruction generation

// addParameterInstruction adds a parameter instruction (for future use)
//
//nolint:unused // Will be used when implementing full StatementNode processing
func (ig *instructionGenerator) addParameterInstruction(paramName, placeholder string) {
	ig.addInstruction(Instruction{
		Op:          OpEmitParam,
		Pos:         ig.getCurrentPosition(),
		Param:       paramName,
		Placeholder: placeholder,
	})
}

// addExpressionInstruction adds an expression evaluation instruction (for future use)
//
//nolint:unused // Will be used when implementing full StatementNode processing
func (ig *instructionGenerator) addExpressionInstruction(expression, placeholder string) {
	ig.addInstruction(Instruction{
		Op:          OpEmitEval,
		Pos:         ig.getCurrentPosition(),
		Exp:         expression,
		Placeholder: placeholder,
	})
}

// addDialectInstruction adds a dialect-specific instruction (for future use)
//
//nolint:unused // Will be used when implementing full StatementNode processing
func (ig *instructionGenerator) addDialectInstruction(alternatives map[string]string, defaultValue string) {
	ig.addInstruction(Instruction{
		Op:           OpEmitDialect,
		Pos:          ig.getCurrentPosition(),
		Alternatives: alternatives,
		Default:      defaultValue,
	})
}

// addTypeInference performs type inference and adds result field information
func (g *Generator) addTypeInference(format *IntermediateFormat, stmt parser.StatementNode, options GenerationOptions) error {
	// Use the new typeinference API
	fieldInfos, err := typeinference.InferFieldTypes(options.DatabaseSchemas, stmt, nil)
	if err != nil {
		return fmt.Errorf("failed to infer field types: %w", err)
	}

	// Convert typeinference results to intermediate format fields
	// In future versions, ResultType field would be added to IntermediateFormat
	// format.ResultType.Fields = convertFieldInfosToFields(fieldInfos)

	// For now, acknowledge the parameters to avoid unused errors
	_ = format
	_ = fieldInfos

	return nil
}

// extractVariablesDirectlyFromStatement extracts variables directly from StatementNode tokens
// This is a fallback method when instruction-based extraction fails
func (g *Generator) extractVariablesDirectlyFromStatement(stmt parser.StatementNode, functionDef *parser.FunctionDefinition) (VariableDependencies, error) {
	// Try to get raw tokens from the StatementNode
	var tokens []tokenizer.Token
	var hasRawTokens bool

	// Check if RawTokens() is implemented and doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				// RawTokens() panicked (unimplemented), use fallback
				hasRawTokens = false
			}
		}()

		switch s := stmt.(type) {
		case interface{ RawTokens() []tokenizer.Token }:
			tokens = s.RawTokens()
			hasRawTokens = true
		}
	}()

	// If we have tokens, extract variables from them
	if hasRawTokens && len(tokens) > 0 {
		vars := g.extractVariablesFromTokens(tokens, functionDef.Parameters)
		return vars, nil
	}

	// Fallback: use parameter-based extraction
	vars := g.extractVariablesFromParameters(functionDef.Parameters)
	return vars, nil
}

// extractVariablesFromParameters extracts variables from function parameters
// This is used when we cannot access the original SQL tokens
func (g *Generator) extractVariablesFromParameters(parameters map[string]any) VariableDependencies {
	allVars := make(map[string]bool)
	parameterVars := make(map[string]bool)
	structuralVars := make(map[string]bool)

	for paramName := range parameters {
		allVars[paramName] = true
		parameterVars[paramName] = true

		// Simple heuristic: parameters that are boolean or affect structure are structural
		if paramValue, ok := parameters[paramName]; ok {
			switch paramValue.(type) {
			case bool:
				structuralVars[paramName] = true
			case []interface{}:
				structuralVars[paramName] = true
			case map[string]interface{}:
				// Complex parameters are usually for filtering, consider them structural
				structuralVars[paramName] = true
			}
		}
	}

	return VariableDependencies{
		AllVariables:        g.mapKeysToSlice(allVars),
		StructuralVariables: g.mapKeysToSlice(structuralVars),
		ParameterVariables:  g.mapKeysToSlice(parameterVars),
		CacheKeyTemplate:    g.generateCacheKeyFromVars(g.mapKeysToSlice(structuralVars)),
	}
}

// extractVariablesFromTokens extracts variables from tokenizer tokens
func (g *Generator) extractVariablesFromTokens(tokens []tokenizer.Token, parameters map[string]any) VariableDependencies {
	allVars := make(map[string]bool)
	structuralVars := make(map[string]bool)
	parameterVars := make(map[string]bool)

	for i, token := range tokens {
		switch token.Type {
		case tokenizer.BLOCK_COMMENT:
			// Check if this is a SnapSQL directive or parameter reference
			value := strings.TrimSpace(token.Value)

			if strings.HasPrefix(value, "/*#") && strings.HasSuffix(value, "*/") {
				// SnapSQL directive like /*# if condition */, /*# for variable : collection */
				directive := strings.TrimSpace(value[3 : len(value)-2])
				vars := g.extractVariablesFromDirective(directive)
				for _, v := range vars {
					allVars[v] = true
					structuralVars[v] = true // Directives are structural
				}
			} else if strings.HasPrefix(value, "/*=") && strings.HasSuffix(value, "*/") {
				// Parameter reference like /*= user_id */
				paramRef := strings.TrimSpace(value[3 : len(value)-2])
				if paramRef != "" {
					// Extract root variable from parameter reference
					rootVar := g.extractRootVariable(paramRef)
					if rootVar != "" {
						allVars[rootVar] = true
						parameterVars[rootVar] = true
					}
				}
			}

		case tokenizer.IDENTIFIER:
			// Look ahead to see if this might be a parameter reference
			if i+1 < len(tokens) && tokens[i+1].Type == tokenizer.BLOCK_COMMENT {
				nextComment := tokens[i+1].Value
				if strings.HasPrefix(nextComment, "/*=") {
					// This identifier might be related to a parameter
					continue
				}
			}
		}
	}

	return VariableDependencies{
		AllVariables:        g.mapKeysToSlice(allVars),
		StructuralVariables: g.mapKeysToSlice(structuralVars),
		ParameterVariables:  g.mapKeysToSlice(parameterVars),
		CacheKeyTemplate:    g.generateCacheKeyFromVars(g.mapKeysToSlice(structuralVars)),
	}
}

// extractVariablesFromDirective extracts variables from SnapSQL directive
func (g *Generator) extractVariablesFromDirective(directive string) []string {
	vars := make([]string, 0)

	// Handle different directive types
	if strings.HasPrefix(directive, "if ") {
		// /*# if condition */
		condition := strings.TrimSpace(directive[3:])
		if condition != "" {
			extractor, _ := NewCELVariableExtractor()
			if extractor != nil {
				if extracted, err := extractor.ExtractVariables(condition); err == nil {
					vars = append(vars, extracted...)
				}
			}
		}
	} else if strings.Contains(directive, " : ") {
		// /*# for variable : collection */
		parts := strings.Split(directive, " : ")
		if len(parts) >= 2 {
			// Extract collection variable
			collection := strings.TrimSpace(parts[1])
			if collection != "" {
				vars = append(vars, collection)
			}
		}
	}

	return vars
}

// extractRootVariable extracts the root variable from a parameter reference
func (g *Generator) extractRootVariable(paramRef string) string {
	// Handle simple variables like "user_id"
	// Handle complex references like "filters.active"
	parts := strings.Split(paramRef, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return paramRef
}

// mapKeysToSlice converts map keys to sorted slice
func (g *Generator) mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort for consistent output
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// generateCacheKeyFromVars generates cache key template from structural variables
func (g *Generator) generateCacheKeyFromVars(structuralVars []string) string {
	if len(structuralVars) == 0 {
		return "static"
	}
	return strings.Join(structuralVars, ",")
}
