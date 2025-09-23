package main

import (
	"github.com/shibukawa/snapsql"
)

// Config is a type alias to maintain backward compatibility with snapsql.Config.
// Deprecated: use snapsql.Config directly in new code.
type Config = snapsql.Config
// Database is a type alias for snapsql.Database kept for CLI compatibility.
type Database = snapsql.Database
// SchemaExtractionConfig is a type alias for snapsql.SchemaExtractionConfig for legacy configs.
type SchemaExtractionConfig = snapsql.SchemaExtractionConfig
// TablePatterns is a type alias for snapsql.TablePatterns.
type TablePatterns = snapsql.TablePatterns
// GenerationConfig is a type alias for snapsql.GenerationConfig.
type GenerationConfig = snapsql.GenerationConfig
// GeneratorConfig is a type alias for snapsql.GeneratorConfig.
type GeneratorConfig = snapsql.GeneratorConfig
// LanguageConfig is a type alias for snapsql.LanguageConfig.
type LanguageConfig = snapsql.LanguageConfig
// ValidationConfig is a type alias for snapsql.ValidationConfig.
type ValidationConfig = snapsql.ValidationConfig
// QueryConfig is a type alias for snapsql.QueryConfig.
type QueryConfig = snapsql.QueryConfig
// SystemConfig is a type alias for snapsql.SystemConfig.
type SystemConfig = snapsql.SystemConfig
// SystemField is a type alias for snapsql.SystemField.
type SystemField = snapsql.SystemField

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	return snapsql.LoadConfig(configPath)
}
