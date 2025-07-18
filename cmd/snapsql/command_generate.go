package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Error definitions
var (
	ErrGeneratorNotConfigured = errors.New("generator not configured")
	ErrPluginNotFound        = errors.New("generator plugin not found")
	ErrInputFileNotExist     = errors.New("input file does not exist")
	ErrNoASTGenerated        = errors.New("no AST generated")
)

// GenerateCmd represents the generate command
type GenerateCmd struct {
	Input    string   `short:"i" help:"Input file or directory" type:"path"`
	Lang     string   `help:"Output language/format" default:"json"`
	Package  string   `help:"Package name (language-specific)"`
	Const    []string `help:"Constant definition files"`
	Validate bool     `help:"Validate templates before generation"`
	Watch    bool     `help:"Watch for file changes and regenerate automatically"`
}

// ParseResult holds the result of parsing a template file
type ParseResult struct {
	Schema *parser.FunctionDefinition
	AST    parser.AstNode
}

func (g *GenerateCmd) Run(ctx *Context) error {
	// Load configuration
	config, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine input path
	inputPath := g.Input
	if inputPath == "" {
		inputPath = config.Generation.InputDir
	}

	// Merge constant files from config and command line
	constantFiles := append([]string{}, config.ConstantFiles...)
	constantFiles = append(constantFiles, g.Const...)

	if ctx.Verbose {
		if g.Lang != "json" {
			color.Blue("Generating %s files from %s", g.Lang, inputPath)
		} else {
			color.Blue("Generating files from %s", inputPath)
		}
	}

	// If specific language is requested, generate only that
	if g.Lang != "" {
		return g.generateSpecificLanguage(ctx, config, inputPath, constantFiles)
	}

	// Generate all configured languages
	return g.generateAllLanguages(ctx, config, inputPath, constantFiles)
}

// generateAllLanguages generates files for all configured languages
func (g *GenerateCmd) generateAllLanguages(ctx *Context, config *Config, inputPath string, constantFiles []string) error {
	// Generate JSON intermediate files (always generated)
	intermediateFiles, err := g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
	if err != nil {
		if !ctx.Quiet {
			color.Red("Failed to generate JSON intermediate files: %v", err)
		}
		return err
	}

	if len(intermediateFiles) == 0 {
		if !ctx.Quiet {
			color.Yellow("No intermediate files generated")
		}
		return nil
	}

	// Generate files for all configured generators
	generatedLanguages := 0

	// JSON is always generated
	generatedLanguages++
	
	if ctx.Verbose {
		color.Blue("Generating files from %d intermediate files", len(intermediateFiles))
	}

	// If specific language is requested, generate only that language
	if g.Lang != "" && g.Lang != "json" {
		if generator, exists := config.Generation.Generators[g.Lang]; exists && generator.Enabled {
			if err := generateForLanguage(g.Lang, generator, intermediateFiles, ctx); err != nil {
				return fmt.Errorf("failed to generate %s files: %w", g.Lang, err)
			}
			generatedLanguages++
		} else {
			return fmt.Errorf("%w: '%s'", ErrGeneratorNotConfigured, g.Lang)
		}
	} else if g.Lang == "" {
		// Generate all other enabled generators
		for lang, generator := range config.Generation.Generators {
			if lang != "json" && generator.Enabled {
				if err := generateForLanguage(lang, generator, intermediateFiles, ctx); err != nil {
					if !ctx.Quiet {
						color.Red("Failed to generate %s files: %v", lang, err)
					}
					continue
				}
				generatedLanguages++
			}
		}
	}

	if !ctx.Quiet {
		color.Green("Generation completed for %d language(s)", generatedLanguages)
	}

	return nil
}

// generateForLanguage generates files for a specific language/generator
func generateForLanguage(lang string, generator GeneratorConfig, intermediateFiles []string, ctx *Context) error {
	if ctx.Verbose {
		color.Blue("Generating %s files", lang)
	}

	switch lang {
	case "json":
		// JSON generation is handled in the main loop, nothing to do here
		return nil
	case "go":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-go"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		if !ctx.Quiet {
			color.Yellow("Go generation not yet implemented")
		}
		return nil
	case "typescript":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-typescript"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		if !ctx.Quiet {
			color.Yellow("TypeScript generation not yet implemented")
		}
		return nil
	case "java":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-java"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		if !ctx.Quiet {
			color.Yellow("Java generation not yet implemented")
		}
		return nil
	case "python":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-python"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		if !ctx.Quiet {
			color.Yellow("Python generation not yet implemented")
		}
		return nil
	default:
		// Try to find external generator plugin
		return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
	}
}

