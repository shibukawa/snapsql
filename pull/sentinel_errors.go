package pull

import "errors"

// Connection errors
var (
	ErrConnectionFailed    = errors.New("failed to connect to database")
	ErrInvalidDatabaseURL  = errors.New("invalid database URL")
	ErrUnsupportedDatabase = errors.New("unsupported database type")
	ErrDatabaseTimeout     = errors.New("database connection timeout")
)

// Schema extraction errors
var (
	ErrSchemaNotFound     = errors.New("schema not found")
	ErrTableNotFound      = errors.New("table not found")
	ErrColumnNotFound     = errors.New("column not found")
	ErrConstraintNotFound = errors.New("constraint not found")
	ErrIndexNotFound      = errors.New("index not found")
	ErrViewNotFound       = errors.New("view not found")
)

// Configuration errors
var (
	ErrInvalidOutputFormat      = errors.New("invalid output format")
	ErrInvalidOutputPath        = errors.New("invalid output path")
	ErrEmptyDatabaseURL         = errors.New("database URL cannot be empty")
	ErrEmptyDatabaseType        = errors.New("database type cannot be empty")
	ErrInvalidConnectionInfo    = errors.New("invalid connection info")
	ErrConflictingSchemaFilters = errors.New("conflicting schema filters: same schema in both include and exclude lists")
	ErrConflictingTableFilters  = errors.New("conflicting table filters: same table in both include and exclude lists")
)

// YAML generation errors
var (
	ErrYAMLGenerationFailed  = errors.New("YAML generation failed")
	ErrFileWriteFailed       = errors.New("failed to write file")
	ErrDirectoryCreateFailed = errors.New("failed to create directory")
)

// Type mapping errors
var (
	ErrUnknownDatabaseType = errors.New("unknown database type")
	ErrTypeMappingFailed   = errors.New("type mapping failed")
)

// Query execution errors
var (
	ErrQueryExecutionFailed = errors.New("query execution failed")
	ErrResultScanFailed     = errors.New("result scan failed")
	ErrInvalidQueryResult   = errors.New("invalid query result")
)
