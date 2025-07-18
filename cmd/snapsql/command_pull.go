package main

import (
	"errors"
	"fmt"

	"github.com/fatih/color"
	"github.com/shibukawa/snapsql/pull"
)

// Error definitions
var (
	ErrNoDatabasesConfigured  = errors.New("no databases configured")
	ErrEnvironmentNotFound    = errors.New("environment not found")
	ErrMissingDBOrEnv        = errors.New("either database URL or environment must be specified")
	ErrEmptyConnectionString = errors.New("database connection string is empty")
	ErrEmptyDatabaseType     = errors.New("database type is not specified")
)

// PullCmd represents the pull command
type PullCmd struct {
	// Database connection options
	DB   string `help:"Database connection string"`
	Env  string `help:"Environment name from configuration"`
	Type string `help:"Database type (postgresql, mysql, sqlite)"`

	// Output options
	Output      string `short:"o" help:"Output directory" default:"./schema" type:"path"`
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
	return pull.PullConfig{
		DatabaseURL:    dbURL,
		DatabaseType:   dbType,
		OutputPath:     p.Output,
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
