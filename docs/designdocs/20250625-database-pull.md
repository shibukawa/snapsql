# Database Pull Feature Design Document

## Overview

The database pull feature enables SnapSQL to extract schema information from existing databases (PostgreSQL, MySQL, SQLite) and generate YAML schema files. This feature supports the development workflow by providing accurate type information for SQL template generation and validation.

## Requirements

### Functional Requirements

1. **Database Connection Support**
   - PostgreSQL: Connect via standard connection strings
   - MySQL: Connect via standard connection strings  
   - SQLite: Connect to local database files
   - Support for connection parameters from snapsql.yaml configuration

2. **Schema Information Extraction**
   - Table names and schemas
   - Column information (name, type, nullable, default values)
   - Primary key constraints
   - Foreign key relationships
   - Unique constraints
   - Index information
   - Table comments and column comments

3. **YAML Schema Generation**
   - Generate structured YAML files with extracted schema information
   - Support for multiple output formats (single file vs. per-table files)
   - Include metadata such as extraction timestamp and database version

4. **Type Mapping**
   - Map database-specific types to SnapSQL standard types
   - Handle database dialect differences appropriately
   - Preserve original type information for reference

### Non-Functional Requirements

1. **Performance**
   - Efficient schema extraction using system tables/information_schema
   - Minimal impact on target database performance
   - Support for large databases with hundreds of tables

2. **Security**
   - Read-only database access
   - Secure credential handling
   - No modification of target database

3. **Reliability**
   - Graceful handling of connection failures
   - Partial extraction support (continue on individual table errors)
   - Comprehensive error reporting

## Architecture

### Package Structure

```
pull/
├── pull.go              // Main pull orchestrator
├── connector.go         // Database connection management
├── extractor.go         // Schema extraction interface
├── postgresql.go        // PostgreSQL-specific extraction
├── mysql.go            // MySQL-specific extraction
├── sqlite.go           // SQLite-specific extraction
├── schema.go           // Schema data structures
├── yaml_generator.go   // YAML output generation
└── type_mapper.go      // Database type mapping
```

### Core Components

#### 1. Pull Orchestrator (`pull.go`)

Main entry point that coordinates the entire pull process:

```go
type PullConfig struct {
    DatabaseURL     string
    DatabaseType    string
    OutputPath      string
    OutputFormat    OutputFormat
    SchemaAware     bool     // Enable schema-aware directory structure
    IncludeSchemas  []string // Schema filter (PostgreSQL/MySQL)
    ExcludeSchemas  []string // Schema exclusion (PostgreSQL/MySQL)
    IncludeTables   []string
    ExcludeTables   []string
    IncludeViews    bool
    IncludeIndexes  bool
}

type PullResult struct {
    Schemas       []DatabaseSchema
    ExtractedAt   time.Time
    DatabaseInfo  DatabaseInfo
    Errors        []error
}

func Pull(config PullConfig) (*PullResult, error)
```

#### 2. Database Connector (`connector.go`)

Manages database connections with proper resource cleanup:

```go
type Connector interface {
    Connect(url string) (*sql.DB, error)
    GetDatabaseInfo(db *sql.DB) (DatabaseInfo, error)
    Close() error
}

type DatabaseInfo struct {
    Type     string
    Version  string
    Name     string
    Charset  string
}
```

#### 3. Schema Extractor (`extractor.go`)

Database-agnostic interface for schema extraction:

```go
type Extractor interface {
    ExtractSchemas(db *sql.DB, config ExtractConfig) ([]DatabaseSchema, error)
    ExtractTables(db *sql.DB, schemaName string) ([]TableSchema, error)
    ExtractColumns(db *sql.DB, tableName string) ([]ColumnSchema, error)
    ExtractConstraints(db *sql.DB, tableName string) ([]ConstraintSchema, error)
    ExtractIndexes(db *sql.DB, tableName string) ([]IndexSchema, error)
}

type ExtractConfig struct {
    IncludeSchemas []string // Schema filter (PostgreSQL/MySQL)
    ExcludeSchemas []string // Schema exclusion (PostgreSQL/MySQL)
    IncludeTables  []string
    ExcludeTables  []string
    IncludeViews   bool
    IncludeIndexes bool
}
```

