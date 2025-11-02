# Database Pull Feature Design Document

## Overview

The database pull feature extracts schema information from existing databases (PostgreSQL, MySQL, SQLite) and generates YAML schema files. This functionality provides accurate type information for SQL template generation and validation, supporting the development workflow.

## Process Flow

1. **Database Connection**
   - Retrieve connection information from `.tbls.yaml` or command line arguments (`--db` / `--url`)
   - Select appropriate driver based on database type
   - Test connection and obtain basic information (version, character set)

2. **Schema Information Extraction**
   - Access system tables based on database type
   - Retrieve table list (with filtering)
   - Get column information for each table
   - Extract constraint information (primary keys, foreign keys, unique constraints)
   - Obtain index information
   - Retrieve comment information (where supported)

3. **Type Mapping Processing**
   - Map database-specific types to SnapSQL standard types
   - Handle custom types (fallback to string type)
   - Process special types (arrays, JSON)

4. **YAML Generation**
   - Determine output format (single file, per table, per schema)
   - Add metadata (extraction timestamp, database information)
   - Generate files to specified output path

## Database-Specific Processing

### PostgreSQL
1. Retrieve table information from information_schema
2. Get index and constraint information from pg_catalog
3. Extract comment information from pg_description
4. Handle PostgreSQL-specific types (arrays, JSON)

### MySQL
1. Get table and column information from information_schema
2. Retrieve constraint information from KEY_COLUMN_USAGE
3. Extract index information from STATISTICS
4. Handle MySQL-specific types and storage engines

### SQLite
1. Get table information from sqlite_master
2. Retrieve column information using PRAGMA table_info()
3. Get foreign key information using PRAGMA foreign_key_list()
4. Extract index information using PRAGMA index_list()

## Output Format

### Single File Format
```yaml
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
          - {name: created_at, type: "timestamp with time zone", snapsql_type: timestamp, nullable: false}
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
  constraints:
    - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
    - {name: users_email_unique, type: UNIQUE, columns: [email]}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

## Type Mapping

### Common Type Mapping
| Database Type | SnapSQL Type |
|--------------|-------------|
| String types | string |
| Integer types | int |
| Floating point types | float |
| Boolean types | bool |
| Timestamp types | timestamp (aliases: datetime, date, time) |
| JSON types | json |
| Binary types | binary |

### Database-Specific Type Mapping
- PostgreSQL: Array types, Custom types
- MySQL: ENUM types, SET types
- SQLite: Dynamic type system

## Error Handling

1. **Connection Errors**
   - Connection timeouts
   - Authentication errors
   - Network errors

2. **Extraction Errors**
   - Access permission errors
   - System table access errors
   - Metadata retrieval errors

3. **Type Mapping Errors**
   - Unknown type handling
   - Custom type processing
   - Type conversion errors

## CLI Interface

```bash
# Basic usage
snapsql pull --database development

# Custom connection
snapsql pull --url "postgres://user:pass@localhost/myapp"

# Output format specification
snapsql pull --database development --format per_table

# Filtering
snapsql pull --database production --schemas public,auth --tables users,posts
```

pull:
## Configuration

Connection information for the pull command should be provided via `.tbls.yaml` or the `--db`/`--url` flags. Pull-specific options can be supplied via CLI flags. Example usage:

```bash
snapsql pull --db "postgres://user:pass@localhost/myapp_dev" --output .snapsql/schema --include-views --include-indexes
```

## Security Considerations

1. **Database Access**
   - Use of read-only connections
   - Application of principle of least privilege
   - Restricted access to system tables

2. **Credential Management**
   - Reading credentials from environment variables
   - Secure handling of connection strings
   - Protection of credentials in memory

3. **Output File Protection**
   - Filtering of sensitive information
   - Setting appropriate file permissions
   - Output directory restrictions
