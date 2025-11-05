package pygen

import (
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// Config represents configuration for Python code generation
type Config struct {
	// PackageName is the Python module name (defaults to "generated")
	PackageName string

	// OutputPath is the directory where generated Python files will be written
	OutputPath string

	// Dialect specifies the target database (postgres, mysql, sqlite)
	Dialect snapsql.Dialect

	// MockPath is the path to mock data JSON files for testing
	MockPath string

	// Features contains optional feature flags
	Features FeatureConfig
}

// FeatureConfig contains optional feature flags for code generation
type FeatureConfig struct {
	// EnableQueryLogging enables query logging functionality
	EnableQueryLogging bool

	// EnableCaching enables query result caching
	EnableCaching bool

	// EnableRetry enables retry logic with exponential backoff
	EnableRetry bool

	// PreserveHierarchy enables hierarchical response structure support
	PreserveHierarchy bool
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		PackageName: "generated",
		OutputPath:  "./generated",
		Dialect:     snapsql.DialectPostgres,
		MockPath:    "",
		Features: FeatureConfig{
			EnableQueryLogging: true,
			EnableCaching:      false,
			EnableRetry:        false,
			PreserveHierarchy:  true,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate dialect
	switch c.Dialect {
	case snapsql.DialectPostgres, snapsql.DialectMySQL, snapsql.DialectSQLite:
		// Valid dialects
	case "":
		return ErrDialectRequired
	default:
		return ErrUnsupportedDialect
	}

	// Validate package name
	if c.PackageName == "" {
		c.PackageName = "generated"
	}

	// Validate output path
	if c.OutputPath == "" {
		c.OutputPath = "./generated"
	}

	return nil
}

// ApplyToGenerator applies configuration to a Generator
func (c *Config) ApplyToGenerator(g *Generator) {
	if c.PackageName != "" {
		g.PackageName = c.PackageName
	}

	if c.OutputPath != "" {
		g.OutputPath = c.OutputPath
	}

	if c.Dialect != "" {
		g.Dialect = c.Dialect
	}

	if c.MockPath != "" {
		g.MockPath = c.MockPath
	}
}

// NewGeneratorFromConfig creates a Generator from Config
func NewGeneratorFromConfig(format *intermediate.IntermediateFormat, config Config) (*Generator, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create generator with options
	g := New(
		format,
		WithPackageName(config.PackageName),
		WithOutputPath(config.OutputPath),
		WithDialect(config.Dialect),
		WithMockPath(config.MockPath),
	)

	return g, nil
}
