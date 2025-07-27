# SnapSQL

SnapSQL is a SQL template engine that enables dynamic SQL generation using the 2-way SQL format. It allows developers to write SQL templates that can be executed as standard SQL during development while providing runtime flexibility for dynamic query construction.

## Features

- **2-way SQL Format**: Write SQL templates that work as standard SQL when comments are removed
- **Dynamic Query Building**: Add WHERE clauses, ORDER BY, and SELECT fields dynamically at runtime
- **Security First**: Controlled modifications prevent SQL injection - only allows safe operations like field selection, simple conditions, and table suffix changes
- **Multi-Database Support**: Designed to work with PostgreSQL, MySQL, and SQLite databases
- **Google CEL Integration**: Uses Common Expression Language for conditions and parameter references
- **Advanced SQL Parsing**: Comprehensive SQL parser with support for complex queries, CTEs, and DML operations
- **Template Engine**: Powerful template processing with conditional blocks, loops, and variable substitution
- **Bulk Operations**: Support for bulk INSERT operations with dynamic field mapping
- **Type Safety**: Strong type checking and parameter validation

## How It Works

SnapSQL works with your existing database infrastructure and table schemas:

1. **Build Time**: The `snapsql` tool (Go) converts SQL templates and markdown files containing SQL into intermediate JSON files containing AST and metadata *(Planned)*
2. **Runtime**: Language-specific libraries use the intermediate files to dynamically construct SQL queries based on runtime parameters *(Planned)*
3. **Current State**: The core SQL parser and template engine are implemented in Go

**Prerequisites**: SnapSQL assumes you have an existing database with tables already created. It works with PostgreSQL, MySQL, and SQLite databases.

## Runtime Features

### Type-Safe Query Functions

The runtime libraries generate type-safe functions based on the parsed SQL structure, providing:

- **Strongly Typed Parameters**: Function signatures that match your SQL template parameters
- **Result Type Mapping**: Automatic mapping of SQL results to language-specific types
- **Compile-Time Safety**: Catch parameter mismatches and type errors at compile time
- **IDE Support**: Full autocomplete and IntelliSense support for query parameters and results

### Mock and Testing Support

SnapSQL provides built-in testing capabilities without requiring database connections:

