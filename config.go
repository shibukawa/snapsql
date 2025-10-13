package snapsql

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

// ErrConfigValidation is returned when configuration validation fails
var ErrConfigValidation = errors.New("configuration validation failed")

// Config represents the SnapSQL configuration
type Config struct {
	Dialect       string                      `yaml:"dialect"`
	InputDir      string                      `yaml:"input_dir"` // Moved from GenerationConfig
	Databases     map[string]Database         `yaml:"databases"`
	ConstantFiles []string                    `yaml:"constant_files"`
	Schema        SchemaExtractionConfig      `yaml:"schema_extraction"`
	Generation    GenerationConfig            `yaml:"generation"`
	Validation    ValidationConfig            `yaml:"validation"`
	Query         QueryConfig                 `yaml:"query"`
	System        SystemConfig                `yaml:"system"`
	Performance   PerformanceConfig           `yaml:"performance"`
	Tables        map[string]TablePerformance `yaml:"tables"`
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
	Validate         bool                       `yaml:"validate"`
	GenerateMockData bool                       `yaml:"generate_mock_data"`
	Generators       map[string]GeneratorConfig `yaml:"generators"`
}

// GeneratorConfig represents a single generator configuration
type GeneratorConfig struct {
	Output            string         `yaml:"output"`
	Disabled          *bool          `yaml:"disabled"` // Pointer to distinguish between unset and true. If nil or false, generator is enabled
	PreserveHierarchy bool           `yaml:"preserve_hierarchy"`
	Settings          map[string]any `yaml:"settings,omitempty"`
}

// IsEnabled returns true if the generator is not explicitly disabled
// Generators are enabled by default unless disabled: true is set
func (g *GeneratorConfig) IsEnabled() bool {
	return g.Disabled == nil || !*g.Disabled
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

// QueryConfig represents query execution settings
type QueryConfig struct {
	DefaultFormat         string `yaml:"default_format"`
	DefaultEnvironment    string `yaml:"default_environment"`
	Timeout               int    `yaml:"timeout"`
	MaxRows               int    `yaml:"max_rows"`
	Limit                 int    `yaml:"limit"`
	Offset                int    `yaml:"offset"`
	ExecuteDangerousQuery bool   `yaml:"execute_dangerous_query"`
	DeprecatedExplain        *bool `yaml:"explain,omitempty"`
	DeprecatedExplainAnalyze *bool `yaml:"explain_analyze,omitempty"`
}

// PerformanceConfig represents performance-related defaults
type PerformanceConfig struct {
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold"`
}

// TablePerformance defines per-table performance metadata
type TablePerformance struct {
	ExpectedRows  int64 `yaml:"expected_rows"`
	AllowFullScan bool  `yaml:"allow_full_scan"`
}

// SystemConfig represents system-level configuration
// This information will be used to augment schema information during pull operations
type SystemConfig struct {
	Fields []SystemField `yaml:"fields"`
}

// SystemField represents a single system field configuration
type SystemField struct {
	// Field name
	Name string `yaml:"name"`

	// Field type (for implicit parameters)
	Type string `yaml:"type"`

	// Whether to exclude this field from SELECT statements by default
	ExcludeFromSelect bool `yaml:"exclude_from_select"`

	// Configuration for INSERT operations
	OnInsert SystemFieldOperation `yaml:"on_insert"`

	// Configuration for UPDATE operations
	OnUpdate SystemFieldOperation `yaml:"on_update"`
}

// SystemFieldOperation represents the configuration for a system field in a specific operation
type SystemFieldOperation struct {
	// Default value (if specified, this field gets this default value)
	// Can be any type: string, int, bool, nil for SQL NULL, etc.
	Default any `yaml:"default,omitempty"`

	// Parameter configuration (how this field should be handled as a parameter)
	Parameter SystemFieldParameter `yaml:"parameter,omitempty"`
}

// SystemFieldParameter represents parameter handling configuration
type SystemFieldParameter string

const (
	// ParameterExplicit means the parameter must be explicitly provided by the user
	ParameterExplicit SystemFieldParameter = "explicit"

	// ParameterImplicit means the parameter is obtained from context/thread-local storage
	ParameterImplicit SystemFieldParameter = "implicit"

	// ParameterError means providing this parameter should result in an error
	ParameterError SystemFieldParameter = "error"

	// ParameterNone means no parameter handling (used when only default is specified)
	ParameterNone SystemFieldParameter = ""
)

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	// Load .env files first
	err := loadEnvFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to load environment files: %w", err)
	}

	// Check if config file exists
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
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

	// Parse YAML with strict mode to detect unknown fields
	var config Config

	err = yaml.UnmarshalWithOptions(data, &config, yaml.Strict())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Apply defaults for missing values
	applyDefaults(&config)

	// Expand environment variables
	expandConfigEnvVars(&config)

	return &config, nil
}

