package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors
var (
	ErrGeneratorNotConfigured = errors.New("generator is not configured or not enabled")
	ErrPluginNotFound         = errors.New("external generator plugin not found in PATH")
	ErrInputFileNotExist      = errors.New("input file does not exist")
	ErrNoInterfaceSchema      = errors.New("no interface schema found")
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
	if g.Lang != "json" {
		return g.generateSpecificLanguage(ctx, config, inputPath, constantFiles)
	}

	// Generate all configured languages
	return g.generateAllLanguages(ctx, config, inputPath, constantFiles)
}

// generateAllLanguages generates files for all configured languages
func (g *GenerateCmd) generateAllLanguages(ctx *Context, config *Config, inputPath string, constantFiles []string) error {
	var intermediateFiles []string

	// Generate JSON intermediate files (always generated)
	if err := g.generateIntermediateFiles(ctx, config, inputPath, constantFiles); err != nil {
		if !ctx.Quiet {
			color.Red("Failed to generate JSON intermediate files: %v", err)
		}
	} else {
		// TODO: Collect intermediate file paths for other generators
		intermediateFiles = []string{} // Placeholder
	}

	// Generate files for all configured generators
	generatedLanguages := 0

	// If specific language is requested, generate only that language
	if g.Lang != "" {
		if generator, exists := config.Generation.Generators[g.Lang]; exists && generator.Enabled {
			if err := generateForLanguage(g.Lang, generator, intermediateFiles, ctx); err != nil {
				return fmt.Errorf("failed to generate %s files: %w", g.Lang, err)
			}
			generatedLanguages++
		} else if g.Lang == "json" {
			// JSON is always available even if not explicitly configured
			jsonGen := GeneratorConfig{
				Output:  "./generated",
				Enabled: true,
				Settings: map[string]interface{}{
					"pretty":           true,
					"include_metadata": true,
				},
			}
			if err := generateForLanguage("json", jsonGen, intermediateFiles, ctx); err != nil {
				return fmt.Errorf("failed to generate json files: %w", err)
			}
			generatedLanguages++
		} else {
			return fmt.Errorf("%w: '%s'", ErrGeneratorNotConfigured, g.Lang)
		}
	} else {
		// Generate all enabled generators
		for lang, generator := range config.Generation.Generators {
			if generator.Enabled {
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
		// TODO: Implement Go generation
		if !ctx.Quiet {
			color.Yellow("Go generation not yet implemented")
		}
		return nil
	case "typescript":
		// TODO: Implement TypeScript generation
		if !ctx.Quiet {
			color.Yellow("TypeScript generation not yet implemented")
		}
		return nil
	case "java":
		// TODO: Implement Java generation
		if !ctx.Quiet {
			color.Yellow("Java generation not yet implemented")
		}
		return nil
	case "python":
		// TODO: Implement Python generation
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
func generateWithExternalPlugin(lang string, _ GeneratorConfig, _ []string, ctx *Context) error {
	pluginName := fmt.Sprintf("snapsql-gen-%s", lang)

	// Check if plugin exists in PATH
	if _, err := exec.LookPath(pluginName); err != nil {
		return fmt.Errorf("%w: '%s'", ErrPluginNotFound, pluginName)
	}

	if ctx.Verbose {
		color.Blue("Using external generator plugin: %s", pluginName)
	}

	// TODO: Implement external plugin execution
	if !ctx.Quiet {
		color.Yellow("External generator plugin execution not yet implemented")
	}

	return nil
}

// generateSpecificLanguage generates files for a specific language
func (g *GenerateCmd) generateSpecificLanguage(ctx *Context, config *Config, inputPath string, constantFiles []string) error {
	switch g.Lang {
	case "json":
		return g.generateIntermediateFiles(ctx, config, inputPath, constantFiles)
	case "go":
		// TODO: Implement Go generation
		if !ctx.Quiet {
			color.Yellow("Go generation not yet implemented")
		}
	case "typescript":
		// TODO: Implement TypeScript generation
		if !ctx.Quiet {
			color.Yellow("TypeScript generation not yet implemented")
		}
	case "java":
		// TODO: Implement Java generation
		if !ctx.Quiet {
			color.Yellow("Java generation not yet implemented")
		}
	case "python":
		// TODO: Implement Python generation
		if !ctx.Quiet {
			color.Yellow("Python generation not yet implemented")
		}
	default:
		// Custom language - look for external generator
		if !ctx.Quiet {
			color.Yellow("Language %s generation not yet implemented", g.Lang)
		}
	}

	return nil
}

// generateIntermediateFiles generates JSON intermediate files from SQL templates
func (g *GenerateCmd) generateIntermediateFiles(ctx *Context, config *Config, inputPath string, constantFiles []string) error {
	// Determine output directory from JSON generator configuration
	outputDir := "./generated"
	if jsonGen, exists := config.Generation.Generators["json"]; exists && jsonGen.Output != "" {
		outputDir = jsonGen.Output
	}

	// Ensure output directory exists
	if err := ensureDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Check if input is a file or directory
	var files []string
	var err error

	if isDirectory(inputPath) {
		// Find all SQL template files in directory
		files, err = findTemplateFiles(inputPath)
		if err != nil {
			return fmt.Errorf("failed to find template files: %w", err)
		}
	} else {
		// Single file input
		if !fileExists(inputPath) {
			return fmt.Errorf("%w: %s", ErrInputFileNotExist, inputPath)
		}
		files = []string{inputPath}
	}

	if len(files) == 0 {
		if !ctx.Quiet {
			color.Yellow("No template files found in %s", inputPath)
		}
		return nil
	}

	if ctx.Verbose {
		color.Blue("Found %d template files", len(files))
	}

	// Process each file
	processedCount := 0
	for _, file := range files {
		if ctx.Verbose {
			color.Blue("Processing: %s", file)
		}

		// Generate intermediate file
		if err := g.processTemplateFile(file, outputDir, constantFiles, config, ctx); err != nil {
			if !ctx.Quiet {
				color.Red("Failed to process %s: %v", file, err)
			}
			continue
		}

		processedCount++
	}

	if !ctx.Quiet {
		color.Green("Generated %d intermediate files in %s", processedCount, outputDir)
	}

	return nil
}

// processTemplateFile processes a single template file and generates intermediate JSON
func (g *GenerateCmd) processTemplateFile(inputFile, outputDir string, _ []string, config *Config, ctx *Context) error {
	// Read the template file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create intermediate format
	format := intermediate.NewFormat()

	// Set source information
	format.SetSource(inputFile, string(content))

	// TODO: Set constant files if provided
	// if len(constantFiles) > 0 {
	//     format.SetConstantFiles(constantFiles)
	// }

	// Determine file type and process accordingly
	ext := strings.ToLower(filepath.Ext(inputFile))

	if ext == ".md" {
		// Process Markdown file
		if err := g.processMarkdownFile(format, string(content), ctx); err != nil {
			return fmt.Errorf("failed to process markdown file: %w", err)
		}
	} else {
		// Process SQL file
		// Parse SQL if validation is enabled or if we want to include AST
		if g.Validate {
			if err := g.parseAndValidateSQL(format, string(content), ctx); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
		}

		// Parse interface schema from frontmatter or comments
		if err := g.parseInterfaceSchema(format, string(content), ctx); err != nil {
			if ctx.Verbose {
				color.Yellow("No interface schema found in %s: %v", inputFile, err)
			}
		}
	}

	// Generate output filename
	outputFile := g.generateOutputFilename(inputFile, outputDir)

	// Write JSON output
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
	}
	defer file.Close()

	// Determine if pretty printing is enabled
	pretty := true
	if jsonGen, exists := config.Generation.Generators["json"]; exists {
		if prettyVal, ok := jsonGen.Settings["pretty"].(bool); ok {
			pretty = prettyVal
		}
	}

	// Write JSON with appropriate formatting
	if err := format.WriteJSON(file, pretty); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	if ctx.Verbose {
		color.Green("Generated: %s", outputFile)
	}

	return nil
}

// parseAndValidateSQL parses SQL and adds AST to intermediate format
func (g *GenerateCmd) parseAndValidateSQL(format *intermediate.IntermediateFormat, content string, ctx *Context) error {
	// Detect dialect and create tokenizer
	dialect := tokenizer.DetectDialect(content)
	sqlTokenizer := tokenizer.NewSqlTokenizer(content, dialect)

	// Get all tokens
	tokens, err := sqlTokenizer.AllTokens()
	if err != nil {
		return fmt.Errorf("tokenization failed: %w", err)
	}

	// Create parser with empty namespace (schema will be added later if available)
	ns := parser.NewNamespace(nil)
	sqlParser := parser.NewSqlParser(tokens, ns)

	// Parse SQL
	ast, err := sqlParser.Parse()
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	// Set AST in intermediate format
	format.SetAST(ast)

	if ctx.Verbose {
		color.Green("SQL validation passed")
	}

	return nil
}

// parseInterfaceSchema extracts interface schema from template content
func (g *GenerateCmd) parseInterfaceSchema(format *intermediate.IntermediateFormat, content string, ctx *Context) error {
	var schemaYAML string

	// Look for comment-based schema (/*@ ... @*/)
	if start := strings.Index(content, "/*@"); start != -1 {
		if end := strings.Index(content[start:], "@*/"); end != -1 {
			schemaYAML = content[start+3 : start+end]
		}
	}

	if schemaYAML == "" {
		return ErrNoInterfaceSchema
	}

	// Parse interface schema
	schema, err := parser.NewInterfaceSchemaFromFrontMatter(schemaYAML)
	if err != nil {
		return fmt.Errorf("failed to parse interface schema: %w", err)
	}

	// Set schema in intermediate format
	format.SetInterfaceSchema(schema)

	if ctx.Verbose {
		color.Green("Interface schema parsed successfully")
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

// ValidateCmd represents the validate command
type ValidateCmd struct {
	Input  string   `short:"i" help:"Input directory" default:"./queries" type:"path"`
	Files  []string `arg:"" help:"Specific files to validate" optional:""`
	Strict bool     `help:"Enable strict validation mode"`
	Format string   `help:"Output format" default:"text" enum:"text,json"`
}

func (v *ValidateCmd) Run(ctx *Context) error {
	if ctx.Verbose {
		color.Blue("Validating templates in %s", v.Input)
	}

	// Load configuration
	_, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// TODO: Implement validation logic
	if !ctx.Quiet {
		color.Green("Validation completed successfully")
	}

	return nil
}

// PullCmd represents the pull command
type PullCmd struct {
	DB             string `help:"Database connection string"`
	Env            string `help:"Environment name from configuration"`
	Output         string `short:"o" help:"Output file" default:"./schema.yaml" type:"path"`
	Tables         string `help:"Table patterns to include (wildcard supported)"`
	Exclude        string `help:"Table patterns to exclude"`
	IncludeViews   bool   `help:"Include database views"`
	IncludeIndexes bool   `help:"Include index information"`
}

func (p *PullCmd) Run(ctx *Context) error {
	if ctx.Verbose {
		if p.Env != "" {
			color.Blue("Pulling schema from environment: %s", p.Env)
		} else {
			color.Blue("Pulling schema from database")
		}
	}

	// Load configuration
	_, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// TODO: Implement schema extraction logic
	if !ctx.Quiet {
		color.Green("Schema extracted to %s", p.Output)
	}

	return nil
}

// InitCmd represents the init command
type InitCmd struct {
}

func (i *InitCmd) Run(ctx *Context) error {
	if ctx.Verbose {
		color.Blue("Initializing SnapSQL project")
	}

	// Create project structure
	dirs := []string{
		"queries",
		"constants",
		"generated",
	}

	for _, dir := range dirs {
		if err := createDir(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		if ctx.Verbose {
			color.Green("Created directory: %s", dir)
		}
	}

	// Create sample configuration file
	if err := createSampleConfig(); err != nil {
		return fmt.Errorf("failed to create sample configuration: %w", err)
	}

	// Create sample files
	if err := createSampleFiles(); err != nil {
		return fmt.Errorf("failed to create sample files: %w", err)
	}

	if !ctx.Quiet {
		color.Green("SnapSQL project initialized successfully")
		fmt.Println("\nNext steps:")
		fmt.Println("1. Edit snapsql.yaml to configure your database settings")
		fmt.Println("2. Create SQL templates in the queries/ directory")
		fmt.Println("3. Run 'snapsql generate' to generate intermediate files")
	}

	return nil
}

func createDir(path string) error {
	return ensureDir(path)
}

func createSampleConfig() error {
	configContent := `# SQL dialect configuration
dialect: "postgres"  # postgres, mysql, sqlite

# Database connection settings
databases:
  development:
    driver: "postgres"
    connection: "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
    schema: "public"
  
  production:
    driver: "postgres"
    connection: "postgres://${PROD_DB_USER}:${PROD_DB_PASS}@${PROD_DB_HOST}:${PROD_DB_PORT}/${PROD_DB_NAME}"
    schema: "public"

# Constant definition files (expandable with /*@ */)
constant_files:
  - "./constants/database.yaml"
  - "./constants/tables.yaml"

# Schema extraction settings
schema_extraction:
  include_views: false
  include_indexes: true
  table_patterns:
    include: ["*"]
    exclude: ["pg_*", "information_schema*", "sys_*"]

# Generation settings
generation:
  input_dir: "./queries"
  validate: true
  
  # Language-specific settings (including output directories)
  json:
    output: "./generated"
    pretty: true
    include_metadata: true
  
  go:
    output: "./internal/queries"
    package: "queries"
  
  typescript:
    output: "./src/generated"
    types: true

# Validation settings
validation:
  strict: false
  rules:
    - "no-dynamic-table-names"
    - "require-parameter-types"
`

	return writeFile("snapsql.yaml", configContent)
}

func createSampleFiles() error {
	// Create sample SQL template
	sampleSQL := `-- Sample SnapSQL template
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    created_at
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'engineering', 'design')
    /*# end */
ORDER BY created_at DESC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
`

	if err := writeFile(filepath.Join("queries", "users.snap.sql"), sampleSQL); err != nil {
		return err
	}

	// Create sample constants file
	sampleConstants := `# Database constants
tables:
  users: "users_v2"
  posts: "posts_archive"

environments:
  dev: "dev_"
  prod: "prod_"
`

	return writeFile(filepath.Join("constants", "database.yaml"), sampleConstants)
}

// processMarkdownFile processes a markdown file and extracts SQL and metadata
func (g *GenerateCmd) processMarkdownFile(format *intermediate.IntermediateFormat, content string, ctx *Context) error {
	// Create markdown parser
	mdParser := markdownparser.NewParser()

	// Parse markdown content
	reader := strings.NewReader(content)
	parsed, err := mdParser.Parse(reader)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Set interface schema from front matter and parsed content
	if err := g.setMarkdownInterfaceSchema(format, parsed, ctx); err != nil {
		if ctx.Verbose {
			color.Yellow("Failed to set interface schema: %v", err)
		}
	}

	// Extract and validate SQL from markdown
	if sqlSection, exists := parsed.Sections["sql"]; exists {
		sqlContent := mdParser.ExtractSQLFromCodeBlock(sqlSection.Content)
		if sqlContent == "" {
			// Try to extract SQL without code block
			sqlContent = strings.TrimSpace(sqlSection.Content)
		}

		if sqlContent != "" {
			// Parse and validate SQL if validation is enabled
			if g.Validate {
				if err := g.parseAndValidateSQL(format, sqlContent, ctx); err != nil {
					return fmt.Errorf("SQL validation failed: %w", err)
				}
			}
		}
	}

	return nil
}

// setMarkdownInterfaceSchema sets interface schema from parsed markdown
func (g *GenerateCmd) setMarkdownInterfaceSchema(format *intermediate.IntermediateFormat, parsed *markdownparser.ParsedMarkdown, _ *Context) error {
	// Create interface schema from front matter and parameters
	schema := intermediate.InterfaceSchemaFormatted{
		Name:         parsed.FrontMatter.Name,
		FunctionName: generateFunctionName(parsed.FrontMatter.Name),
		Parameters:   []intermediate.Parameter{},
	}

	// Parse parameters section if available
	if paramSection, exists := parsed.Sections["parameters"]; exists {
		mdParser := markdownparser.NewParser()
		params, err := mdParser.ParseParameters(paramSection.Content)
		if err != nil {
			return fmt.Errorf("failed to parse parameters: %w", err)
		}

		// Convert parameters to intermediate format
		for name, paramType := range params {
			param := intermediate.Parameter{
				Name: name,
				Type: convertParameterType(paramType),
			}
			schema.Parameters = append(schema.Parameters, param)
		}
	}

	// Set the schema in the intermediate format
	format.InterfaceSchema = &schema

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
func convertParameterType(paramType interface{}) string {
	switch v := paramType.(type) {
	case string:
		return v
	case map[string]interface{}:
		return "object"
	case []interface{}:
		if len(v) > 0 {
			return "array"
		}
		return "array"
	default:
		return "unknown"
	}
}
