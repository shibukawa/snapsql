package pull

import (
	"time"

	snapsql "github.com/shibukawa/snapsql"
)

// OutputFormat defines the output format for schema files
type OutputFormat string

const (
	OutputSingleFile OutputFormat = "single"
	OutputPerTable   OutputFormat = "per_table"
	OutputPerSchema  OutputFormat = "per_schema"
)

// PullConfig contains configuration for the pull operation
type PullConfig struct {
	DatabaseURL    string
	DatabaseType   string
	OutputPath     string
	OutputFormat   OutputFormat
	SchemaAware    bool     // Enable schema-aware directory structure
	IncludeSchemas []string // Schema filter (PostgreSQL/MySQL)
	ExcludeSchemas []string // Schema exclusion (PostgreSQL/MySQL)
	IncludeTables  []string
	ExcludeTables  []string
	IncludeViews   bool
	IncludeIndexes bool
}

// PullResult contains the result of a pull operation
type PullResult struct {
	Schemas      []snapsql.DatabaseSchema
	ExtractedAt  time.Time
	DatabaseInfo snapsql.DatabaseInfo
	Errors       []error
}

// ExtractConfig contains configuration for schema extraction
type ExtractConfig struct {
	IncludeSchemas []string // Schema filter (PostgreSQL/MySQL)
	ExcludeSchemas []string // Schema exclusion (PostgreSQL/MySQL)
	IncludeTables  []string
	ExcludeTables  []string
	IncludeViews   bool
	IncludeIndexes bool
}

// Pull performs the database schema extraction operation
func Pull(config PullConfig) (*PullResult, error) {
	// This is a placeholder implementation for testing
	// The actual implementation will be added in the next phase
	return &PullResult{
		Schemas:     []snapsql.DatabaseSchema{},
		ExtractedAt: time.Now(),
		DatabaseInfo: snapsql.DatabaseInfo{
			Type: config.DatabaseType,
		},
		Errors: []error{},
	}, nil
}
