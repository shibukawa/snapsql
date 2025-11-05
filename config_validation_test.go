package snapsql

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestLoadConfig_StrictMode_UnknownKeys(t *testing.T) {
	// Create a temporary config file with unknown keys
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "snapsql.yaml")

	configContent := `
dialect: "postgres"
input_dir: "./queries"
unknown_key: "should cause error"
generation:
  validate: true
  generators:
    json:
      output: "./generated"
      unknown_generator_key: "should also cause error"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Load config should fail due to unknown keys
	_, err = LoadConfig(configPath)
	assert.Error(t, err, "expected error for unknown keys in strict mode")
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Create a temporary config file with valid keys
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "snapsql.yaml")

	configContent := `
dialect: "postgres"
input_dir: "./queries"
generation:
  validate: true
  generators:
    json:
      output: "./generated"
      preserve_hierarchy: true
      settings:
        pretty: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Load config should succeed
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "postgres", config.Dialect)
	assert.Equal(t, "./queries", config.InputDir)
	assert.Equal(t, 3*time.Second, config.Performance.SlowQueryThreshold)
	assert.Equal(t, 0, len(config.Tables))

	// JSON generator should be enabled by default
	jsonGen := config.Generation.Generators["json"]
	assert.True(t, jsonGen.IsEnabled())
}

func TestValidateConfig_InvalidDialect(t *testing.T) {
	config := &Config{
		Dialect: "invalid_dialect",
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dialect")
}

func TestValidateConfig_InvalidGeneratorType(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Generation: GenerationConfig{
			Generators: map[string]GeneratorConfig{
				"unknown_lang": {
					Output:   "./generated",
					Disabled: boolPtr(false), // Enabled
				},
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown generator 'unknown_lang'")
}

func TestValidateConfig_MissingOutputWhenEnabled(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Generation: GenerationConfig{
			Generators: map[string]GeneratorConfig{
				"json": {
					Output:   "",
					Disabled: nil, // Enabled by default
				},
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output path is required when enabled")
}

func TestValidateConfig_InvalidSystemFieldParameter(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name: "test_field",
					OnInsert: SystemFieldOperation{
						Parameter: "invalid_parameter",
					},
				},
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid on_insert.parameter")
}

func TestValidateConfig_InvalidQueryTimeout(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Query: QueryConfig{
			Timeout: -1,
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.timeout must be non-negative")
}

func TestValidateConfig_InvalidDefaultFormat(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Query: QueryConfig{
			DefaultFormat: "invalid_format",
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.default_format")
}

func TestValidateConfig_InvalidSlowQueryThreshold(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Performance: PerformanceConfig{
			SlowQueryThreshold: -1 * time.Second,
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "performance.slow_query_threshold")
}

func TestValidateConfig_InvalidTableMetadata(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		Tables: map[string]TablePerformance{
			"public.users": {
				ExpectedRows: 0,
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tables.public.users.expected_rows")
}

func TestLoadConfig_PerformanceAndTables(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "snapsql.yaml")

	configContent := `
dialect: "postgres"
performance:
  slow_query_threshold: 2500ms
tables:
  public.users:
    expected_rows: 1500000
    allow_full_scan: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.NoError(t, err)

	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 2500*time.Millisecond, config.Performance.SlowQueryThreshold)
	meta, ok := config.Tables["public.users"]
	assert.True(t, ok)
	assert.Equal(t, int64(1500000), meta.ExpectedRows)
	assert.False(t, meta.AllowFullScan)
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	config := getDefaultConfig()

	err := validateConfig(config)
	assert.NoError(t, err)
}

func TestGeneratorConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		disabled *bool
		expected bool
	}{
		{
			name:     "explicitly disabled",
			disabled: boolPtr(true),
			expected: false,
		},
		{
			name:     "explicitly enabled (disabled: false)",
			disabled: boolPtr(false),
			expected: true,
		},
		{
			name:     "unset (nil) - enabled by default",
			disabled: nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := GeneratorConfig{
				Disabled: tt.disabled,
			}
			assert.Equal(t, tt.expected, gen.IsEnabled())
		})
	}
}

func TestLoadConfig_EnabledFlagDefaults(t *testing.T) {
	// Create a temporary config file without explicit disabled flags
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "snapsql.yaml")

	configContent := `
dialect: "postgres"
generation:
  generators:
    json:
      output: "./generated"
      # disabled is not specified, should default to enabled
    go:
      output: "./internal/queries"
      disabled: true
      # explicitly disabled
    typescript:
      output: "./src/generated"
      disabled: false
      # explicitly enabled
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(configPath)
	assert.NoError(t, err)

	// json generator should be enabled by default (nil or false)
	jsonGen := config.Generation.Generators["json"]
	assert.True(t, jsonGen.IsEnabled(), "json generator should be enabled when disabled is not specified")

	// go generator should be explicitly disabled
	goGen := config.Generation.Generators["go"]
	assert.False(t, goGen.IsEnabled(), "go generator should be disabled when disabled: true is specified")

	// typescript generator should be explicitly enabled
	tsGen := config.Generation.Generators["typescript"]
	assert.True(t, tsGen.IsEnabled(), "typescript generator should be enabled when disabled: false is specified")
}

func TestValidateConfig_EmptySystemFieldName(t *testing.T) {
	config := &Config{
		Dialect: "postgres",
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name: "", // empty name should cause error
				},
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}
