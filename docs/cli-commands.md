# CLI Commands

The `snapsql` command-line tool provides various commands for working with SnapSQL projects.

## Global Options

```bash
snapsql [global options] <command> [command options]
```

### Global Flags

- `--config <file>` - Configuration file path (default: `snapsql.yaml`)
- `--verbose` - Enable verbose output
- `--quiet` - Suppress non-essential output
- `--help` - Show help information

## Commands

### init - Initialize Project

Create a new SnapSQL project with sample files.

```bash
snapsql init <project-name>
```

**Options:**
- `--template <name>` - Use a specific project template
- `--database <driver>` - Set default database driver (postgres, mysql, sqlite3)

**Example:**
```bash
# Create a new project
snapsql init my-project

# Create with PostgreSQL template
snapsql init my-project --database postgres
```

### generate - Generate Intermediate Files

Process SQL templates and generate intermediate JSON files.

```bash
snapsql generate [options]
```

**Options:**
- `--output <dir>` - Output directory for generated files (default: `./generated`)
- `--watch` - Watch for file changes and regenerate automatically
- `--force` - Overwrite existing generated files

**Example:**
```bash
# Generate all templates
snapsql generate

# Generate with custom output directory
snapsql generate --output ./build

# Watch for changes
snapsql generate --watch
```

### query - Execute Query

Execute a SQL template with parameters.

```bash
snapsql query <template-file> [options]
```

**Options:**
- `--dry-run` - Show generated SQL without executing
- `--params-file <file>` - Load parameters from JSON/YAML file
- `--param <key=value>` - Set individual parameter (can be used multiple times)
- `--output <file>` - Write results to file instead of stdout
- `--format <format>` - Output format: `table`, `json`, `csv` (default: `table`)
- `--limit <n>` - Limit number of rows returned
- `--offset <n>` - Offset for result set
- `--timeout <duration>` - Query timeout (e.g., `30s`, `5m`)
- `--explain` - Show query execution plan
- `--explain-analyze` - Show detailed execution plan with statistics
- `--execute-dangerous-query` - Allow DELETE/UPDATE without WHERE clause

**Examples:**
```bash
# Dry run to see generated SQL
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# Execute with parameters
snapsql query queries/users.snap.sql --param active=true --param limit=50

# Output as JSON
snapsql query queries/users.snap.sql --format json --output results.json

# Show execution plan
snapsql query queries/users.snap.sql --explain --params-file params.json
```

### validate - Validate Templates

Validate SQL templates for syntax and parameter consistency.

```bash
snapsql validate [template-files...]
```

**Options:**
- `--all` - Validate all templates in the project
- `--strict` - Enable strict validation mode
- `--check-params` - Validate parameter usage

**Examples:**
```bash
# Validate specific template
snapsql validate queries/users.snap.sql

# Validate all templates
snapsql validate --all

# Strict validation
snapsql validate --all --strict
```

### config - Configuration Management

Manage project configuration.

```bash
snapsql config <subcommand>
```

**Subcommands:**
- `show` - Display current configuration
- `validate` - Validate configuration file
- `test-db` - Test database connection

**Examples:**
```bash
# Show current configuration
snapsql config show

# Validate configuration
snapsql config validate

# Test database connection
snapsql config test-db
```

### version - Show Version

Display version information.

```bash
snapsql version
```

## Parameter Formats

### Command Line Parameters

```bash
# Simple parameters
--param name=john --param active=true --param limit=50

# Nested parameters (use dot notation)
--param filters.active=true --param pagination.limit=20
```

### JSON Parameter File

```json
{
  "name": "john",
  "active": true,
  "filters": {
    "active": true,
    "department": "engineering"
  },
  "pagination": {
    "limit": 20,
    "offset": 0
  }
}
```

### YAML Parameter File

```yaml
name: john
active: true
filters:
  active: true
  department: engineering
pagination:
  limit: 20
  offset: 0
```

## Output Formats

### Table Format (Default)

```
+----+----------+-------------------+
| id | name     | email             |
+----+----------+-------------------+
|  1 | John Doe | john@example.com  |
|  2 | Jane Doe | jane@example.com  |
+----+----------+-------------------+
```

### JSON Format

```json
{
  "rows": [
    {"id": 1, "name": "John Doe", "email": "john@example.com"},
    {"id": 2, "name": "Jane Doe", "email": "jane@example.com"}
  ],
  "count": 2,
  "execution_time": "15ms"
}
```

### CSV Format

```csv
id,name,email
1,John Doe,john@example.com
2,Jane Doe,jane@example.com
```

## Environment Variables

- `SNAPSQL_CONFIG` - Default configuration file path
- `DATABASE_URL` - Database connection string
- `SNAPSQL_VERBOSE` - Enable verbose output (true/false)
- `SNAPSQL_QUIET` - Enable quiet mode (true/false)

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Template validation error
- `4` - Database connection error
- `5` - Query execution error

## Examples

### Complete Workflow

```bash
# 1. Initialize project
snapsql init my-project
cd my-project

# 2. Generate intermediate files
snapsql generate

# 3. Test query with dry-run
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# 4. Execute query
snapsql query queries/users.snap.sql --params-file params.json --format json

# 5. Validate all templates
snapsql validate --all
```

### Development Workflow

```bash
# Watch for changes and regenerate
snapsql generate --watch &

# Test queries as you develop
snapsql query queries/new-query.snap.sql --dry-run --param test=true

# Validate before commit
snapsql validate --all --strict
```

### Production Usage

```bash
# Set production database
export DATABASE_URL="postgres://prod-server:5432/mydb"

# Execute with production parameters
snapsql query queries/report.snap.sql \
  --params-file prod-params.json \
  --format csv \
  --output daily-report.csv \
  --timeout 5m
```