- **Mock Data Generation**: Return dummy data that matches your query structure for unit testing
- **YAML-Based Mock Data**: Define mock responses using YAML format with [dbtestify](https://github.com/shibukawa/dbtestify) library integration
- **Configurable Responses**: Define custom mock responses for different test scenarios
- **Zero Database Dependencies**: Run tests without setting up test databases
- **Consistent Data Shapes**: Mock data follows the same type structure as real query results

### Performance Analysis

Built-in performance analysis tools help optimize your queries:

- **Execution Plan Analysis**: Generate and analyze query execution plans
- **Performance Estimation**: Predict query performance based on table statistics and query complexity
- **Bottleneck Detection**: Identify potential performance issues before deployment
- **Optimization Suggestions**: Receive recommendations for query improvements

These features work seamlessly without requiring changes to your application code - simply toggle between production, mock, and analysis modes through configuration.

## Template Syntax

SnapSQL uses comment-based directives that don't interfere with standard SQL execution, following the **2-way SQL format**:

### Control Flow Directives
- `/*# if condition */` - Conditional blocks
- `/*# elseif condition */` - Alternative conditions (Note: uses `elseif`, not `else if`)
- `/*# else */` - Default condition
- `/*# end */` - End conditional blocks (unified ending for all control structures)
- `/*# for variable : list */` - Loop over collections
- `/*# end */` - End loop blocks

### Variable Substitution
- `/*= variable */` - Variable placeholders with dummy literals for 2-way SQL compatibility

### 2-Way SQL Format

SnapSQL templates are designed to work as **valid SQL** when comments are removed, enabling:
- **IDE Support**: Full syntax highlighting and IntelliSense
- **SQL Linting**: Standard SQL tools can validate basic syntax
- **Development Testing**: Execute templates with dummy values during development

#### Variable Substitution with Dummy Literals

Variables include dummy literals that make the SQL valid when comments are stripped:

```sql
-- Template with dummy literals
SELECT * FROM users_/*= table_suffix */test
WHERE active = /*= filters.active */true
  AND department IN (/*= filters.departments */'sales', 'marketing')
LIMIT /*= pagination.limit */10;

-- Valid SQL when comments removed
SELECT * FROM users_test
WHERE active = true
  AND department IN ('sales', 'marketing')
LIMIT 10;

-- Runtime result with actual parameters
SELECT * FROM users_prod
WHERE active = false
  AND department IN ('engineering', 'design', 'product')
LIMIT 20;
```

#### Automatic Runtime Adjustments

The runtime automatically handles:
- **Trailing Commas**: Removed when conditional fields are excluded
- **Empty Clauses**: WHERE/ORDER BY clauses removed when all conditions are false
- **Array Expansion**: `/*= array_var */` expands to `'val1', 'val2', 'val3'`
- **Dummy Literal Removal**: Development dummy values replaced with actual parameters
- **Conditional Clause Removal**: Entire clauses removed when variables are null/empty
  - `WHERE` clause: Removed when all conditions are null/empty
  - `ORDER BY` clause: Removed when sort fields are null/empty
  - `LIMIT` clause: Removed when limit is null or negative
  - `OFFSET` clause: Removed when offset is null or negative
  - `AND/OR` conditions: Individual conditions removed when variables are null/empty

### Template Formatting Guidelines

1. **Indentation in Control Blocks**: Content inside `/*# if */` and `/*# for */` blocks should be indented one level
2. **Line Breaks for Readability**: Start `/*# for */` blocks on new lines for better visibility
3. **Consistent Structure**: Maintain consistent indentation to improve template readability

```sql
-- Good formatting with proper indentation
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# for field : additional_fields */
        /*= field */
    /*# end */
FROM users
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */
    /*# end */name ASC;

-- Avoid: Poor formatting without indentation
SELECT id, name, /*# if include_email */email,/*# end */ /*# for field : additional_fields *//*= field *//*# end */ FROM users;
```

### Template Writing Guidelines

1. **Use trailing delimiters**: Write `field,` not `,field` for better readability and JSON-like syntax
2. **Include dummy literals**: Ensure SQL remains valid when comments are removed
3. **Use consistent endings**: Always use `/*# end */` for all control structures
4. **Leverage automatic adjustments**: Don't worry about trailing commas or empty clauses
5. **Omit obvious conditions**: Skip `/*# if */` blocks for automatic clause removal (WHERE, ORDER BY, LIMIT, OFFSET)

### Template Formatting Guidelines

1. **Trailing Delimiters**: Use trailing commas, AND, OR for better readability (JSON-like syntax)
2. **Indentation in Control Blocks**: Content inside `/*# if */` and `/*# for */` blocks should be indented one level
3. **Line Breaks for Readability**: Start `/*# for */` blocks on new lines for better visibility
4. **Consistent Structure**: Maintain consistent indentation to improve template readability

```sql
-- Good formatting with trailing delimiters (JSON-like, familiar to modern developers)
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# for field : additional_fields */
    /*= field */,
    /*# end */
    created_at
FROM users
WHERE active = true
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'sales', 'marketing')
    /*# end */
ORDER BY 
    /*# for sort : sort_fields */
    /*= sort.field */ /*= sort.direction */,
    /*# end */
    name ASC;

-- Avoid: Leading delimiters (SQL traditional but less familiar to JSON/programming users)
SELECT 
    id
    , name
    /*# if include_email */
    , email
    /*# end */
FROM users;
```

### Automatic Clause Removal Examples

```sql
-- You can write simply:
WHERE active = /*= filters.active */true
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'sales', 'marketing')
    /*# end */
ORDER BY 
    /*# for sort : sort_fields */
    /*= sort.field */ /*= sort.direction */,
    /*# end */
    name ASC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */5

-- Instead of verbose conditional blocks:
/*# if filters.active */
WHERE active = /*= filters.active */
    /*# if filters.departments */
    AND department IN (/*= filters.departments */)
    /*# end */
/*# end */
/*# if sort_fields */
ORDER BY 
    /*# for sort : sort_fields */
    /*= sort.field */ /*= sort.direction */,
    /*# end */
/*# end */
/*# if pagination.limit > 0 */
LIMIT /*= pagination.limit */
/*# end */
```

### Example Template

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# if include_profile */
    profile_image,
    bio,
    /*# end */
    /*# for field : additional_fields */
    /*= field */,
    /*# end */
    created_at
FROM users_/*= table_suffix */test
WHERE active = /*= filters.active */true
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'sales', 'marketing')
    /*# end */