#### 4. Schema Data Structures (`schema.go`)

Unified schema representation:

```go
type DatabaseSchema struct {
    Name        string        `yaml:"name"`
    Tables      []TableSchema `yaml:"tables"`
    Views       []ViewSchema  `yaml:"views,omitempty"`
    ExtractedAt time.Time     `yaml:"extracted_at"`
    DatabaseInfo DatabaseInfo `yaml:"database_info"`
}

type TableSchema struct {
    Name        string           `yaml:"name"`
    Schema      string           `yaml:"schema,omitempty"`
    Columns     []ColumnSchema   `yaml:"columns"`
    Constraints []ConstraintSchema `yaml:"constraints,omitempty"`
    Indexes     []IndexSchema    `yaml:"indexes,omitempty"`
    Comment     string           `yaml:"comment,omitempty"`
}

type ColumnSchema struct {
    Name         string `yaml:"name"`
    Type         string `yaml:"type"`
    SnapSQLType  string `yaml:"snapsql_type"`
    Nullable     bool   `yaml:"nullable"`
    DefaultValue string `yaml:"default_value,omitempty"`
    Comment      string `yaml:"comment,omitempty"`
    IsPrimaryKey bool   `yaml:"is_primary_key,omitempty"`
}

type ConstraintSchema struct {
    Name           string   `yaml:"name"`
    Type           string   `yaml:"type"` // PRIMARY_KEY, FOREIGN_KEY, UNIQUE, CHECK
    Columns        []string `yaml:"columns"`
    ReferencedTable string  `yaml:"referenced_table,omitempty"`
    ReferencedColumns []string `yaml:"referenced_columns,omitempty"`
    Definition     string   `yaml:"definition,omitempty"`
}

type IndexSchema struct {
    Name     string   `yaml:"name"`
    Columns  []string `yaml:"columns"`
    IsUnique bool     `yaml:"is_unique"`
    Type     string   `yaml:"type,omitempty"`
}

type ViewSchema struct {
    Name       string `yaml:"name"`
    Schema     string `yaml:"schema,omitempty"`
    Definition string `yaml:"definition"`
    Comment    string `yaml:"comment,omitempty"`
}
```

#### 5. Database-Specific Extractors

Each database has its own extractor implementation:

**PostgreSQL Extractor (`postgresql.go`)**
- Uses `information_schema` and `pg_catalog` system tables
- Handles PostgreSQL-specific types (arrays, JSON, custom types)
- Extracts schema-qualified table names

**MySQL Extractor (`mysql.go`)**
- Uses `information_schema` tables
- Handles MySQL-specific types and storage engines
- Supports both MySQL and MariaDB variants

**SQLite Extractor (`sqlite.go`)**
- Uses `sqlite_master` and `PRAGMA` statements
- Handles SQLite's dynamic typing system
- Extracts table and index information

#### 6. Type Mapper (`type_mapper.go`)

Maps database-specific types to SnapSQL standard types:

```go
type TypeMapper interface {
    MapType(dbType string) string
    GetSnapSQLType(dbType string) string
}

// Standard SnapSQL types
const (
    TypeString   = "string"
    TypeInt      = "int"
    TypeFloat    = "float"
    TypeBool     = "bool"
    TypeDate     = "date"
    TypeTime     = "time"
    TypeDateTime = "datetime"
    TypeJSON     = "json"
    TypeArray    = "array"
    TypeBinary   = "binary"
)
```

#### 7. YAML Generator (`yaml_generator.go`)

Generates YAML output in various formats:

