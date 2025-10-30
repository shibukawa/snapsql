package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// InitCmd represents the init command
type InitCmd struct {
}

func (i *InitCmd) Run(ctx *Context) error {
	if ctx.Verbose {
		color.Blue("Initializing SnapSQL project")
	}

	// Detect if we are inside examples/kanban (special scaffolding for sample)
	pwd, _ := os.Getwd()

	var isKanban bool
	if strings.Contains(filepath.ToSlash(pwd), "/examples/kanban") {
		isKanban = true
	}

	// Create project structure (kanban uses internal/query as output)
	dirs := []string{
		"queries",
		"constants",
	}
	if isKanban {
		dirs = append(dirs, filepath.Join("internal", "query"))
	} else {
		dirs = append(dirs, "generated")
	}

	dirs = append(dirs, filepath.Join("testdata", "mock"))

	for _, dir := range dirs {
		err := createDir(dir)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if ctx.Verbose {
			color.Green("Created directory: %s", dir)
		}
	}

	// Create sample configuration file (kanban variant if in examples/kanban)
	var err error
	if isKanban {
		err = createKanbanConfig()
	} else {
		err = createSampleConfig()
	}

	if err != nil {
		return fmt.Errorf("failed to create sample configuration: %w", err)
	}

	// Create sample files
	err = createSampleFiles()
	if err != nil {
		return fmt.Errorf("failed to create sample files: %w", err)
	}

	// Create VS Code settings for YAML schema validation
	err = createVSCodeSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to create VS Code settings: %w", err)
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
	return os.MkdirAll(path, 0755)
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

# Constant definition files (expandable with /*# */)
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
  
  # Generator configurations
  # Generators are enabled by default unless 'disabled: true' is specified
  generators:
    json:
      output: "./generated"
      preserve_hierarchy: true
      settings:
        pretty: true
        include_metadata: true

    mock:
      output: "./testdata/mock"
      preserve_hierarchy: true
      settings:
        embed: true
        package: "mock"
        filename: "mock.go"
    
    go:
      output: "./internal/queries"
      disabled: true
      preserve_hierarchy: true
      settings:
        package: "queries"
    
    typescript:
      output: "./src/generated"
      disabled: true
      preserve_hierarchy: true
      settings:
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

// createKanbanConfig writes a snapsql.yaml tailored for the kanban example with Go generator
// output directed to internal/query and package name `query`.
func createKanbanConfig() error {
	// Build YAML with only spaces to ensure parser compatibility
	var b strings.Builder
	b.WriteString("dialect: \"sqlite\"\n")
	b.WriteString("input_dir: \"./queries\"\n\n")
	b.WriteString("generation:\n")
	b.WriteString("  validate: true\n")
	b.WriteString("  generators:\n")
	b.WriteString("    json:\n")
	b.WriteString("      output: \"./generated\"\n")
	b.WriteString("      preserve_hierarchy: true\n")
	b.WriteString("      settings:\n")
	b.WriteString("        pretty: true\n")
	b.WriteString("        include_metadata: true\n")
	b.WriteString("    mock:\n")
	b.WriteString("      output: \"./testdata/mock\"\n")
	b.WriteString("      preserve_hierarchy: true\n")
	b.WriteString("      settings:\n")
	b.WriteString("        embed: true\n")
	b.WriteString("        package: \"mock\"\n")
	b.WriteString("        filename: \"mock.go\"\n")
	b.WriteString("    go:\n")
	b.WriteString("      output: \"./internal/query\"\n")
	b.WriteString("      preserve_hierarchy: true\n")
	b.WriteString("      settings:\n")
	b.WriteString("        package: \"query\"\n\n")
	b.WriteString("system:\n")
	b.WriteString("  fields:\n")
	b.WriteString("    - name: created_at\n")
	b.WriteString("      type: timestamp\n")
	b.WriteString("      exclude_from_select: false\n")
	b.WriteString("      on_insert:\n")
	b.WriteString("        default: NOW()\n")
	b.WriteString("    - name: updated_at\n")
	b.WriteString("      type: timestamp\n")
	b.WriteString("      exclude_from_select: false\n")
	b.WriteString("      on_insert:\n")
	b.WriteString("        default: NOW()\n")
	b.WriteString("      on_update:\n")
	b.WriteString("        default: NOW()\n")

	return writeFile("snapsql.yaml", b.String())
}

func createSampleFiles() error {
	// Create sample SQL template
	// (Kanban example) We don't create a default users query because the schema doesn't define users table.
	// Leaving placeholder file creation out intentionally.
	// If needed in future examples, add sample templates here.
	// No query file written in this initialization step now.

	// No sample query file created; proceed to constants file creation.

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

// writeFile writes content to a file
func writeFile(path, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// createVSCodeSettings creates or updates .vscode/settings.json with YAML schema configuration
func createVSCodeSettings(ctx *Context) error {
	vscodeDir := ".vscode"
	settingsPath := filepath.Join(vscodeDir, "settings.json")

	// Create .vscode directory if it doesn't exist
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	// Define the schema URL
	schemaURL := "https://raw.githubusercontent.com/shibukawa/snapsql/refs/heads/main/snapsql-config.schema.json"
	schemaPatterns := []string{"snapsql.yaml", "**/snapsql*.yaml"}

	// Check if settings.json already exists
	var settings map[string]any

	existingData, err := os.ReadFile(settingsPath)
	if err == nil {
		// File exists, parse it
		if err := json.Unmarshal(existingData, &settings); err != nil {
			// If parsing fails, create new settings
			if ctx.Verbose {
				color.Yellow("Warning: existing settings.json is invalid, creating new one")
			}

			settings = make(map[string]any)
		}
	} else {
		// File doesn't exist, create new settings
		settings = make(map[string]any)
	}

	// Add or update yaml.schemas configuration
	yamlSchemas, ok := settings["yaml.schemas"].(map[string]any)
	if !ok {
		yamlSchemas = make(map[string]any)
	}

	// Convert patterns to interface slice for JSON
	patterns := make([]any, len(schemaPatterns))
	for i, p := range schemaPatterns {
		patterns[i] = p
	}

	yamlSchemas[schemaURL] = patterns
	settings["yaml.schemas"] = yamlSchemas

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write settings file
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	if ctx.Verbose {
		color.Green("Created/updated .vscode/settings.json with YAML schema configuration")
	}

	return nil
}