ORDER BY 
    /*# for sort : sort_fields */
    /*= sort.field */ /*= sort.direction */,
    /*# end */
    name ASC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */5;
```

### Bulk Insert Example

SnapSQL supports bulk INSERT operations with multiple VALUES clauses:

```sql
-- Basic bulk insert
INSERT INTO users (name, email, created_at) 
VALUES 
    ('John Doe', 'john@example.com', NOW()),
    ('Jane Smith', 'jane@example.com', NOW()),
    ('Bob Wilson', 'bob@example.com', NOW());

-- Dynamic bulk insert with SnapSQL variables
INSERT INTO products (name, price, category_id) 
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2),
    (/*= product3.name */'Product C', /*= product3.price */150.25, /*= product3.category_id */1);

-- Conditional bulk insert
INSERT INTO orders (user_id, product_id, quantity) 
VALUES 
    (/*= order.user_id */1, /*= order.product_id */1, /*= order.quantity */2)
    /*# if include_bulk_orders */
    , (/*= bulk_order1.user_id */2, /*= bulk_order1.product_id */2, /*= bulk_order1.quantity */1)
    , (/*= bulk_order2.user_id */3, /*= bulk_order2.product_id */3, /*= bulk_order2.quantity */5)
    /*# end */;

-- Map array bulk insert (automatic expansion)
-- When 'products' is []map[string]any, it automatically expands to multiple VALUES clauses
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);

-- Single map insert (non-bulk)
-- When 'product' is map[string]any, it's treated as regular variables
INSERT INTO products (name, price, category_id) 
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

When comments are removed, this becomes valid SQL:
```sql
SELECT 
    id,
    name,
        email,
        profile_image,
        bio
        field1
FROM users_test
WHERE active = true
    AND department IN ('sales', 'marketing')
ORDER BY 
        name ASC, created_at DESC
    name ASC
LIMIT 10
OFFSET 5;
```

Runtime result with actual parameters:
```sql
SELECT 
    id,
    name,
    email
FROM users_prod
WHERE active = false
    AND department IN ('engineering', 'design', 'product')
ORDER BY created_at DESC
LIMIT 20
OFFSET 40;
```

## Supported Dynamic Operations

### ✅ Allowed Operations
- Adding/removing WHERE conditions
- Adding/removing ORDER BY clauses  
- Adding/removing SELECT fields
- Table name suffix modification (e.g., `users_test`, `log_202412`)
- Array expansion in IN clauses (e.g., `/*= departments */` → `'sales', 'marketing', 'engineering'`)
- Trailing comma and parentheses control
- Conditional removal of entire clauses (WHERE, ORDER BY)
- **Bulk INSERT operations** with multiple VALUES clauses
- **Dynamic DML operations** (INSERT, UPDATE, DELETE with SnapSQL variables)

### ❌ Restricted Operations
- Major structural changes to SQL
- Dynamic table name changes (except suffixes)
- Arbitrary SQL injection

### Runtime Processing Features

#### Automatic Cleanup
- **Trailing Commas**: Automatically removed when conditional fields are excluded
- **Empty WHERE Clauses**: Entire WHERE clause removed when no conditions are active
- **Empty ORDER BY**: ORDER BY clause removed when no sort fields are specified
- **Dummy Literals**: Development dummy values replaced with actual runtime values

#### Array Processing
- **IN Clause Expansion**: Array variables automatically expand to comma-separated quoted values
- **Bulk Operations**: Support for bulk insert operations with dynamic field mapping

## Installation

### Build Tool *(Planned)*
```bash
go install github.com/shibukawa/snapsql@latest
```

### Runtime Libraries *(Planned)*

#### Go
```bash
go get github.com/shibukawa/snapsql
```

