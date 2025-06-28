package pull

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
	snapsql "github.com/shibukawa/snapsql"
)

// DatabaseConnector handles database connections and operations
type DatabaseConnector struct {
	poolSettings ConnectionPoolSettings
}

// ConnectionPoolSettings defines database connection pool configuration
type ConnectionPoolSettings struct {
	MaxOpenConns    int // Maximum number of open connections
	MaxIdleConns    int // Maximum number of idle connections
	ConnMaxLifetime int // Maximum lifetime of connections in seconds
}

// ConnectionInfo contains parsed database connection information
type ConnectionInfo struct {
	Type     string
	Host     string
	Port     string
	Database string
	Username string
	Password string
	Options  map[string]string
}

// PullOperation represents a complete pull operation
type PullOperation struct {
	Config    PullConfig
	connector *DatabaseConnector
	extractor Extractor
	generator *YAMLGenerator
}

// NewDatabaseConnector creates a new database connector with default settings
func NewDatabaseConnector() *DatabaseConnector {
	return &DatabaseConnector{
		poolSettings: ConnectionPoolSettings{
			MaxOpenConns:    25,
			MaxIdleConns:    25,
			ConnMaxLifetime: 300, // 5 minutes
		},
	}
}

// SetPoolSettings configures connection pool settings
func (c *DatabaseConnector) SetPoolSettings(settings ConnectionPoolSettings) {
	c.poolSettings = settings
}

// GetPoolSettings returns current connection pool settings
func (c *DatabaseConnector) GetPoolSettings() ConnectionPoolSettings {
	return c.poolSettings
}

// ParseDatabaseURL extracts database type from connection URL
func (c *DatabaseConnector) ParseDatabaseURL(databaseURL string) (string, error) {
	if databaseURL == "" {
		return "", ErrEmptyDatabaseURL
	}

	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", ErrInvalidDatabaseURL
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return "postgresql", nil
	case "mysql":
		return "mysql", nil
	case "sqlite", "sqlite3":
		return "sqlite", nil
	default:
		return "", ErrUnsupportedDatabase
	}
}

// ValidateConnectionString validates the format of a database connection string
func (c *DatabaseConnector) ValidateConnectionString(databaseURL string) error {
	if databaseURL == "" {
		return ErrEmptyDatabaseURL
	}

	u, err := url.Parse(databaseURL)
	if err != nil {
		return ErrInvalidDatabaseURL
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return c.validatePostgreSQLURL(u)
	case "mysql":
		return c.validateMySQLURL(u)
	case "sqlite", "sqlite3":
		return c.validateSQLiteURL(u)
	default:
		return ErrUnsupportedDatabase
	}
}

// Connect establishes a database connection
func (c *DatabaseConnector) Connect(databaseURL string) (*sql.DB, error) {
	if err := c.ValidateConnectionString(databaseURL); err != nil {
		return nil, err
	}

	dbType, err := c.ParseDatabaseURL(databaseURL)
	if err != nil {
		return nil, err
	}

	// Convert URL to driver-specific connection string
	connStr, err := c.convertToDriverString(databaseURL, dbType)
	if err != nil {
		return nil, err
	}

	// Open database connection
	db, err := sql.Open(c.getDriverName(dbType), connStr)
	if err != nil {
		return nil, ErrConnectionFailed
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.poolSettings.MaxOpenConns)
	db.SetMaxIdleConns(c.poolSettings.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(c.poolSettings.ConnMaxLifetime) * time.Second)

	return db, nil
}

// Close closes a database connection
func (c *DatabaseConnector) Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// Ping tests the database connection
func (c *DatabaseConnector) Ping(db *sql.DB) error {
	if db == nil {
		return ErrConnectionFailed
	}
	return db.Ping()
}

