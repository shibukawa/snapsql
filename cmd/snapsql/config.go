package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Config represents the SnapSQL configuration
type Config struct {
	Dialect       string                 `yaml:"dialect"`
	Databases     map[string]Database    `yaml:"databases"`
	ConstantFiles []string               `yaml:"constant_files"`
	Schema        SchemaExtractionConfig `yaml:"schema_extraction"`
	Generation    GenerationConfig       `yaml:"generation"`
	Validation    ValidationConfig       `yaml:"validation"`
}

// Database represents database connection configuration
type Database struct {
	Driver     string `yaml:"driver"`
	Connection string `yaml:"connection"`
	Schema     string `yaml:"schema"`
	Database   string `yaml:"database"`
}

// SchemaExtractionConfig represents schema extraction settings
type SchemaExtractionConfig struct {
	IncludeViews   bool          `yaml:"include_views"`
	IncludeIndexes bool          `yaml:"include_indexes"`
	TablePatterns  TablePatterns `yaml:"table_patterns"`
}

// TablePatterns represents table inclusion/exclusion patterns
type TablePatterns struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// GenerationConfig represents code generation settings
type GenerationConfig struct {
	InputDir   string                     `yaml:"input_dir"`
	Validate   bool                       `yaml:"validate"`
	Generators map[string]GeneratorConfig `yaml:"generators"`
}

// GeneratorConfig represents a single generator configuration
type GeneratorConfig struct {
	Output   string                 `yaml:"output"`
	Enabled  bool                   `yaml:"enabled"`
	Settings map[string]interface{} `yaml:"settings,omitempty"`
}

// LanguageConfig represents language-specific generation settings (deprecated, kept for backward compatibility)
type LanguageConfig struct {
	Output          string `yaml:"output"`
	Package         string `yaml:"package"`
	Pretty          bool   `yaml:"pretty"`
	IncludeMetadata bool   `yaml:"include_metadata"`
	Types           bool   `yaml:"types"`
}

// ValidationConfig represents validation settings
type ValidationConfig struct {
	Strict bool     `yaml:"strict"`
	Rules  []string `yaml:"rules"`
}

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	// Load .env files first
	if err := loadEnvFiles(); err != nil {
		return nil, fmt.Errorf("failed to load environment files: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default configuration if file doesn't exist
		config := getDefaultConfig()
		expandConfigEnvVars(config)
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values
	applyDefaults(&config)

	// Expand environment variables
	expandConfigEnvVars(&config)

	return &config, nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Dialect:       "postgres",
		Databases:     make(map[string]Database),
		ConstantFiles: []string{},
		Schema: SchemaExtractionConfig{
			IncludeViews:   false,
			IncludeIndexes: true,
			TablePatterns: TablePatterns{
				Include: []string{"*"},
				Exclude: []string{"pg_*", "information_schema*", "sys_*"},
			},
		},
		Generation: GenerationConfig{
			InputDir: "./queries",
			Validate: true,
			Generators: map[string]GeneratorConfig{
				"json": {
					Output:  "./generated",
					Enabled: true,
					Settings: map[string]interface{}{
						"pretty":           true,
						"include_metadata": true,
					},
				},
				"go": {
					Output:  "./internal/queries",
					Enabled: false,
					Settings: map[string]interface{}{
						"package": "queries",
					},
				},
				"typescript": {
					Output:  "./src/generated",
					Enabled: false,
					Settings: map[string]interface{}{
						"types": true,
					},
				},
			},
		},
		Validation: ValidationConfig{
			Strict: false,
			Rules: []string{
				"no-dynamic-table-names",
				"require-parameter-types",
			},
		},
	}
}

// applyDefaults applies default values to missing configuration fields
func applyDefaults(config *Config) {
	if config.Dialect == "" {
		config.Dialect = "postgres"
	}

	if config.Generation.InputDir == "" {
		config.Generation.InputDir = "./queries"
	}

	// Initialize generators map if nil
	if config.Generation.Generators == nil {
		config.Generation.Generators = make(map[string]GeneratorConfig)
	}

	// Apply default JSON generator if not configured
	if _, exists := config.Generation.Generators["json"]; !exists {
		config.Generation.Generators["json"] = GeneratorConfig{
			Output:  "./generated",
			Enabled: true,
			Settings: map[string]interface{}{
				"pretty":           true,
				"include_metadata": true,
			},
		}
	}

	// Ensure JSON generator is always enabled (it's required for other generators)
	if jsonGen, exists := config.Generation.Generators["json"]; exists {
		jsonGen.Enabled = true
		config.Generation.Generators["json"] = jsonGen
	}

	// Apply default schema extraction settings
	if len(config.Schema.TablePatterns.Include) == 0 {
		config.Schema.TablePatterns.Include = []string{"*"}
	}

	if len(config.Schema.TablePatterns.Exclude) == 0 {
		config.Schema.TablePatterns.Exclude = []string{"pg_*", "information_schema*", "sys_*"}
	}
}
