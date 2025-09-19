package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/langs/gogen"
	"github.com/shibukawa/snapsql/markdownparser"
)

// GenerateCmd represents the generate command
type GenerateCmd struct {
	Input    string   `short:"i" help:"Input file or directory" type:"path"`
	Lang     string   `help:"Output language/format"`
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

	if g.Lang != "" {
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
	// Generate files for all enabled generators
	generatedLanguages := 0

	var (
		intermediateFiles []string
		err               error
	)

	// Generate all enabled generators

	for lang, generator := range config.Generation.Generators {
		if !generator.Enabled {
			continue
		}

		// For non-JSON languages, we need intermediate files first
		if lang != "json" {
			if intermediateFiles == nil {
				intermediateFiles, err = g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
				if err != nil {
					color.Red("Failed to generate intermediate files: %v", err)
					return err
				}

				if len(intermediateFiles) == 0 {
					color.Yellow("No intermediate files generated")
					return nil
				}
			}

			// Generate other language files
			err = generateForLanguage(lang, generator, intermediateFiles, ctx)
			if err != nil {
				color.Red("Failed to generate %s files: %v", lang, err)
				continue
			}
		} else {
			// Generate JSON intermediate files
			_, err = g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
			if err != nil {
				color.Red("Failed to generate JSON files: %v", err)
				continue
			}
		}

		generatedLanguages++
	}

	if generatedLanguages == 0 {
		color.Yellow("No generators are enabled in configuration")
		return nil
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
		// Use built-in Go generator
		return generateGoFiles(generator, intermediateFiles, ctx)
	case "typescript":
		// Use external plugin if available, otherwise show not implemented message
		_, err := exec.LookPath("snapsql-gen-typescript")
		if err == nil {
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

// generateGoFiles generates Go files using the built-in generator
func generateGoFiles(generator GeneratorConfig, intermediateFiles []string, ctx *Context) error {
	// Load config to get dialect
	config, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Import the Go generator
	goGen := &gogen.Generator{}

	// Configure the generator
	if packageName, ok := generator.Settings["package"].(string); ok {
		goGen.PackageName = packageName
	} else {
		// Infer package name from output path
		outputPath := generator.Output
		if outputPath == "" {
			outputPath = "./generated/go"
		}

		goGen.PackageName = gogen.InferPackageNameFromPath(outputPath)
	}

	// Process each intermediate file
	for _, intermediateFile := range intermediateFiles {
		// Read intermediate format
		data, err := os.ReadFile(intermediateFile)
		if err != nil {
			return fmt.Errorf("failed to read intermediate file %s: %w", intermediateFile, err)
		}

		var format intermediate.IntermediateFormat
		if err := json.Unmarshal(data, &format); err != nil {
			return fmt.Errorf("failed to parse intermediate file %s: %w", intermediateFile, err)
		}

		// Set format and dialect
		goGen.Format = &format
		goGen.Dialect = config.Dialect

		// Generate Go code
		var output strings.Builder
		if err := goGen.Generate(&output); err != nil {
			return fmt.Errorf("failed to generate Go code for %s: %w", intermediateFile, err)
		}

		// Determine output file path
		outputDir := generator.Output
		if outputDir == "" {
			outputDir = "./generated/go"
		}

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}

		// Generate output file name
		baseName := strings.TrimSuffix(filepath.Base(intermediateFile), ".json")
		outputFile := filepath.Join(outputDir, baseName+".go")

		// Write Go code to file
		if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
			return fmt.Errorf("failed to write Go file %s: %w", outputFile, err)
		}

		if ctx.Verbose {
			color.Green("Generated: %s", outputFile)
		}
	}

	return nil
}

// generateWithExternalPlugin attempts to use an external generator plugin
func generateWithExternalPlugin(lang string, generator GeneratorConfig, intermediateFiles []string, ctx *Context) error {
	_ = ctx // Context not currently used for external plugin generation
	pluginName := "snapsql-gen-" + lang

	// Check if plugin exists in PATH
	if _, err := exec.LookPath(pluginName); err != nil {
		return fmt.Errorf("%w: '%s'", ErrPluginNotFound, pluginName)
	}

	// Prepare output directory
	outputDir := generator.Output
	if outputDir == "" {
		outputDir = "./generated/" + lang
	}

	err := ensureDir(outputDir)
	if err != nil {
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
				strValue = strconv.FormatBool(v)
			case int, int64, float64:
				strValue = fmt.Sprintf("%v", v)
			default:
				// Skip complex types
				continue
			}

			args = append(args, "--"+key, strValue)
		}

		// Execute plugin
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, pluginName, args...)
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
				Output:   "./generated/" + g.Lang,
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
				Output:   "./generated/" + g.Lang,
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
	var (
		files []string
		err   error
	)

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
		outputFile, err := g.processTemplateFile(file, outputDir, inputPath, constantFiles, config, ctx)
		if err != nil {
			if ctx.Verbose {
				color.Red("Failed to process %s: %v", file, err)
			}

			continue
		}

		generatedFiles = append(generatedFiles, outputFile)
		processedCount++
		// Output message is handled in processTemplateFile
	}

	return generatedFiles, nil
}

// processTemplateFile processes a single template file and generates intermediate JSON
func (g *GenerateCmd) processTemplateFile(inputFile, outputDir, inputDir string, constantFiles []string, config *Config, ctx *Context) (string, error) {
	_ = constantFiles // Constant files are loaded through config, not directly used here
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
	jsonGen := config.Generation.Generators["json"]
	outputFile := g.generateOutputFilename(inputFile, outputDir, inputDir, jsonGen.PreserveHierarchy)

	// Ensure output directory exists (including subdirectories if preserving hierarchy)
	outputFileDir := filepath.Dir(outputFile)
	if err := ensureDir(outputFileDir); err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outputFileDir, err)
	}

	// Write intermediate format to file
	outputData, err := format.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal intermediate format: %w", err)
	}

	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write intermediate file: %w", err)
	}

	// Only show output message if verbose mode is enabled
	if ctx.Verbose {
		color.Green("Generated: %s", outputFile)
	}

	return outputFile, nil
}

// loadConstants loads constants from configuration and constant files
func (g *GenerateCmd) loadConstants(config *Config, ctx *Context) (map[string]any, error) {
	_ = config // Config not currently used for constant loading
	_ = ctx    // Context not currently used for constant loading
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
func (g *GenerateCmd) generateOutputFilename(inputFile, outputDir, inputDir string, preserveHierarchy bool) string {
	// Get base filename without extension
	base := filepath.Base(inputFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Remove .snap suffix if present
	name = strings.TrimSuffix(name, ".snap")

	if preserveHierarchy {
		// Calculate relative path from input directory
		relPath, err := filepath.Rel(inputDir, inputFile)
		if err != nil {
			// Fallback to flat structure if relative path calculation fails
			return filepath.Join(outputDir, name+".json")
		}

		// Get directory part of the relative path
		relDir := filepath.Dir(relPath)
		if relDir == "." {
			// File is in the root input directory
			return filepath.Join(outputDir, name+".json")
		}

		// Create subdirectory structure in output
		outputSubDir := filepath.Join(outputDir, relDir)

		return filepath.Join(outputSubDir, name+".json")
	}

	// Flat structure (original behavior)
	return filepath.Join(outputDir, name+".json")
}

// findTemplateFiles finds all SQL template files in the input directory recursively
func findTemplateFiles(inputDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check for .snap.sql or .snap.md files
		fileName := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(fileName, ".snap.sql") || strings.HasSuffix(fileName, ".snap.md") {
			files = append(files, path)
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
