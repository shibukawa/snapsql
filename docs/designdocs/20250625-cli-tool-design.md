# SnapSQL CLI Tool Design

## Overview

The SnapSQL command-line tool (`snapsql`) provides a comprehensive interface for managing SQL templates, generating code, and extracting database schemas. It supports multiple programming languages and integrates with the SnapSQL template engine.

## Architecture

### Command Structure

```
snapsql <command> [options] [arguments]
```

### Core Commands

1. **generate** - Generate intermediate files or runtime code
2. **validate** - Validate SQL templates
3. **pull** - Extract database schema information
4. **init** - Initialize a new SnapSQL project

### Global Options

- `--config <file>` - Configuration file path (default: `./snapsql.yaml`)
- `--verbose, -v` - Enable verbose output
- `--quiet, -q` - Enable quiet mode
- `--help, -h` - Show help information
- `--version` - Show version information
- `--no-color` - Disable colored output

## Commands Specification

### 1. generate Command

Generates intermediate files or language-specific code from SQL templates.

#### Syntax
```bash
snapsql generate [options]
```

#### Options
- `-i, --input <path>` - Input file or directory
- `--lang <language>` - Output language/format (default: `json`)
  - Supports any language name (extensible for custom generators)
  - When not specified, generates all configured languages
- `--package <name>` - Package name (language-specific)
- `--const <files>` - Constant definition files (can be specified multiple times)
- `--validate` - Validate templates before generation
- `--watch` - Watch for file changes and regenerate automatically

#### Built-in Generators

| Language | Internal | Output |
|----------|----------|---------|
| `json` | Yes | Intermediate JSON files |
| `go` | Yes | Go language code |
| `typescript` | Yes | TypeScript code |
| `java` | Yes | Java language code |
| `python` | Yes | Python language code |

#### External Generator Plugin Support

For languages not built-in, the tool looks for external executables:
- Pattern: `snapsql-gen-<language>`
- Location: System PATH
- Interface: Command-line with JSON input

#### Examples
```bash
# Generate all configured languages (JSON + any configured languages)
snapsql generate

# Generate specific language only
snapsql generate --lang go --package queries

# Generate from single file
snapsql generate -i queries/users.snap.sql

# Generate with constant files
snapsql generate --const constants.yaml --const tables.yaml

# Generate with validation
snapsql generate --lang typescript --validate

# Watch mode for development
snapsql generate --watch
```

### 2. validate Command

Validates SQL templates for syntax and structure.

#### Syntax
```bash
snapsql validate [options] [files...]
```

#### Options
- `-i, --input <dir>` - Input directory (default: `./queries`)
- `--files <files...>` - Specific files to validate
- `--strict` - Enable strict validation mode
- `--format <format>` - Output format (`text`, `json`) (default: `text`)

#### Validation Rules
- SnapSQL syntax validation
- Parameter type checking
- Template structure validation
- Cross-reference validation with constants

#### Examples
```bash
# Validate all templates
snapsql validate

# Validate specific files
snapsql validate queries/users.snap.sql queries/posts.snap.md

# Strict mode with JSON output
snapsql validate --strict --format json
```

### 3. pull Command

Extracts database schema information and saves it to a YAML file.

#### Syntax
```bash
snapsql pull [options]
```

#### Options
- `--db <connection>` - Database connection string
- `--env <environment>` - Environment name from configuration
- `-o, --output <file>` - Output file (default: `./schema.yaml`)
- `--tables <pattern>` - Table patterns to include (wildcard supported)
- `--exclude <pattern>` - Table patterns to exclude
- `--include-views` - Include database views
- `--include-indexes` - Include index information

#### Supported Databases
- PostgreSQL
- MySQL
- SQLite

#### Examples
```bash
# Extract from environment configuration
snapsql pull --env development

# Extract with direct connection
snapsql pull --db "postgres://user:pass@localhost/mydb"

# Extract specific tables with views
snapsql pull --env production --tables "user*,post*" --include-views

# Exclude system tables
snapsql pull --env development --exclude "pg_*,information_schema*"
```

### 4. init Command

Initializes a new SnapSQL project with directory structure and sample files.

#### Syntax
```bash
snapsql init
```

#### Generated Structure
```
./
├── snapsql.yaml           # Configuration file
├── queries/               # SQL templates directory
│   └── users.snap.sql     # Sample SQL template
├── constants/             # Constants directory
│   └── database.yaml      # Sample constants
└── generated/             # Output directory (created on first generate)
```

#### Generated Files
- **snapsql.yaml**: Complete configuration with examples
- **queries/users.snap.sql**: Sample SnapSQL template
- **constants/database.yaml**: Sample constants file

#### Examples
```bash
# Initialize in current directory
snapsql init
```

## Configuration File

### File Location
- Default: `./snapsql.yaml`
- Override: `--config <file>`

### Configuration Structure