// validateConfig validates the configuration for common errors and inconsistencies
func validateConfig(config *Config) error {
	// Validate dialect
	validDialects := map[string]bool{
		"postgres":  true,
		"mysql":     true,
		"sqlite":    true,
		"sqlserver": true,
	}
	if config.Dialect != "" && !validDialects[config.Dialect] {
		return fmt.Errorf("%w: invalid dialect '%s': must be one of postgres, mysql, sqlite, sqlserver", ErrConfigValidation, config.Dialect)
	}

	// Validate generator configurations
	for name, generator := range config.Generation.Generators {
		// Validate output path is specified if enabled
		if generator.IsEnabled() && generator.Output == "" {
			return fmt.Errorf("%w: generator '%s': output path is required when enabled", ErrConfigValidation, name)
		}

		// Validate known generator types
		validGenerators := map[string]bool{
			"json":       true,
			"go":         true,
			"typescript": true,
		}
		if !validGenerators[name] {
			return fmt.Errorf("%w: unknown generator type '%s': must be one of json, go, typescript", ErrConfigValidation, name)
		}
	}

	// Validate system field parameter values
	for _, field := range config.System.Fields {
		if field.Name == "" {
			return fmt.Errorf("%w: system field: name is required", ErrConfigValidation)
		}

		// Validate parameter values for insert
		if field.OnInsert.Parameter != "" {
			validParams := map[SystemFieldParameter]bool{
				ParameterExplicit: true,
				ParameterImplicit: true,
				ParameterError:    true,
			}
			if !validParams[field.OnInsert.Parameter] {
				return fmt.Errorf("%w: system field '%s': invalid on_insert.parameter '%s': must be one of explicit, implicit, error", ErrConfigValidation, field.Name, field.OnInsert.Parameter)
			}
		}

		// Validate parameter values for update
		if field.OnUpdate.Parameter != "" {
			validParams := map[SystemFieldParameter]bool{
				ParameterExplicit: true,
				ParameterImplicit: true,
				ParameterError:    true,
			}
			if !validParams[field.OnUpdate.Parameter] {
				return fmt.Errorf("%w: system field '%s': invalid on_update.parameter '%s': must be one of explicit, implicit, error", ErrConfigValidation, field.Name, field.OnUpdate.Parameter)
			}
		}
	}

	// Validate query configuration
	if config.Query.Timeout < 0 {
		return fmt.Errorf("%w: query.timeout must be non-negative, got %d", ErrConfigValidation, config.Query.Timeout)
	}

	if config.Query.MaxRows < 0 {
		return fmt.Errorf("%w: query.max_rows must be non-negative, got %d", ErrConfigValidation, config.Query.MaxRows)
	}

	if config.Query.Limit < 0 {
		return fmt.Errorf("%w: query.limit must be non-negative, got %d", ErrConfigValidation, config.Query.Limit)
	}

	if config.Query.Offset < 0 {
		return fmt.Errorf("%w: query.offset must be non-negative, got %d", ErrConfigValidation, config.Query.Offset)
	}

	// Validate default format
	if config.Query.DefaultFormat != "" {
		validFormats := map[string]bool{
			"table":    true,
			"json":     true,
			"csv":      true,
			"markdown": true,
		}
		if !validFormats[config.Query.DefaultFormat] {
			return fmt.Errorf("%w: query.default_format '%s' is invalid: must be one of table, json, csv, markdown", ErrConfigValidation, config.Query.DefaultFormat)
		}
	}

	if config.Performance.SlowQueryThreshold < 0 {
		return fmt.Errorf("%w: performance.slow_query_threshold must be >= 0, got %s", ErrConfigValidation, config.Performance.SlowQueryThreshold)
	}

	for tableName, meta := range config.Tables {
		if meta.ExpectedRows <= 0 {
			return fmt.Errorf("%w: tables.%s.expected_rows must be a positive integer", ErrConfigValidation, tableName)
		}
	}

	return nil
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Dialect:       "postgres",
		InputDir:      "./queries", // Moved from GenerationConfig
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
			Validate: true,
			Generators: map[string]GeneratorConfig{
				"json": {
					Output:            "./generated",
					Disabled:          nil, // Enabled by default
					PreserveHierarchy: true,
					Settings: map[string]any{
						"pretty":           true,
						"include_metadata": true,
					},
				},
				"go": {
					Output:            "./internal/queries",
					Disabled:          boolPtr(true), // Disabled by default
					PreserveHierarchy: true,
					Settings: map[string]any{
						"package": "queries",
					},
				},
				"typescript": {
					Output:            "./src/generated",
					Disabled:          boolPtr(true), // Disabled by default
					PreserveHierarchy: true,
					Settings: map[string]any{
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
		Query: QueryConfig{
			DefaultFormat:         "table",
			DefaultEnvironment:    "development",
			Timeout:               30,
			MaxRows:               1000,
			Limit:                 0,
			Offset:                0,
			ExecuteDangerousQuery: false,
		},
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name:              "created_at",
					ExcludeFromSelect: false,
					OnInsert: SystemFieldOperation{
						Default: "NOW()",
					},
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterError,
					},
				},
				{
					Name:              "updated_at",
					ExcludeFromSelect: false,
					OnInsert: SystemFieldOperation{
						Default: "NOW()",
					},
					OnUpdate: SystemFieldOperation{
						Default: "NOW()",
					},
				},
				{
					Name:              "created_by",
					ExcludeFromSelect: false,
					OnInsert: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterError,
					},
				},
				{
					Name:              "updated_by",
					ExcludeFromSelect: false,
					OnInsert: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
				},
			},
		},
		Performance: PerformanceConfig{
			SlowQueryThreshold: 3 * time.Second,
		},
		Tables: make(map[string]TablePerformance),
	}
}

