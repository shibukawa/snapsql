package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

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

// writeFile writes content to a file
func writeFile(path, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, []byte(content), 0644)
}