// ParseConnectionInfo parses a database URL into connection information
func (c *DatabaseConnector) ParseConnectionInfo(databaseURL string) (ConnectionInfo, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return ConnectionInfo{}, ErrInvalidDatabaseURL
	}

	info := ConnectionInfo{
		Options: make(map[string]string),
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		info.Type = "postgresql"
		info.Host = u.Hostname()
		info.Port = u.Port()
		if info.Port == "" {
			info.Port = "5432"
		}
		info.Database = strings.TrimPrefix(u.Path, "/")
		if u.User != nil {
			info.Username = u.User.Username()
			if password, ok := u.User.Password(); ok {
				info.Password = password
			}
		}
	case "mysql":
		info.Type = "mysql"
		info.Host = u.Hostname()
		info.Port = u.Port()
		if info.Port == "" {
			info.Port = "3306"
		}
		info.Database = strings.TrimPrefix(u.Path, "/")
		if u.User != nil {
			info.Username = u.User.Username()
			if password, ok := u.User.Password(); ok {
				info.Password = password
			}
		}
	case "sqlite", "sqlite3":
		info.Type = "sqlite"
		if u.Host == "" {
			// sqlite:///path/to/db.db format
			info.Database = u.Path
		} else {
			// sqlite://./db.db format
			info.Database = u.Host + u.Path
		}
	}

	// Parse query parameters as options
	for key, values := range u.Query() {
		if len(values) > 0 {
			info.Options[key] = values[0]
		}
	}

	return info, nil
}

// BuildConnectionString builds a connection URL from connection info
func (c *DatabaseConnector) BuildConnectionString(info ConnectionInfo) string {
	hostPort := net.JoinHostPort(info.Host, info.Port)

	switch info.Type {
	case "postgresql":
		if info.Password != "" {
			return fmt.Sprintf("postgres://%s:%s@%s/%s",
				info.Username, info.Password, hostPort, info.Database)
		}
		return fmt.Sprintf("postgres://%s@%s/%s",
			info.Username, hostPort, info.Database)
	case "mysql":
		if info.Password != "" {
			return fmt.Sprintf("mysql://%s:%s@%s/%s",
				info.Username, info.Password, hostPort, info.Database)
		}
		return fmt.Sprintf("mysql://%s@%s/%s",
			info.Username, hostPort, info.Database)
	case "sqlite":
		return fmt.Sprintf("sqlite://%s", info.Database)
	default:
		return ""
	}
}

// NewPullOperation creates a new pull operation
func NewPullOperation(config PullConfig) *PullOperation {
	return &PullOperation{
		Config:    config,
		connector: NewDatabaseConnector(),
	}
}

// ValidateConfig validates the pull configuration
func (p *PullOperation) ValidateConfig() error {
	if p.Config.DatabaseURL == "" {
		return ErrEmptyDatabaseURL
	}
	if p.Config.DatabaseType == "" {
		return ErrEmptyDatabaseType
	}
	if p.Config.OutputPath == "" {
		return ErrInvalidOutputPath
	}

	// Validate database URL format
	return p.connector.ValidateConnectionString(p.Config.DatabaseURL)
}