// generateWithExternalPlugin attempts to use an external generator plugin
func generateWithExternalPlugin(lang string, generator GeneratorConfig, intermediateFiles []string, ctx *Context) error {
	pluginName := fmt.Sprintf("snapsql-gen-%s", lang)

	// Check if plugin exists in PATH
	if _, err := exec.LookPath(pluginName); err != nil {
		return fmt.Errorf("%w: '%s'", ErrPluginNotFound, pluginName)
	}

	if ctx.Verbose {
		color.Blue("Using external generator plugin: %s", pluginName)
	}

	// Prepare output directory
	outputDir := generator.Output
	if outputDir == "" {
		outputDir = fmt.Sprintf("./generated/%s", lang)
	}
	
	if err := ensureDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Process each intermediate file
	for _, intermediateFile := range intermediateFiles {
		// Load intermediate file
		intermediateData, err := intermediate.LoadFromFile(intermediateFile)
		if err != nil {
			if ctx.Verbose {
				color.Yellow("Failed to load intermediate file %s: %v", intermediateFile, err)
			}
			continue
		}

		// Convert to JSON for plugin input
		jsonData, err := intermediateData.ToJSON()
		if err != nil {
			if ctx.Verbose {
				color.Yellow("Failed to serialize intermediate data: %v", err)
			}
			continue
		}

		// Prepare command arguments
		args := []string{
			"--output", outputDir,
		}
		
		// Add settings as command line arguments
		for key, value := range generator.Settings {
			// Convert setting value to string
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case bool:
				strValue = fmt.Sprintf("%t", v)
			case int, int64, float64:
				strValue = fmt.Sprintf("%v", v)
			default:
				// Skip complex types
				continue
			}
			
			args = append(args, fmt.Sprintf("--%s", key), strValue)
		}

		// Execute plugin
		cmd := exec.Command(pluginName, args...)
		cmd.Stdin = bytes.NewReader(jsonData)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if ctx.Verbose {
				color.Yellow("Generator plugin execution failed: %v", err)
			}
			continue
		}

		if ctx.Verbose {
			color.Green("Generated %s code from %s", lang, intermediateFile)
		}
	}

	return nil
}

// generateSpecificLanguage generates files for a specific language
func (g *GenerateCmd) generateSpecificLanguage(ctx *Context, config *Config, inputPath string, constantFiles []string) error {
	switch g.Lang {
	case "json":
		// Just generate intermediate files
		_, err := g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
		return err
	case "go", "typescript", "java", "python":
		// Generate intermediate files first
		intermediateFiles, err := g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
		if err != nil {
			return err
		}
		
		// Find generator config
		generator, exists := config.Generation.Generators[g.Lang]
		if !exists || !generator.Enabled {
			if !ctx.Quiet {
				color.Yellow("%s generation not configured, using default settings", g.Lang)
			}
			// Use default config
			generator = GeneratorConfig{
				Output:  fmt.Sprintf("./generated/%s", g.Lang),
				Enabled: true,
				Settings: map[string]any{},
			}
		}
		
		// If package name is specified, add it to settings
		if g.Package != "" {
			if generator.Settings == nil {
				generator.Settings = make(map[string]any)
			}
			generator.Settings["package"] = g.Package
		}
		
		// Generate code
		return generateForLanguage(g.Lang, generator, intermediateFiles, ctx)
	default:
		// Custom language - look for external generator
		intermediateFiles, err := g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
		if err != nil {
			return err
		}
		
		// Find generator config
		generator, exists := config.Generation.Generators[g.Lang]
		if !exists {
			// Use default config
			generator = GeneratorConfig{
				Output:  fmt.Sprintf("./generated/%s", g.Lang),
				Enabled: true,
				Settings: map[string]any{},
			}
		}
		
		// Generate code using external plugin
		return generateWithExternalPlugin(g.Lang, generator, intermediateFiles, ctx)
	}
}

// generateIntermediateFiles generates JSON intermediate files from SQL templates
func (g *GenerateCmd) generateIntermediateFiles(ctx *Context, config *Config, inputPath string, constantFiles []string) ([]string, error) {
	// Determine output directory from JSON generator configuration
	outputDir := "./generated"
	if jsonGen, exists := config.Generation.Generators["json"]; exists && jsonGen.Output != "" {
		outputDir = jsonGen.Output
	}

	// Ensure output directory exists
	if err := ensureDir(outputDir); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Check if input is a file or directory
	var files []string
	var err error

	if isDirectory(inputPath) {
		// Find all SQL template files in directory
		files, err = findTemplateFiles(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to find template files: %w", err)
		}
	} else {
		// Single file input
		if !fileExists(inputPath) {
			return nil, fmt.Errorf("%w: %s", ErrInputFileNotExist, inputPath)
		}
		files = []string{inputPath}
	}

	if len(files) == 0 {
		if !ctx.Quiet {
			color.Yellow("No template files found in %s", inputPath)
		}
		return nil, nil
	}

	if ctx.Verbose {
		color.Blue("Found %d template files", len(files))
	}

	// Process each file
	processedCount := 0
	generatedFiles := make([]string, 0, len(files))
	
	for _, file := range files {
		if ctx.Verbose {
			color.Blue("Processing: %s", file)
		}

		// Generate intermediate file
		outputFile, err := g.processTemplateFile(file, outputDir, constantFiles, config, ctx)
		if err != nil {
			if !ctx.Quiet {
				color.Red("Failed to process %s: %v", file, err)
			}
			continue
		}

		generatedFiles = append(generatedFiles, outputFile)
		processedCount++
	}

	if !ctx.Quiet {
		color.Green("Generated %d intermediate files in %s", processedCount, outputDir)
	}

	return generatedFiles, nil
}

