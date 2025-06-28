package intermediate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
)

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
		Metadata: FormatMetadata{
			Version:     "2.0.0",
			GeneratedAt: time.Now().Format(time.RFC3339),
			Generator:   "snapsql-intermediate-generator",
			SchemaURL:   "https://github.com/shibukawa/snapsql/schemas/intermediate-format.json",
		},
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

	// Validate metadata
	if format.Metadata.Version == "" {
		validationErrors = append(validationErrors, ErrMetadataVersionRequired)
	}
	if format.Metadata.GeneratedAt == "" {
		validationErrors = append(validationErrors, ErrMetadataTimeRequired)
	}
	if format.Metadata.Generator == "" {
		validationErrors = append(validationErrors, ErrMetadataGeneratorRequired)
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
func InferResultTypeFields(schema *typeinference.SchemaStore, selectClause *typeinference.SelectClause, context *typeinference.InferenceContext) ([]Field, error) {
	tie := typeinference.NewTypeInferenceEngine(schema)
	inferred, err := tie.InferSelectTypes(selectClause, context)
	if err != nil {
		return nil, err
	}
	fields := make([]Field, len(inferred))
	for i, f := range inferred {
		fields[i] = Field{
			Name:       f.Name,
			Type:       f.Type.BaseType,
			BaseType:   f.Type.BaseType,
			IsNullable: f.Type.IsNullable,
			MaxLength:  f.Type.MaxLength,
			Precision:  f.Type.Precision,
			Scale:      f.Type.Scale,
		}
	}
	return fields, nil
}