```go
type OutputFormat string

const (
    OutputSingleFile OutputFormat = "single"
    OutputPerTable   OutputFormat = "per_table"
    OutputPerSchema  OutputFormat = "per_schema"
)

type YAMLGenerator struct {
    Format OutputFormat
    Pretty bool
    SchemaAware bool  // Enable schema-aware directory structure
    FlowStyle   bool  // Use flow style for columns, constraints, indexes
}

func (g *YAMLGenerator) Generate(schemas []DatabaseSchema, outputPath string) error

// Schema-aware path generation
func (g *YAMLGenerator) getSchemaPath(outputPath, schemaName string) string {
    if !g.SchemaAware {
        return outputPath
    }
    
    // Use 'global' for empty or default schemas
    if schemaName == "" || schemaName == "main" {
        schemaName = "global"
    }
    
    return filepath.Join(outputPath, schemaName)
}

// Configure YAML encoder for mixed block/flow style
func (g *YAMLGenerator) configureEncoder(writer io.Writer) *yaml.Encoder {
    encoder := yaml.NewEncoder(writer)
    if g.FlowStyle {
        // Configure to use flow style for specific fields
        encoder.SetIndent(2)
        // Note: go-yaml supports flow style through struct tags or custom marshaling
    }
    return encoder
}
}
```

## Database-Specific Implementation Details

### PostgreSQL

**System Tables Used:**
- `information_schema.tables` - Table information
- `information_schema.columns` - Column information
- `information_schema.table_constraints` - Constraint information
- `information_schema.key_column_usage` - Key relationships
- `pg_catalog.pg_indexes` - Index information
- `pg_catalog.pg_description` - Comments

**Key Queries:**
```sql
-- Schemas
SELECT schema_name FROM information_schema.schemata 
WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast');

-- Tables with schema
SELECT schemaname, tablename, tableowner 
FROM pg_tables 
WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
ORDER BY schemaname, tablename;

-- Columns with types
SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
       c.character_maximum_length, c.numeric_precision, c.numeric_scale
FROM information_schema.columns c
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position;
```

### MySQL

**System Tables Used:**
- `information_schema.TABLES` - Table information
- `information_schema.COLUMNS` - Column information
- `information_schema.TABLE_CONSTRAINTS` - Constraint information
- `information_schema.KEY_COLUMN_USAGE` - Key relationships
- `information_schema.STATISTICS` - Index information

**Key Queries:**
```sql
-- Schemas
SELECT SCHEMA_NAME FROM information_schema.SCHEMATA
WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys');

-- Tables with schema
SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE, TABLE_COMMENT
FROM information_schema.TABLES
WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
ORDER BY TABLE_SCHEMA, TABLE_NAME;

-- Columns
SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT,
       CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE,
       COLUMN_KEY, EXTRA, COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION;
```

### SQLite

**System Tables Used:**
- `sqlite_master` - Main system table
- `PRAGMA table_info()` - Column information
- `PRAGMA foreign_key_list()` - Foreign key information
- `PRAGMA index_list()` - Index information

**Key Queries:**
```sql
-- Tables (SQLite uses 'main' as default schema, map to 'global')
SELECT name, type, sql FROM sqlite_master 
WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%';

-- Columns (via PRAGMA)
PRAGMA table_info(table_name);

-- Foreign keys (via PRAGMA)
PRAGMA foreign_key_list(table_name);

-- Note: SQLite schema will be mapped to 'global' directory
```

## Type Mapping Strategy

### PostgreSQL Type Mapping

| PostgreSQL Type | SnapSQL Type | Notes |
|----------------|--------------|-------|
| `varchar`, `text`, `char` | `string` | |
| `integer`, `bigint`, `smallint` | `int` | |
| `numeric`, `decimal`, `real`, `double precision` | `float` | |
| `boolean` | `bool` | |
| `date` | `date` | |
| `time`, `timetz` | `time` | |
| `timestamp`, `timestamptz` | `datetime` | |
| `json`, `jsonb` | `json` | |
| `array types` | `array` | With element type |
| `bytea` | `binary` | |

### MySQL Type Mapping

| MySQL Type | SnapSQL Type | Notes |
|------------|--------------|-------|
| `varchar`, `text`, `char` | `string` | |
| `int`, `bigint`, `smallint`, `tinyint` | `int` | |
| `decimal`, `numeric`, `float`, `double` | `float` | |
| `boolean`, `tinyint(1)` | `bool` | |
| `date` | `date` | |
| `time` | `time` | |
| `datetime`, `timestamp` | `datetime` | |
| `json` | `json` | |
| `blob`, `binary`, `varbinary` | `binary` | |

### SQLite Type Mapping

