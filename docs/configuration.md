# Configuration

SnapSQL projects are configured using a `snapsql.yaml` file in the project root.

## Project Structure

```
my-project/
├── snapsql.yaml          # Main configuration file
├── queries/              # SQL template files
│   ├── users.snap.sql
│   └── posts.snap.sql
├── params.json          # Default parameters (optional)
├── constants.yaml       # Project constants (optional)
└── generated/           # Generated intermediate files
    └── queries.json
```

## Configuration File (snapsql.yaml)

### Basic Configuration

```yaml
# Project metadata
name: "my-project"
version: "1.0.0"
description: "My SnapSQL project"

# Database configuration
database:
  default_driver: "postgres"
  connection_string: "${DATABASE_URL}"
  timeout: "30s"

# Query settings
query:
  execute_dangerous_query: false
  default_format: "table"
  
# File paths
paths:
  queries: "./queries"
  generated: "./generated"
  constants: "./constants.yaml"
  params: "./params.json"
```

### Database Configuration

#### PostgreSQL

```yaml
database:
  default_driver: "postgres"
  connection_string: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
  timeout: "30s"
```

#### MySQL

```yaml
database:
  default_driver: "mysql"
  connection_string: "user:password@tcp(localhost:3306)/dbname"
  timeout: "30s"
```

#### SQLite

```yaml
database:
  default_driver: "sqlite3"
  connection_string: "./database.db"
  timeout: "30s"
```

### Environment Variables

Use environment variables in configuration:

```yaml
database:
  connection_string: "${DATABASE_URL}"
  
# With defaults
database:
  connection_string: "${DATABASE_URL:-postgres://localhost:5432/mydb}"
```

### Query Settings

```yaml
query:
  # Allow dangerous queries (DELETE/UPDATE without WHERE)
  execute_dangerous_query: false
  
  # Default output format (table, json, csv)
  default_format: "table"
  
  # Default timeout for queries
  timeout: "30s"
  
  # Maximum rows to return
  max_rows: 1000
```

### File Paths

```yaml
paths:
  # Directory containing .snap.sql files
  queries: "./queries"
  
  # Directory for generated intermediate files
  generated: "./generated"
  
  # Constants file
  constants: "./constants.yaml"
  
  # Default parameters file
  params: "./params.json"
```

## Constants File (constants.yaml)

Define project-wide constants:

```yaml
# Table name mappings
tables:
  users: "users_v2"
  posts: "posts_archive"
  comments: "comments_new"

# Environment-specific prefixes
environments:
  dev: "dev_"
  staging: "staging_"
  prod: "prod_"

# Common values
pagination:
  default_limit: 50
  max_limit: 1000

# Feature flags
features:
  enable_caching: true
  enable_analytics: false
```

## Parameters File (params.json)

Default parameters for development:

```json
{
  "environment": "dev",
  "pagination": {
    "limit": 20,
    "offset": 0
  },
  "filters": {
    "active": true
  },
  "include_email": true,
  "table_suffix": "dev"
}
```

## Environment-Specific Configuration

### Using Multiple Config Files

```bash
# Development
snapsql --config snapsql.dev.yaml query users.snap.sql

# Production
snapsql --config snapsql.prod.yaml query users.snap.sql
```

### Environment Variables

```yaml
# snapsql.yaml
database:
  connection_string: "${DATABASE_URL}"
  
query:
  execute_dangerous_query: "${ALLOW_DANGEROUS_QUERIES:-false}"
```

```bash
# Set environment variables
export DATABASE_URL="postgres://localhost:5432/mydb"
export ALLOW_DANGEROUS_QUERIES="true"

# Run query
snapsql query users.snap.sql
```

## Advanced Configuration

### Custom Dialects

```yaml
database:
  dialect: "postgresql"
  features:
    - "window_functions"
    - "cte"
    - "json_operators"
```

### Logging

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json" # json, text
  output: "stdout" # stdout, stderr, file path
```

### Performance

```yaml
performance:
  connection_pool_size: 10
  max_idle_connections: 5
  connection_max_lifetime: "1h"
  query_timeout: "30s"
```

## Configuration Validation

Validate your configuration:

```bash
# Check configuration syntax
snapsql config validate

# Show resolved configuration
snapsql config show

# Test database connection
snapsql config test-db
```

## Examples

### Development Setup

```yaml
# snapsql.dev.yaml
name: "myapp-dev"
database:
  default_driver: "postgres"
  connection_string: "postgres://dev:dev@localhost:5432/myapp_dev"
query:
  execute_dangerous_query: true
  default_format: "table"
paths:
  queries: "./queries"
  constants: "./dev-constants.yaml"
```

### Production Setup

```yaml
# snapsql.prod.yaml
name: "myapp-prod"
database:
  default_driver: "postgres"
  connection_string: "${DATABASE_URL}"
query:
  execute_dangerous_query: false
  default_format: "json"
  max_rows: 10000
paths:
  queries: "./queries"
  constants: "./prod-constants.yaml"
logging:
  level: "warn"
  format: "json"
```
