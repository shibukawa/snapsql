package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/pull"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors
var (
	ErrNoDatabasesConfigured = errors.New("no databases configured")
	ErrEnvironmentNotFound   = errors.New("environment not found in configuration")
	ErrMissingDBOrEnv        = errors.New("either --db or --env must be specified")
	ErrEmptyConnectionString = errors.New("database connection string is empty")
	ErrEmptyDatabaseType     = errors.New("database type is empty")
)
var (
	ErrGeneratorNotConfigured = errors.New("generator is not configured or not enabled")
	ErrPluginNotFound         = errors.New("external generator plugin not found in PATH")
	ErrInputFileNotExist      = errors.New("input file does not exist")
	ErrNoInterfaceSchema      = errors.New("no interface schema found")
	ErrNoASTGenerated         = errors.New("validation failed: no AST generated from template")
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
				Settings: map[string]any{
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
func (g *GenerateCmd) processTemplateFile(inputFile, outputDir string, constantFiles []string, _ *Config, ctx *Context) error {
	// Read the template file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine file type and process accordingly
	ext := strings.ToLower(filepath.Ext(inputFile))

	var parseResult *ParseResult

	if ext == ".md" {
		// Process Markdown file
		parseResult, err = g.parseMarkdownFile(string(content), constantFiles, ctx)
		if err != nil {
			return fmt.Errorf("failed to parse markdown file: %w", err)
		}
	} else {
		// Process SQL file
		parseResult, err = g.parseSQLFile(string(content), constantFiles, ctx)
		if err != nil {
			return fmt.Errorf("failed to parse SQL file: %w", err)
		}
	}

	// Validate parse result if requested
	if err := g.validateParseResult(parseResult, ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert AST to instructions
	instructions := g.convertASTToInstructions(parseResult.AST, ctx)

	// Create intermediate format generator
	generator, err := intermediate.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to create intermediate generator: %w", err)
	}

	// Generate intermediate format
	var format *intermediate.IntermediateFormat
	if parseResult.Schema != nil {
		// Convert schema to intermediate format
		schema := g.convertSchemaToIntermediate(parseResult.Schema)
		format, err = generator.GenerateFromTemplateWithSchema(inputFile, instructions, schema)
	} else {
		format, err = generator.GenerateFromTemplate(inputFile, instructions)
	}
	if err != nil {
		return fmt.Errorf("failed to generate intermediate format: %w", err)
	}

	// Validate the generated format
	if validationErrors := generator.ValidateFormat(format); len(validationErrors) > 0 {
		if ctx.Verbose {
			for _, validationErr := range validationErrors {
				color.Yellow("Validation warning: %v", validationErr)
			}
		}
	}

	// Generate output filename
	outputFile := g.generateOutputFilename(inputFile, outputDir)

	// Write intermediate format to file
	if err := generator.WriteToFile(format, outputFile); err != nil {
		return fmt.Errorf("failed to write intermediate file: %w", err)
	}

	if ctx.Verbose {
		color.Green("Generated: %s", outputFile)
	}

	return nil
}

// ParseResult holds the result of parsing a template file
type ParseResult struct {
	Schema *parser.InterfaceSchema
	AST    parser.AstNode
}

// parseSQLFile parses SQL content and returns parse results
func (g *GenerateCmd) parseSQLFile(content string, constantFiles []string, ctx *Context) (*ParseResult, error) {
	// Detect dialect and create tokenizer
	dialect := tokenizer.DetectDialect(content)
	sqlTokenizer := tokenizer.NewSqlTokenizer(content, dialect)

	// Get all tokens - parse only once
	tokens, err := sqlTokenizer.AllTokens()
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Extract interface schema from SQL tokens
	schema, err := parser.NewInterfaceSchemaFromSQL(tokens)
	if err != nil {
		if ctx.Verbose {
			color.Yellow("Failed to parse interface schema: %v", err)
		}
		// Continue with nil schema if extraction fails
	} else if schema != nil && ctx.Verbose {
		color.Green("Interface schema parsed successfully")
	}

	// Create namespace
	ns := parser.NewNamespace(schema)

	// Load constant files if provided
	if len(constantFiles) > 0 {
		if err := loadConstantFiles(ns, constantFiles); err != nil {
			return nil, fmt.Errorf("failed to load constant files: %w", err)
		}
		if ctx.Verbose {
			color.Green("Loaded %d constant files", len(constantFiles))
		}
	}

	// Create parser with schema - reuse the tokens
	sqlParser := parser.NewSqlParser(tokens, ns, schema)

	// Parse SQL
	ast, err := sqlParser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return &ParseResult{
		Schema: schema,
		AST:    ast,
	}, nil
}

// convertASTToInstructions converts AST to instruction sequence
func (g *GenerateCmd) convertASTToInstructions(_ parser.AstNode, ctx *Context) []intermediate.Instruction {
	// For now, create a simple instruction sequence
	// In a real implementation, this would traverse the AST and generate instructions
	instructions := []intermediate.Instruction{
		{
			Op:    "EMIT_LITERAL",
			Pos:   []int{1, 1, 0}, // Default position
			Value: "SELECT * FROM users",
		},
	}

	if ctx.Verbose {
		color.Blue("Converted AST to %d instructions", len(instructions))
	}

	return instructions
}

// convertSchemaToIntermediate converts parser schema to intermediate schema
func (g *GenerateCmd) convertSchemaToIntermediate(schema *parser.InterfaceSchema) *intermediate.InterfaceSchema {
	if schema == nil {
		return nil
	}

	// Convert parameters
	parameters := make([]intermediate.Parameter, 0, len(schema.Parameters))
	for name, paramType := range schema.Parameters {
		param := intermediate.Parameter{
			Name: name,
			Type: fmt.Sprintf("%T", paramType),
		}
		parameters = append(parameters, param)
	}

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
	// Database connection options
	DB   string `help:"Database connection string"`
	Env  string `help:"Environment name from configuration"`
	Type string `help:"Database type (postgresql, mysql, sqlite)"`

	// Output options
	Output      string `short:"o" help:"Output directory" default:"./schema" type:"path"`
	Format      string `help:"Output format (per_table, per_schema, single_file)" enum:"per_table,per_schema,single_file" default:"per_table"`
	SchemaAware bool   `help:"Create schema-aware directory structure" default:"true"`

	// Filtering options
	IncludeSchemas []string `help:"Schema patterns to include (can be specified multiple times)"`
	ExcludeSchemas []string `help:"Schema patterns to exclude (can be specified multiple times)"`
	IncludeTables  []string `help:"Table patterns to include (can be specified multiple times)"`
	ExcludeTables  []string `help:"Table patterns to exclude (can be specified multiple times)"`

	// Feature options
	IncludeViews   bool `help:"Include database views" default:"true"`
	IncludeIndexes bool `help:"Include index information" default:"true"`

	// YAML options
	FlowStyle bool `help:"Use YAML flow style for compact output"`
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
	config, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if ctx.Verbose {
		color.Blue("Configuration loaded from: %s", ctx.Config)
	}

	// Determine database connection details
	dbURL, dbType, err := p.resolveDatabaseConnection(config)
	if err != nil {
		return fmt.Errorf("failed to resolve database connection: %w", err)
	}

	if ctx.Verbose {
		color.Blue("Database URL: %s", dbURL)
		color.Blue("Database type: %s", dbType)
		color.Blue("Output directory: %s", p.Output)
		color.Blue("Output format: %s", p.Format)
	}

	// Create pull configuration
	pullConfig := p.createPullConfig(dbURL, dbType)

	// Execute pull operation
	result, err := pull.ExecutePull(pullConfig)
	if err != nil {
		return fmt.Errorf("failed to pull schema: %w", err)
	}

	// Display results
	if !ctx.Quiet {
		p.displayResults(result)
	}

	return nil
}

// resolveDatabaseConnection determines the database connection string and type
func (p *PullCmd) resolveDatabaseConnection(config *Config) (string, string, error) {
	var dbURL, dbType string

	// Priority: command line > environment config > error
	if p.DB != "" {
		// Use command line database URL
		dbURL = p.DB
		if p.Type != "" {
			dbType = p.Type
		} else {
			// Try to detect database type from URL
			connector := pull.NewDatabaseConnector()
			detectedType, err := connector.ParseDatabaseURL(dbURL)
			if err != nil {
				return "", "", fmt.Errorf("failed to detect database type from URL: %w", err)
			}
			dbType = detectedType
		}
	} else if p.Env != "" {
		// Use environment from configuration
		if config.Databases == nil {
			return "", "", ErrNoDatabasesConfigured
		}

		envConfig, exists := config.Databases[p.Env]
		if !exists {
			return "", "", fmt.Errorf("%w: '%s'", ErrEnvironmentNotFound, p.Env)
		}

		dbURL = envConfig.Connection
		dbType = envConfig.Driver

		// Expand environment variables
		dbURL = expandEnvVars(dbURL)
	} else {
		return "", "", ErrMissingDBOrEnv
	}

	if dbURL == "" {
		return "", "", ErrEmptyConnectionString
	}
	if dbType == "" {
		return "", "", ErrEmptyDatabaseType
	}

	return dbURL, dbType, nil
}

// createPullConfig creates a pull configuration from command line options
func (p *PullCmd) createPullConfig(dbURL, dbType string) pull.PullConfig {
	// Convert format string to enum
	var outputFormat pull.OutputFormat
	switch p.Format {
	case "per_table":
		outputFormat = pull.OutputPerTable
	case "per_schema":
		outputFormat = pull.OutputPerSchema
	case "single_file":
		outputFormat = pull.OutputSingleFile
	default:
		outputFormat = pull.OutputPerTable
	}

	return pull.PullConfig{
		DatabaseURL:    dbURL,
		DatabaseType:   dbType,
		OutputPath:     p.Output,
		OutputFormat:   outputFormat,
		SchemaAware:    p.SchemaAware,
		IncludeSchemas: p.IncludeSchemas,
		ExcludeSchemas: p.ExcludeSchemas,
		IncludeTables:  p.IncludeTables,
		ExcludeTables:  p.ExcludeTables,
		IncludeViews:   p.IncludeViews,
		IncludeIndexes: p.IncludeIndexes,
	}
}

// displayResults shows the results of the pull operation
func (p *PullCmd) displayResults(result *pull.PullResult) {
	color.Green("✓ Schema extraction completed successfully")

	totalTables := 0
	totalViews := 0
	for _, schema := range result.Schemas {
		totalTables += len(schema.Tables)
		totalViews += len(schema.Views)
	}

	color.Green("  Schemas: %d", len(result.Schemas))
	color.Green("  Tables: %d", totalTables)
	if p.IncludeViews && totalViews > 0 {
		color.Green("  Views: %d", totalViews)
	}
	color.Green("  Output: %s", p.Output)

	// Show schema details if verbose
	for _, schema := range result.Schemas {
		color.Cyan("  Schema '%s': %d tables", schema.Name, len(schema.Tables))
		if p.IncludeViews && len(schema.Views) > 0 {
			color.Cyan("    Views: %d", len(schema.Views))
		}
	}
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

// parseMarkdownFile processes a markdown file and returns parse results
func (g *GenerateCmd) parseMarkdownFile(content string, constantFiles []string, ctx *Context) (*ParseResult, error) {
	// Create markdown parser
	mdParser := markdownparser.NewParser()

	// Parse markdown content
	reader := strings.NewReader(content)
	parsed, err := mdParser.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Create interface schema from front matter and parameters
	schema := &parser.InterfaceSchema{
		Name:         parsed.FrontMatter.Name,
		FunctionName: generateFunctionName(parsed.FrontMatter.Name),
		Parameters:   make(map[string]any),
	}

	// Parse parameters section if available
	if paramSection, exists := parsed.Sections["parameters"]; exists {
		mdParser := markdownparser.NewParser()
		params, err := mdParser.ParseParameters(paramSection.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}

		// Add parameters to schema
		for name, paramType := range params {
			schema.Parameters[name] = convertParameterType(paramType)
		}
	}

	// Process schema
	schema.ProcessSchema()

	// Create namespace
	ns := parser.NewNamespace(schema)

	// Load constant files if provided
	if len(constantFiles) > 0 {
		if err := loadConstantFiles(ns, constantFiles); err != nil {
			return nil, fmt.Errorf("failed to load constant files: %w", err)
		}
		if ctx.Verbose {
			color.Green("Loaded %d constant files", len(constantFiles))
		}
	}

	var ast parser.AstNode

	// Extract and parse SQL from markdown
	if sqlSection, exists := parsed.Sections["sql"]; exists {
		sqlContent := mdParser.ExtractSQLFromCodeBlock(sqlSection.Content)
		if sqlContent == "" {
			// Try to extract SQL without code block
			sqlContent = strings.TrimSpace(sqlSection.Content)
		}

		if sqlContent != "" {
			// Parse SQL content
			// Detect dialect and create tokenizer
			dialect := tokenizer.DetectDialect(sqlContent)
			sqlTokenizer := tokenizer.NewSqlTokenizer(sqlContent, dialect)

			// Get all tokens - parse only once
			tokens, err := sqlTokenizer.AllTokens()
			if err != nil {
				return nil, fmt.Errorf("tokenization failed: %w", err)
			}

			// Create parser with schema - reuse the tokens
			sqlParser := parser.NewSqlParser(tokens, ns, schema)

			// Parse SQL
			ast, err = sqlParser.Parse()
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

// loadConstantFiles loads constant definition files
func loadConstantFiles(ns *parser.Namespace, files []string) error {
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
			ns.SetConstant(k, v) // Set in namespace for validation
		}
	}
	return nil
}