```yaml
# SQL dialect

dialect: "postgres"  # postgres, mysql, sqlite

# Database connections are typically supplied via `.tbls.yaml` or the `--db` flag for commands that need a database connection.

# Constant definition files
constant_files:
  - "./constants/database.yaml"
  - "./constants/tables.yaml"

# Generation settings
generation:
  input_dir: "./queries"
  validate: true
  
  # Generator configurations (flexible structure for any language)
  generators:
    json:
      output: "./generated"
      enabled: true
      settings:
        pretty: true
        include_metadata: true
    
    go:
      output: "./internal/queries"
      enabled: false
      settings:
        package: "queries"
        generate_tests: true
    
    typescript:
      output: "./src/generated"
      enabled: false
      settings:
        types: true
        module_type: "esm"
    
    java:
      output: "./src/main/java"
      enabled: false
      settings:
        package: "com.example.queries"
        use_lombok: true
    
    python:
      output: "./src/queries"
      enabled: false
      settings:
        package: "queries"
        use_dataclasses: true
    
    # Custom generators can be added
    rust:
      output: "./src/generated"
      enabled: false
      settings:
        crate_name: "queries"
        async_runtime: "tokio"

# Validation settings
validation:
  strict: false
  rules:
    - "no-dynamic-table-names"
    - "require-parameter-types"
```

### Environment Variable Support

SnapSQL supports environment variable expansion in configuration files:

- **Syntax**: `${VAR_NAME}` or `$VAR_NAME`
- **`.env` File**: Automatically loaded from current directory if present
- **Use Cases**: Database credentials, paths, environment-specific settings

**Example `.env` file:**
```bash
DB_USER=myuser
DB_PASS=mypassword
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
```

**Configuration with environment variables:**
```yaml
# Example: specify a connection string using an environment variable
production:
  connection: "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
```

## File Processing

### Supported File Types

1. **`.snap.sql`** - Pure SQL templates
2. **`.snap.md`** - Markdown-based literate programming files

### Processing Pipeline

1. **Discovery**: Find all `.snap.sql` and `.snap.md` files
2. **Parsing**: Parse SnapSQL syntax and extract metadata
3. **Validation**: Validate syntax and structure
4. **Constant Resolution**: Resolve `/*@ */` constants
5. **Generation**: Generate target language code
6. **Output**: Write generated files

### Template Processing

#### SnapSQL Syntax Support
- `/*# if condition */` - Conditional blocks
- `/*# for variable : list */` - Loop blocks
- `/*# end */` - End blocks
- `/*= variable */` - Variable substitution
- `/*@ constant */` - Constant expansion

#### Constant File Processing
- YAML format constant definitions
- Hierarchical constant access
- Multiple constant file support

## Error Handling

### Exit Codes
- `0` - Success
- `1` - General error
- `2` - Invalid arguments
- `3` - File I/O error
- `4` - Validation error
- `5` - Database connection error

### Error Messages
- Colored output (unless `--no-color`)
- Detailed error descriptions
- File and line number information
- Suggestions for common issues

### Verbose Mode
- Progress information
- File processing details
- Generation statistics
- Performance metrics

## Plugin System

### External Generator Discovery
- Executable name pattern: `snapsql-gen-<language>`
- Search location: System PATH
- Automatic discovery and execution

### Plugin Interface
```bash
snapsql-gen-<lang> [options] <intermediate-json-file>
```

#### Standard Options
- `--output-dir <path>` - Output directory
- `--package <name>` - Package/namespace name
- `--schema-file <path>` - Database schema file
- `--constants <files>` - Constant definition files
- `--config <file>` - SnapSQL configuration file
- `--verbose` - Enable verbose output

### Plugin Communication
- **Input**: Intermediate JSON via file
- **Output**: Generated code files
- **Logging**: stdout/stderr for messages
- **Exit Codes**: Standard exit code conventions

## Development Workflow

### Typical Usage Patterns

#### 1. New Project Setup
```bash
# Initialize project
snapsql init

# Edit configuration
vim snapsql.yaml

# Create SQL templates
vim queries/my-query.snap.sql

# Generate code
snapsql generate --lang go
```

#### 2. Existing Database Integration
```bash
# Extract schema
snapsql pull --env production

# Create templates based on schema
# Edit queries/*.snap.sql

# Validate templates
snapsql validate

# Generate code
snapsql generate --lang typescript
```

#### 3. Development Mode
```bash
# Watch mode for continuous generation
snapsql generate --watch --lang go

# In another terminal, edit templates
vim queries/users.snap.sql
# Files are automatically regenerated
```

#### 4. CI/CD Integration
```bash
# Validation in CI
snapsql validate --strict --format json

# Code generation for deployment
snapsql generate --lang java --validate
```

## Performance Considerations

### File Processing
- Parallel processing of independent templates
- Incremental generation (only changed files)
- Efficient memory usage for large template sets

### Watch Mode
- File system event-based watching
- Debounced regeneration
- Selective processing of changed files

### Database Operations
- Connection pooling for database operations
- Efficient metadata queries
- Timeout handling for slow connections

## Security Considerations

### Database Connections
- Secure credential handling
- Connection string validation
- Timeout and retry mechanisms

### File Operations
- Path traversal protection
- Safe file writing with atomic operations
- Permission validation

### Code Generation
- SQL injection prevention
- Safe template processing
- Output sanitization

## Testing Strategy

### Unit Tests
- Command parsing and validation
- Configuration file processing
- Template parsing and generation
- Error handling scenarios

### Integration Tests
- End-to-end command execution
- Multi-language code generation
- Plugin system integration

### Performance Tests
- Large template set processing
- Memory usage optimization
- Generation speed benchmarks