// applyDefaults applies default values to missing configuration fields
func applyDefaults(config *Config) {
	if config.Dialect == "" {
		config.Dialect = "postgres"
	}

	if config.InputDir == "" {
		config.InputDir = "./queries"
	}

	// Initialize generators map if nil
	if config.Generation.Generators == nil {
		config.Generation.Generators = make(map[string]GeneratorConfig)
	}

	// Apply default JSON generator if not configured
	jsonGen, exists := config.Generation.Generators["json"]
	if !exists {
		jsonGen = GeneratorConfig{
			Output:            "./generated",
			Disabled:          boolPtr(true), // Default to disabled unless explicitly enabled: false or omitted
			PreserveHierarchy: true,
			Settings: map[string]any{
				"pretty":           true,
				"include_metadata": true,
			},
		}
	} else {
		// Apply defaults for missing fields
		if jsonGen.Output == "" {
			jsonGen.Output = "./generated"
		}

		// For PreserveHierarchy, we'll default to true if not explicitly configured
		// Since we can't distinguish between false and unset in YAML, we'll use a heuristic:
		// If the config has no other custom settings, assume it's using defaults
		if jsonGen.Settings == nil && jsonGen.Output == "" {
			jsonGen.PreserveHierarchy = true
		}

		// Initialize settings if nil
		if jsonGen.Settings == nil {
			jsonGen.Settings = make(map[string]any)
		}

		// Apply default settings if not present
		if _, exists := jsonGen.Settings["pretty"]; !exists {
			jsonGen.Settings["pretty"] = true
		}

		if _, exists := jsonGen.Settings["include_metadata"]; !exists {
			jsonGen.Settings["include_metadata"] = true
		}
	}

	config.Generation.Generators["json"] = jsonGen

	// Apply default schema extraction settings
	if len(config.Schema.TablePatterns.Include) == 0 {
		config.Schema.TablePatterns.Include = []string{"*"}
	}

	if len(config.Schema.TablePatterns.Exclude) == 0 {
		config.Schema.TablePatterns.Exclude = []string{"pg_*", "information_schema*", "sys_*"}
	}

	// Apply default query settings
	if config.Query.DefaultFormat == "" {
		config.Query.DefaultFormat = "table"
	}

	if config.Query.Timeout == 0 {
		config.Query.Timeout = 30
	}

	if config.Query.MaxRows == 0 {
		config.Query.MaxRows = 1000
	}

	// Apply performance defaults
	if config.Performance.SlowQueryThreshold <= 0 {
		config.Performance.SlowQueryThreshold = 3 * time.Second
	}

	if config.Tables == nil {
		config.Tables = make(map[string]TablePerformance)
	}

	// Apply default system field settings
	applySystemFieldDefaults(config)
}