#### Java *(Planned)*
```xml
<dependency>
    <groupId>com.github.shibukawa</groupId>
    <artifactId>snapsql-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

#### Python *(Planned)*
```bash
pip install snapsql-python
```

#### Node.js *(Planned)*
```bash
npm install snapsql-js
```

### Current Usage (Development)

Currently, you can use SnapSQL as a Go library for parsing and processing SQL templates:

```bash
go get github.com/shibukawa/snapsql
```

## Configuration

SnapSQL uses a `snapsql.yaml` configuration file to manage database connections, generation settings, and other options.

### Environment Variable Support

SnapSQL supports environment variable expansion in configuration files:

```yaml
# Database connections with environment variables
databases:
  development:
    driver: "postgres"
    connection: "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
    schema: "public"
  
  production:
    driver: "postgres"
    connection: "postgres://${PROD_DB_USER}:${PROD_DB_PASS}@${PROD_DB_HOST}:${PROD_DB_PORT}/${PROD_DB_NAME}"
    schema: "public"

# Generation settings
generation:
  input_dir: "./queries"
  validate: true
  
  # Configure generators for multiple languages
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
    
    typescript:
      output: "./src/generated"
      enabled: false
      settings:
        types: true
```

### Generator Configuration

Each generator in the `generators` section supports:

- **`output`**: Output directory for generated files
- **`enabled`**: Whether the generator is enabled (default: false, except JSON which is always enabled)
- **`settings`**: Generator-specific settings (flexible key-value pairs)

#### Built-in Generators

| Generator | Settings | Description |
|-----------|----------|-------------|
| `json` | `pretty`, `include_metadata` | Always enabled, generates intermediate JSON files |
| `go` | `package`, `generate_tests` | Go language code generation |
| `typescript` | `types`, `module_type` | TypeScript code generation |
| `java` | `package`, `use_lombok` | Java language code generation |
| `python` | `package`, `use_dataclasses` | Python code generation |

#### Custom Generators

You can add any custom generator by specifying its name and configuration:

```yaml
generation:
  generators:
    rust:
      output: "./src/generated"
      enabled: true
      settings:
        crate_name: "queries"
        async_runtime: "tokio"
    
    csharp:
      output: "./Generated"
      enabled: true
      settings:
        namespace: "MyApp.Queries"
        use_nullable: true
```

Custom generators are executed as external plugins named `snapsql-gen-<language>` (e.g., `snapsql-gen-rust`, `snapsql-gen-csharp`).

### .env File Support

Create a `.env` file in your project root:

```bash
# Database credentials
DB_USER=myuser
DB_PASS=mypassword
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb

# Production database
PROD_DB_USER=produser
PROD_DB_PASS=prodpassword
PROD_DB_HOST=prod.example.com
PROD_DB_PORT=5432
PROD_DB_NAME=proddb
```

## Usage

### 1. Create SQL Templates

Create `.snap.sql` or `.snap.md` files with SnapSQL syntax:

#### Simple SQL Template (`.snap.sql`)
```sql
-- queries/users.snap.sql
SELECT 
    id,
    name
    /*# if include_email */,
    email
    /*# end */
FROM users_/*= env */test
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */
/*# if sort_by != "" */
ORDER BY /*= sort_by */name
/*# end */
```

#### Literate Programming with Markdown (`.snap.md`)

SnapSQL supports literate programming through `.snap.md` files that combine SQL templates with documentation, design notes, and comprehensive testing:

````markdown
# User Query Template

## Design Overview

This template handles user data retrieval with dynamic field selection and filtering capabilities.

### Requirements
- Support conditional email field inclusion based on user permissions
- Environment-specific table selection (dev/staging/prod)
- Optional filtering by user status
- Configurable sorting

## SQL Template

```sql
SELECT 
    id, 
    name
    /*# if include_email */,
    email
    /*# endif */
FROM users_/*= env */
/*# if filters.active */
WHERE active = true
/*# endif */
/*# if sort_by != "" */
ORDER BY /*= sort_by */
/*# endif */
```

## Test Cases

### Test Case 1: Basic Query
**Input Parameters:**
```json
{
    "include_email": false,
    "env": "prod",
    "filters": {"active": false},
    "sort_by": ""
}
```

**Expected Output:**
```sql
SELECT id, name FROM users_prod
```

### Test Case 2: Full Query with All Options
**Input Parameters:**
```json
{
    "include_email": true,
    "env": "dev",
    "filters": {"active": true},
    "sort_by": "created_at"
}
```

**Expected Output:**
```sql
SELECT id, name, email FROM users_dev WHERE active = true ORDER BY created_at
```

## Mock Data Examples

```yaml
# Mock response for user query
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```
````

### 2. Generate Code

```bash
# Generate all configured languages
snapsql generate