// processTemplateFile processes a single template file and generates intermediate JSON
func (g *GenerateCmd) processTemplateFile(inputFile, outputDir string, constantFiles []string, _ *Config, ctx *Context) (string, error) {
	// Read the template file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Determine file type and process accordingly
	ext := strings.ToLower(filepath.Ext(inputFile))

	var parseResult *ParseResult

	if ext == ".md" {
		// Process Markdown file
		parseResult, err = g.parseMarkdownFile(string(content), constantFiles, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to parse markdown file: %w", err)
		}
	} else {
		// Process SQL file
		parseResult, err = g.parseSQLFile(string(content), constantFiles, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to parse SQL file: %w", err)
		}
	}

	// Validate parse result if requested
	if err := g.validateParseResult(parseResult, ctx); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// Create a minimal intermediate format
	format := &intermediate.IntermediateFormat{
		Source: intermediate.SourceInfo{
			File:    inputFile,
			Content: string(content),
			Hash:    calculateHash(string(content)),
		},
	}
	
	// Add schema if available
	if parseResult.Schema != nil {
		format.InterfaceSchema = g.convertSchemaToIntermediate(parseResult.Schema)
	}
	
	// Add dependencies
	format.Dependencies = intermediate.VariableDependencies{
		AllVariables:        extractVariableNames(parseResult.Schema),
		StructuralVariables: []string{},
		ParameterVariables:  extractVariableNames(parseResult.Schema),
		CacheKeyTemplate:    "static",
	}
	
	// Add simple instruction
	format.Instructions = []intermediate.Instruction{
		{
			Op:    intermediate.OpEmitLiteral,
			Pos:   []int{1, 1, 0},
			Value: string(content), // Use original SQL content
		},
	}

	// Generate output filename
	outputFile := g.generateOutputFilename(inputFile, outputDir)

	// Create intermediate format generator
	generator, err := intermediate.NewGenerator()
	if err != nil {
		return "", fmt.Errorf("failed to create intermediate generator: %w", err)
	}

	// Write intermediate format to file
	if err := generator.WriteToFile(format, outputFile); err != nil {
		return "", fmt.Errorf("failed to write intermediate file: %w", err)
	}

	if ctx.Verbose {
		color.Green("Generated: %s", outputFile)
	}

	return outputFile, nil
}

// extractVariableNames extracts variable names from function definition
func extractVariableNames(schema *parser.FunctionDefinition) []string {
	if schema == nil || len(schema.Parameters) == 0 {
		return []string{}
	}
	
	names := make([]string, 0, len(schema.Parameters))
	for name := range schema.Parameters {
		names = append(names, name)
	}
	
	sort.Strings(names)
	return names
}