// applySystemFieldDefaults applies default values for system field configuration
func applySystemFieldDefaults(config *Config) {
	// Apply default system fields if empty
	if len(config.System.Fields) == 0 {
		config.System.Fields = []SystemField{
			{
				Name:              "created_at",
				ExcludeFromSelect: false,
				OnInsert: SystemFieldOperation{
					Default: "NOW()",
				},
				OnUpdate: SystemFieldOperation{
					Parameter: ParameterError,
				},
			},
			{
				Name:              "updated_at",
				ExcludeFromSelect: false,
				OnInsert: SystemFieldOperation{
					Default: "NOW()",
				},
				OnUpdate: SystemFieldOperation{
					Default: "NOW()",
				},
			},
			{
				Name:              "created_by",
				ExcludeFromSelect: false,
				OnInsert: SystemFieldOperation{
					Parameter: ParameterImplicit,
				},
				OnUpdate: SystemFieldOperation{
					Parameter: ParameterError,
				},
			},
			{
				Name:              "updated_by",
				ExcludeFromSelect: false,
				OnInsert: SystemFieldOperation{
					Parameter: ParameterImplicit,
				},
				OnUpdate: SystemFieldOperation{
					Parameter: ParameterImplicit,
				},
			},
		}
	}
}

// loadEnvFiles loads .env files if they exist
func loadEnvFiles() error {
	// Try to load .env file from current directory
	if fileExists(".env") {
		err := godotenv.Load(".env")
		if err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	return nil
}

// expandEnvVars expands environment variables in the format ${VAR} or $VAR
func expandEnvVars(s string) string {
	// Pattern for ${VAR} format
	re1 := regexp.MustCompile(`\$\{([^}]+)\}`)
	s = re1.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		return os.Getenv(varName)
	})

	// Pattern for $VAR format (word boundaries)
	re2 := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = re2.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1:] // Remove $
		return os.Getenv(varName)
	})

	return s
}