# Generate specific language
snapsql generate --lang go

# Generate from single file
snapsql generate -i queries/users.snap.sql

# Generate with constant files
snapsql generate --const constants.yaml
```

### 3. Use Runtime Libraries *(Planned)*

#### Go Example *(Planned)*
```go
package main

import (
    "github.com/shibukawa/snapsql"
)

func main() {
    engine := snapsql.New("generated/")
    
    params := map[string]any{
        "include_email": true,
        "env": "prod",
        "filters": map[string]any{
            "active": true,
        },
        "sort_by": "created_at",
    }
    
    query, args, err := engine.Build("users", params)
    if err != nil {
        panic(err)
    }
    
    // Execute with your database driver
    rows, err := db.Query(query, args...)
}
```

#### Python Example *(Planned)*
```python
import snapsql

engine = snapsql.Engine("generated/")

params = {
    "include_email": True,
    "env": "prod", 
    "filters": {
        "active": True
    },
    "sort_by": "created_at"
}

query, args = engine.build("users", params)

# Execute with your database driver
cursor.execute(query, args)
```

### Current Usage (Development)

Currently, you can use SnapSQL CLI tool for generating intermediate files and parsing SQL templates:

```bash
# Initialize a new project
snapsql init

# Generate intermediate JSON files
snapsql generate

# Generate specific language (when implemented)
snapsql generate --lang go

# Validate templates
snapsql validate
```

You can also use SnapSQL as a Go library for parsing SQL templates:

```go
package main

import (
    "fmt"
    "github.com/shibukawa/snapsql/parser"
    "github.com/shibukawa/snapsql/tokenizer"
)

func main() {
    sql := `SELECT id, name FROM users WHERE active = /*= active */true`
    
    tokens, err := tokenizer.Tokenize(sql)
    if err != nil {
        panic(err)
    }
    
    ast, err := parser.Parse(tokens)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Parsed AST: %+v\n", ast)
}
```

## Intermediate Format Schema

SnapSQL generates intermediate JSON files that follow a standardized format. The JSON schema for these files is available at:

- **Schema File**: [`docs/intermediate-format-schema.json`](docs/intermediate-format-schema.json)
- **Schema ID**: `https://github.com/shibukawa/snapsql/schemas/intermediate-format.json`

### Schema Validation

You can validate generated intermediate files using JSON Schema validators:

```bash
# Using ajv-cli
npm install -g ajv-cli
ajv validate -s docs/intermediate-format-schema.json -d generated/your-file.json

# Using other validators
# Most JSON Schema validators support Draft-07 format
```

### Intermediate File Structure

The intermediate format includes:

- **`source`**: Original template file information (path and content)
- **`interface_schema`**: Extracted parameter definitions and metadata (optional)
- **`ast`**: Abstract Syntax Tree of the parsed SQL

**Example intermediate file:**
```json
{
  "source": {
    "file": "/path/to/users.snap.sql",
    "content": "SELECT * FROM users WHERE id = /*= user_id */1;"
  },
  "interface_schema": {
    "name": "user_query",
    "function_name": "getUser",
    "parameters": [
      {
        "name": "user_id",
        "type": "int"
      }
    ]
  },
  "ast": {
    "type": "SELECT_STATEMENT",
    "pos": [1, 1, 0],
    "Children": {
      "select_clause": { ... },
      "from_clause": { ... },
      "where_clause": { ... }
    }
  }
}
```

## Development Status

🚧 **Under Development** - This project is currently in the design and early development phase.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

- **Build Tool (`snapsql`)**: Licensed under AGPL-3.0
- **Runtime Libraries**: Licensed under Apache-2.0

This dual licensing approach ensures the build tool remains open source while allowing flexible use of runtime libraries in various projects.

## Repository

[https://github.com/shibukawa/snapsql](https://github.com/shibukawa/snapsql)