// Execute performs the complete pull operation
func (p *PullOperation) Execute() (*PullResult, error) {
	// Validate configuration
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Connect to database
	db, err := p.connector.Connect(p.Config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	defer p.connector.Close(db)

	// Test connection
	if err := p.connector.Ping(db); err != nil {
		return nil, ErrConnectionFailed
	}

	// Create extractor
	extractor, err := NewExtractor(p.Config.DatabaseType)
	if err != nil {
		return nil, err
	}
	p.extractor = extractor

	// Create extract config
	extractConfig := ExtractConfig{
		IncludeSchemas: p.Config.IncludeSchemas,
		ExcludeSchemas: p.Config.ExcludeSchemas,
		IncludeTables:  p.Config.IncludeTables,
		ExcludeTables:  p.Config.ExcludeTables,
		IncludeViews:   p.Config.IncludeViews,
		IncludeIndexes: p.Config.IncludeIndexes,
	}

	// Extract schemas
	schemas, err := p.extractor.ExtractSchemas(db, extractConfig)
	if err != nil {
		return nil, err
	}

	// Generate YAML files
	generator := NewYAMLGenerator(p.Config.SchemaAware)
	p.generator = generator

	if err := generator.Generate(schemas, p.Config.OutputPath); err != nil {
		return nil, err
	}

	// Create result
	return p.CreateResult(schemas, nil), nil
}

// CreateResult creates a pull result from schemas and errors
func (p *PullOperation) CreateResult(schemas []snapsql.DatabaseSchema, errors []error) *PullResult {
	var dbInfo snapsql.DatabaseInfo
	if len(schemas) > 0 {
		dbInfo = schemas[0].DatabaseInfo
	}

	return &PullResult{
		Schemas:      schemas,
		DatabaseInfo: dbInfo,
		Errors:       errors,
	}
}

// Helper methods for validation and conversion

func (c *DatabaseConnector) validatePostgreSQLURL(u *url.URL) error {
	if u.Host == "" {
		return ErrInvalidDatabaseURL
	}
	if strings.TrimPrefix(u.Path, "/") == "" {
		return ErrInvalidDatabaseURL
	}
	return nil
}

func (c *DatabaseConnector) validateMySQLURL(u *url.URL) error {
	if u.Host == "" {
		return ErrInvalidDatabaseURL
	}
	if strings.TrimPrefix(u.Path, "/") == "" {
		return ErrInvalidDatabaseURL
	}
	return nil
}

func (c *DatabaseConnector) validateSQLiteURL(u *url.URL) error {
	if u.Path == "" && u.Host == "" {
		return ErrInvalidDatabaseURL
	}
	return nil
}

func (c *DatabaseConnector) convertToDriverString(databaseURL, dbType string) (string, error) {
	info, err := c.ParseConnectionInfo(databaseURL)
	if err != nil {
		return "", err
	}

	switch dbType {
	case "postgresql":
		// pgx supports standard PostgreSQL connection strings
		if info.Host != "" && info.Database != "" {
			// Build standard PostgreSQL connection string
			connStr := fmt.Sprintf("postgres://%s", info.Host)
			if info.Port != "" {
				connStr += ":" + info.Port
			}
			connStr += "/" + info.Database

			// Add authentication if provided
			if info.Username != "" {
				auth := info.Username
				if info.Password != "" {
					auth += ":" + info.Password
				}
				connStr = fmt.Sprintf("postgres://%s@%s", auth, connStr[11:]) // Remove "postgres://"
			}

			// Add SSL mode if not specified
			if !strings.Contains(connStr, "sslmode=") {
				if strings.Contains(connStr, "?") {
					connStr += "&sslmode=disable"
				} else {
					connStr += "?sslmode=disable"
				}
			}

			return connStr, nil
		}
		return "", ErrInvalidConnectionInfo

	case "mysql":
		// Convert to go-sql-driver/mysql format
		connStr := ""
		if info.Username != "" {
			connStr += info.Username
			if info.Password != "" {
				connStr += ":" + info.Password
			}
			connStr += "@"
		}
		if info.Host != "" {
			connStr += "tcp(" + info.Host
			if info.Port != "" {
				connStr += ":" + info.Port
			}
			connStr += ")"
		}
		if info.Database != "" {
			connStr += "/" + info.Database
		}
		return connStr, nil

	case "sqlite":
		// SQLite uses file path directly
		return info.Database, nil

	default:
		return "", ErrUnsupportedDatabase
	}
}

func (c *DatabaseConnector) getDriverName(dbType string) string {
	switch dbType {
	case "postgresql":
		return "pgx"
	case "mysql":
		return "mysql"
	case "sqlite":
		return "sqlite3"
	default:
		return ""
	}
}

// ExecutePull is a convenience function that performs a complete pull operation
func ExecutePull(config PullConfig) (*PullResult, error) {
	operation := NewPullOperation(config)
	return operation.Execute()
}