| SQLite Type | SnapSQL Type | Notes |
|-------------|--------------|-------|
| `TEXT` | `string` | |
| `INTEGER` | `int` | |
| `REAL` | `float` | |
| `BLOB` | `binary` | |
| Dynamic types | `string` | Default fallback |

## Output Format Examples

### Single File Format

```yaml
# .snapsql/schema/database_schema.yaml
database_info:
  type: postgresql
  version: "14.2"
  name: myapp_production
  charset: UTF8

extracted_at: 2025-06-25T23:00:00Z

schemas:
  - name: public
    tables:
      - name: users
        columns:
          - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
          - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
          - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false, default_value: "now()"}
        constraints:
          - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
          - {name: users_email_unique, type: UNIQUE, columns: [email]}
        indexes:
          - {name: idx_users_email, columns: [email], is_unique: true}
```

### Per-Table Format

```yaml
# .snapsql/schema/public/users.yaml
table:
  name: users
  schema: public
  columns:
    - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
    - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
    - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false, default_value: "now()"}
    - {name: updated_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: true}
  constraints:
    - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
    - {name: users_email_unique, type: UNIQUE, columns: [email]}
  indexes:
    - {name: idx_users_email, columns: [email], is_unique: true}
    - {name: idx_users_created_at, columns: [created_at], is_unique: false}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

### Per-Schema Format

```yaml
# .snapsql/schema/public.yaml
schema:
  name: public
  tables:
    - name: users
      columns:
        - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
        - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
        - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false}
      constraints:
        - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
        - {name: users_email_unique, type: UNIQUE, columns: [email]}
    - name: posts
      columns:
        - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
        - {name: title, type: "character varying(255)", snapsql_type: string, nullable: false}
        - {name: user_id, type: integer, snapsql_type: int, nullable: false}
        - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false}
      constraints:
        - {name: posts_pkey, type: PRIMARY_KEY, columns: [id]}
        - {name: posts_user_id_fkey, type: FOREIGN_KEY, columns: [user_id], referenced_table: users, referenced_columns: [id]}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

### Directory Structure

The recommended directory structure for schema files with database schema support:

```
project/
├── .snapsql/
│   └── schema/
│       ├── public/           # PostgreSQL/MySQL schema
│       │   ├── users.yaml
│       │   ├── posts.yaml
│       │   └── comments.yaml
│       ├── auth/             # Another schema
│       │   ├── sessions.yaml
│       │   └── permissions.yaml
│       └── global/           # SQLite or schema-less databases
│           ├── users.yaml
│           └── posts.yaml
├── queries/
│   ├── users.snap.sql
│   └── posts.snap.sql
└── snapsql.yaml
```

**Schema Directory Mapping:**
- **PostgreSQL/MySQL**: Use actual schema names (`public`, `auth`, `inventory`, etc.)
- **SQLite**: Use `global` as the default schema directory
- **Schema-less databases**: Use `global` as the fallback schema directory

**Benefits of schema-aware structure:**
- **Database Fidelity**: Mirrors actual database schema organization
- **Namespace Clarity**: Clear separation of tables by database schema
- **Multi-Schema Support**: Easy management of complex databases with multiple schemas
- **Conflict Resolution**: Handles tables with same names in different schemas
- **Migration Friendly**: Easier to track schema-specific changes

## Error Handling Strategy

### Connection Errors
- Retry logic with exponential backoff
- Clear error messages for common connection issues
- Support for connection timeout configuration

### Extraction Errors
- Continue processing other tables when individual table extraction fails
- Collect and report all errors at the end
- Provide detailed error context (table name, query, etc.)

### Type Mapping Errors
- Fallback to string type for unknown database types
- Log warnings for unmapped types
- Allow custom type mapping configuration

## Configuration Integration

### snapsql.yaml Configuration

```yaml
databases:
  development:
    driver: postgres
    connection: "postgres://user:pass@localhost/myapp_dev"
  production:
    driver: postgres
    connection: "postgres://user:pass@prod.example.com/myapp"

pull:
  output_format: per_table     # single, per_table, per_schema
  output_path: ".snapsql/schema"
  schema_aware: true           # Enable schema-aware directory structure
  flow_style: true             # Use flow style for columns, constraints, indexes
  include_views: true
  include_indexes: true
  include_schemas: ["public", "auth"]      # optional schema filter
  exclude_schemas: ["information_schema"]  # optional schema exclusion
  include_tables: ["users", "posts", "comments"]  # optional table filter
  exclude_tables: ["migrations", "temp_*"]        # optional table filter
```

