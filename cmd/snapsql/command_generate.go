package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func (g *GenerateCmd) Run(ctx *Context) error {
	// Load configuration
	config, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine input path
	inputPath := g.Input
	if inputPath == "" {
		inputPath = config.InputDir
	}

	// Merge constant files from config and command line
	constantFiles := append([]string{}, config.ConstantFiles...)
	constantFiles = append(constantFiles, g.Const...)

	if g.Lang != "json" {
		color.Blue("Generating %s files from %s", g.Lang, inputPath)
	} else {
		color.Blue("Generating files from %s", inputPath)
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
		color.Red("Failed to generate JSON intermediate files: %v", err)
		return err
	}

	if len(intermediateFiles) == 0 {
		color.Yellow("No intermediate files generated")
		return nil
	}

	// Generate files for all configured generators
	generatedLanguages := 0

	// JSON is always generated
	generatedLanguages++

	color.Blue("Generating files from %d intermediate files", len(intermediateFiles))

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
					color.Red("Failed to generate %s files: %v", lang, err)
					continue
				}
				generatedLanguages++
			}
		}
	}

	color.Green("Generation completed for %d language(s)", generatedLanguages)

	return nil
}

// generateForLanguage generates files for a specific language/generator
func generateForLanguage(lang string, generator GeneratorConfig, intermediateFiles []string, ctx *Context) error {

	switch lang {
	case "json":
		// JSON generation is handled in the main loop, nothing to do here
		return nil
	case "go":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-go"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		return nil
	case "typescript":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-typescript"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		return nil
	case "java":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-java"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
		}
		return nil
	case "python":
		// Use external plugin if available, otherwise show not implemented message
		if _, err := exec.LookPath("snapsql-gen-python"); err == nil {
			return generateWithExternalPlugin(lang, generator, intermediateFiles, ctx)
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
		fileData, err := os.ReadFile(intermediateFile)
		if err != nil {
			continue
		}

		intermediateData, err := intermediate.FromJSON(fileData)
		if err != nil {
			continue
		}

		// Convert to JSON for plugin input
		jsonData, err := intermediateData.ToJSON()
		if err != nil {
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
			continue
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
			// Use default config
			generator = GeneratorConfig{
				Output:   fmt.Sprintf("./generated/%s", g.Lang),
				Enabled:  true,
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
				Output:   fmt.Sprintf("./generated/%s", g.Lang),
				Enabled:  true,
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
		return nil, nil
	}

	// Process each file
	processedCount := 0
	generatedFiles := make([]string, 0, len(files))

	for _, file := range files {
		// Generate intermediate file
		outputFile, err := g.processTemplateFile(file, outputDir, constantFiles, config, ctx)
		if err != nil {
			if ctx.Verbose {
				color.Red("Failed to process %s: %v", file, err)
			}
			continue
		}

		generatedFiles = append(generatedFiles, outputFile)
		processedCount++

		if ctx.Verbose {
			color.Green("Generated: %s", outputFile)
		}
	}

	return generatedFiles, nil
}

// processTemplateFile processes a single template file and generates intermediate JSON
func (g *GenerateCmd) processTemplateFile(inputFile, outputDir string, constantFiles []string, config *Config, ctx *Context) (string, error) {
	// Load constants
	constants, err := g.loadConstants(config, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load constants: %w", err)
	}

	// Read the template file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Determine file type and process accordingly
	ext := strings.ToLower(filepath.Ext(inputFile))

	var format *intermediate.IntermediateFormat

	if ext == ".md" {
		// Process Markdown file
		doc, err := markdownparser.Parse(strings.NewReader(string(content)))
		if err != nil {
			return "", fmt.Errorf("failed to parse markdown: %w", err)
		}

		format, err = intermediate.GenerateFromMarkdown(doc, inputFile, ".", constants, nil, config)
		if err != nil {
			return "", fmt.Errorf("failed to generate from markdown: %w", err)
		}
	} else {
		// Process SQL file
		reader := strings.NewReader(string(content))
		format, err = intermediate.GenerateFromSQL(reader, constants, inputFile, ".", nil, config)
		if err != nil {
			return "", fmt.Errorf("failed to generate from SQL: %w", err)
		}
	}

	// Generate output filename
	outputFile := g.generateOutputFilename(inputFile, outputDir)

	// Write intermediate format to file
	outputData, err := format.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal intermediate format: %w", err)
	}

	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write intermediate file: %w", err)
	}

	color.Green("Generated: %s", outputFile)

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

// loadConstants loads constants from configuration and constant files
func (g *GenerateCmd) loadConstants(config *Config, ctx *Context) (map[string]any, error) {
	constants := make(map[string]any)

	// Load constants from files
	for _, file := range g.Const {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read constant file %s: %w", file, err)
		}

		var fileConstants map[string]any
		if err := yaml.Unmarshal(data, &fileConstants); err != nil {
			return nil, fmt.Errorf("failed to parse constant file %s: %w", file, err)
		}

		// Merge constants
		for k, v := range fileConstants {
			constants[k] = v
		}
	}

	return constants, nil
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

// ensureDir creates directory if it doesn't exist
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// isDirectory checks if path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