// calculateHash generates SHA-256 hash of content
func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// parseSQLFile parses SQL content and returns parse results
func (g *GenerateCmd) parseSQLFile(content string, constantFiles []string, ctx *Context) (*ParseResult, error) {
	// Create tokenizer
	tokens, err := tokenizer.Tokenize(content)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Extract function definition from SQL tokens
	functionDef, err := parser.NewFunctionDefinitionFromSQL(tokens)
	if err != nil {
		if ctx.Verbose {
			color.Yellow("Failed to extract function definition from SQL: %v", err)
		}
		// Create default schema if extraction fails
		functionDef = &parser.FunctionDefinition{
			Name:         "query", // Default name
			FunctionName: "executeQuery",
			Parameters:   make(map[string]any),
		}
	} else if ctx.Verbose {
		color.Blue("Extracted function definition from SQL")
	}

	// Parse SQL
	ast, err := parser.Parse(tokens, functionDef, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return &ParseResult{
		Schema: functionDef,
		AST:    ast,
	}, nil
}

// parseMarkdownFile processes a markdown file and returns parse results
func (g *GenerateCmd) parseMarkdownFile(content string, constantFiles []string, ctx *Context) (*ParseResult, error) {
	// Parse markdown content using the correct API
	reader := strings.NewReader(content)
	parsed, err := markdownparser.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Create interface schema from metadata and parameters
	schema := &parser.FunctionDefinition{
		Name:         "query", // Default name
		FunctionName: "executeQuery",
		Parameters:   make(map[string]any),
	}

	// Use metadata from parsed document
	if parsed.Metadata != nil {
		if name, ok := parsed.Metadata["name"].(string); ok {
			schema.Name = name
		}
		if funcName, ok := parsed.Metadata["function_name"].(string); ok {
			schema.FunctionName = funcName
		}
	}

	// Parse parameters from parameter block
	if parsed.ParameterBlock != "" {
		params := make(map[string]any)
		if err := yaml.Unmarshal([]byte(parsed.ParameterBlock), &params); err == nil {
			// Add parameters to schema
			for name, paramType := range params {
				schema.Parameters[name] = convertParameterType(paramType)
			}
		}
	}

	var ast parser.AstNode

	// Extract and parse SQL from markdown
	if parsed.SQL != "" {
		sqlContent := strings.TrimSpace(parsed.SQL)
		if sqlContent != "" {
			// Tokenize SQL
			tokens, err := tokenizer.Tokenize(sqlContent)
			if err != nil {
				return nil, fmt.Errorf("tokenization failed: %w", err)
			}

			// Parse SQL using the correct API
			ast, err = parser.Parse(tokens, schema, nil)
			if err != nil {
				return nil, fmt.Errorf("parsing failed: %w", err)
			}
		}
	}

	return &ParseResult{
		Schema: schema,
		AST:    ast,
	}, nil
}

// convertSchemaToIntermediate converts parser schema to intermediate schema
func (g *GenerateCmd) convertSchemaToIntermediate(schema *parser.FunctionDefinition) *intermediate.InterfaceSchema {
	if schema == nil {
		return nil
	}

	// Convert parameters
	parameters := make([]intermediate.Parameter, 0, len(schema.Parameters))
	for name, paramType := range schema.Parameters {
		// Determine parameter type
		typeStr := "any"
		optional := false
		
		switch v := paramType.(type) {
		case string:
			typeStr = v
		case bool, int, float64:
			typeStr = fmt.Sprintf("%T", v)
		case map[string]any:
			typeStr = "object"
		case []any:
			typeStr = "array"
		}
		
		param := intermediate.Parameter{
			Name:     name,
			Type:     typeStr,
			Optional: optional,
		}
		parameters = append(parameters, param)
	}

	// Sort parameters for consistent output
	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Name < parameters[j].Name
	})

	return &intermediate.InterfaceSchema{
		Name:         schema.Name,
		FunctionName: schema.FunctionName,
		Parameters:   parameters,
	}
}

func (g *GenerateCmd) validateParseResult(result *ParseResult, ctx *Context) error {
	if !g.Validate {
		return nil
	}

	// Perform validation checks
	if result.AST == nil {
		return ErrNoASTGenerated
	}

	if ctx.Verbose {
		color.Green("Validation passed")
	}

	return nil
}

// generateOutputFilename generates output filename for intermediate JSON
func (g *GenerateCmd) generateOutputFilename(inputFile, outputDir string) string {
	// Get base filename without extension
	base := filepath.Base(inputFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Remove .snap suffix if present
	name = strings.TrimSuffix(name, ".snap")

	// Add .json extension
	return filepath.Join(outputDir, name+".json")
}

// findTemplateFiles finds all SQL template files in the input directory
func findTemplateFiles(inputDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check for supported extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".sql" || ext == ".md" {
			// Check for .snap. prefix
			if strings.Contains(strings.ToLower(filepath.Base(path)), ".snap.") {
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}

// loadConstantFiles loads constant definition files
func loadConstantFiles(constants map[string]any, files []string) error {
	for _, file := range files {
		// Read constant file
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read constant file %s: %w", file, err)
		}

		// Parse YAML
		var fileConstants map[string]any
		if err := yaml.Unmarshal(data, &fileConstants); err != nil {
			return fmt.Errorf("failed to parse constant file %s: %w", file, err)
		}

		// Merge constants
		for k, v := range fileConstants {
			constants[k] = v
		}
	}
	return nil
}

// generateFunctionName converts a space-separated name to a function name
func generateFunctionName(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return ""
	}

	// Convert to PascalCase for function names
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// convertParameterType converts a parameter type from markdown to intermediate format
func convertParameterType(paramType any) string {
	switch v := paramType.(type) {
	case string:
		return v
	case map[string]any:
		return "object"
	case []any:
		if len(v) > 0 {
			return "array"
		}
		return "array"
	default:
		return "unknown"
	}
}
