package main

import (
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
	b.WriteString("      enabled: true\n")
	b.WriteString("      preserve_hierarchy: true\n")
	b.WriteString("      settings:\n")
	b.WriteString("        pretty: true\n")
	b.WriteString("        include_metadata: true\n")
	b.WriteString("    go:\n")
	b.WriteString("      output: \"./internal/query\"\n")
	b.WriteString("      enabled: true\n")
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