## CLI Integration

The pull functionality will be integrated into the main SnapSQL CLI:

```bash
# Pull from configured database (schema-aware by default)
snapsql pull --database development

# Pull with custom connection
snapsql pull --url "postgres://user:pass@localhost/myapp"

# Pull specific schemas
snapsql pull --database production --schemas public,auth

# Pull specific tables from specific schema
snapsql pull --database production --schema public --tables users,posts,comments

# Pull to specific output path (defaults to .snapsql/schema)
snapsql pull --database development --output .snapsql/custom_schema

# Pull with different format
snapsql pull --database development --format per_schema

# Disable schema-aware structure (flat structure)
snapsql pull --database development --no-schema-aware
```

## Testing Strategy

### Unit Tests
- Test each extractor implementation with mock databases
- Test type mapping for all supported database types
- Test YAML generation with various schema configurations
- Test error handling scenarios

### Integration Tests
- Use TestContainers for PostgreSQL and MySQL testing
- Use in-memory SQLite databases for SQLite testing
- Test with real database schemas of varying complexity
- Test performance with large schemas (100+ tables)

### Test Data
- Create representative test schemas for each database type
- Include edge cases (unusual types, complex constraints, etc.)
- Test with both simple and complex database structures

## Performance Considerations

### Query Optimization
- Use batch queries where possible to reduce round trips
- Implement connection pooling for multiple schema extraction
- Use prepared statements for repeated queries

### Memory Management
- Stream large result sets instead of loading everything into memory
- Implement pagination for databases with many tables
- Use efficient data structures for schema representation

### Caching
- Cache database metadata to avoid repeated queries
- Support incremental updates (only extract changed tables)
- Implement schema comparison for change detection

## Security Considerations

### Database Access
- Use read-only database connections
- Support for limited database user permissions
- No modification of target database structure or data

### Credential Management
- Support for environment variable expansion in connection strings
- Integration with external credential management systems
- Secure handling of connection strings in memory

### Output Security
- Sanitize sensitive information in generated YAML
- Support for excluding sensitive tables/columns
- Option to obfuscate schema names in output

## Future Enhancements

### Advanced Features
- Support for database views and materialized views
- Extraction of stored procedures and functions
- Support for database triggers and events
- Custom type definitions and domains

### Integration Features
- Integration with schema migration tools
- Support for schema versioning and change tracking
- Integration with documentation generation tools
- Support for schema validation and linting

### Performance Features
- Parallel extraction for multiple schemas
- Incremental extraction based on database change logs
- Compression of large schema files
- Support for schema caching and persistence

## Implementation Phases

### Phase 1: Core Infrastructure
- Implement basic package structure
- Create database connector interface
- Implement PostgreSQL extractor
- Basic YAML generation

### Phase 2: Multi-Database Support
- Implement MySQL extractor
- Implement SQLite extractor
- Add type mapping system
- Comprehensive error handling

### Phase 3: Advanced Features
- Multiple output formats
- Configuration integration
- CLI command implementation
- Performance optimizations

### Phase 4: Testing and Polish
- Comprehensive test suite
- Performance testing
- Documentation and examples
- Security review

## Dependencies

### External Libraries
- Database drivers: `lib/pq` (PostgreSQL), `go-sql-driver/mysql` (MySQL), `modernc.org/sqlite` (SQLite)
- YAML processing: `github.com/goccy/go-yaml` (consistent with existing codebase)
- Configuration: Integration with existing snapsql.yaml handling
- Testing: TestContainers for integration tests

### Internal Dependencies
- Configuration system (snapsql.yaml parsing)
- CLI framework (Kong-based command structure)
- Error handling patterns (sentinel errors)
- Logging and output formatting

This design provides a comprehensive foundation for implementing the database pull feature while maintaining consistency with the existing SnapSQL architecture and coding standards.
