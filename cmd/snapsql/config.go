package main

import (
	"github.com/shibukawa/snapsql"
)

// Type aliases for backward compatibility
type Config = snapsql.Config
type Database = snapsql.Database
type SchemaExtractionConfig = snapsql.SchemaExtractionConfig
type TablePatterns = snapsql.TablePatterns
type GenerationConfig = snapsql.GenerationConfig
type GeneratorConfig = snapsql.GeneratorConfig
type LanguageConfig = snapsql.LanguageConfig
type ValidationConfig = snapsql.ValidationConfig
type QueryConfig = snapsql.QueryConfig
type SystemConfig = snapsql.SystemConfig
type SystemField = snapsql.SystemField

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	return snapsql.LoadConfig(configPath)
}