// expandConfigEnvVars recursively expands environment variables in config
func expandConfigEnvVars(config *Config) {
	// Expand database connections
	for name, db := range config.Databases {
		db.Connection = expandEnvVars(db.Connection)
		db.Driver = expandEnvVars(db.Driver)
		db.Schema = expandEnvVars(db.Schema)
		db.Database = expandEnvVars(db.Database)
		config.Databases[name] = db
	}

	// Expand constant files
	for i, file := range config.ConstantFiles {
		config.ConstantFiles[i] = expandEnvVars(file)
	}

	// Expand generation paths
	config.InputDir = expandEnvVars(config.InputDir)

	// Expand generator output paths
	for name, generator := range config.Generation.Generators {
		generator.Output = expandEnvVars(generator.Output)
		config.Generation.Generators[name] = generator
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsSystemField checks if a field name is considered a system field
func (c *Config) IsSystemField(fieldName string) bool {
	for _, field := range c.System.Fields {
		if field.Name == fieldName {
			return true
		}
	}

	return false
}

// GetSystemField returns the system field configuration for a field name
func (c *Config) GetSystemField(fieldName string) (SystemField, bool) {
	for _, field := range c.System.Fields {
		if field.Name == fieldName {
			return field, true
		}
	}

	return SystemField{}, false
}

// ShouldExcludeFromSelect checks if a specific system field should be excluded from SELECT statements by default
func (c *Config) ShouldExcludeFromSelect(fieldName string) bool {
	field, exists := c.GetSystemField(fieldName)
	if !exists {
		return false
	}

	return field.ExcludeFromSelect
}

// GetSystemFieldsForInsert returns all system fields that should be processed for INSERT statements
func (c *Config) GetSystemFieldsForInsert() []SystemField {
	var fields []SystemField

	for _, field := range c.System.Fields {
		// Include fields that have either default or parameter configuration for INSERT
		if field.OnInsert.Default != nil || field.OnInsert.Parameter != ParameterNone {
			fields = append(fields, field)
		}
	}

	return fields
}

// GetSystemFieldsForUpdate returns all system fields that should be processed for UPDATE statements
func (c *Config) GetSystemFieldsForUpdate() []SystemField {
	var fields []SystemField

	for _, field := range c.System.Fields {
		// Include fields that have either default or parameter configuration for UPDATE
		if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != ParameterNone {
			fields = append(fields, field)
		}
	}

	return fields
}

// HasDefaultForInsert checks if a system field has a default value for INSERT operations
func (c *Config) HasDefaultForInsert(fieldName string) bool {
	field, exists := c.GetSystemField(fieldName)
	if !exists {
		return false
	}

	return field.OnInsert.Default != nil
}

// HasDefaultForUpdate checks if a system field has a default value for UPDATE operations
func (c *Config) HasDefaultForUpdate(fieldName string) bool {
	field, exists := c.GetSystemField(fieldName)
	if !exists {
		return false
	}

	return field.OnUpdate.Default != nil
}

// GetParameterHandlingForInsert returns the parameter handling configuration for INSERT operations
func (c *Config) GetParameterHandlingForInsert(fieldName string) SystemFieldParameter {
	field, exists := c.GetSystemField(fieldName)
	if !exists {
		return ParameterNone
	}

	return field.OnInsert.Parameter
}

// GetParameterHandlingForUpdate returns the parameter handling configuration for UPDATE operations
func (c *Config) GetParameterHandlingForUpdate(fieldName string) SystemFieldParameter {
	field, exists := c.GetSystemField(fieldName)
	if !exists {
		return ParameterNone
	}

	return field.OnUpdate.Parameter
}

// GetDefaultValueForInsert returns the default value for INSERT operations
func (c *Config) GetDefaultValueForInsert(fieldName string) (any, bool) {
	field, exists := c.GetSystemField(fieldName)
	if !exists || field.OnInsert.Default == nil {
		return nil, false
	}

	return field.OnInsert.Default, true
}

// GetDefaultValueForUpdate returns the default value for UPDATE operations
func (c *Config) GetDefaultValueForUpdate(fieldName string) (any, bool) {
	field, exists := c.GetSystemField(fieldName)
	if !exists || field.OnUpdate.Default == nil {
		return nil, false
	}

	return field.OnUpdate.Default, true
}
