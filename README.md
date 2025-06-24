# SnapSQL

SnapSQL is a SQL template engine that enables dynamic SQL generation using the 2-way SQL format. It allows developers to write SQL templates that can be executed as standard SQL during development while providing runtime flexibility for dynamic query construction.

## Features

- **2-way SQL Format**: Write SQL templates that work as standard SQL when comments are removed
- **Dynamic Query Building**: Add WHERE clauses, ORDER BY, and SELECT fields dynamically at runtime
- **Security First**: Controlled modifications prevent SQL injection - only allows safe operations like field selection, simple conditions, and table suffix changes
- **Multi-Database Support**: Works with PostgreSQL, MySQL, and SQLite databases
- **Multi-language Support**: Runtime libraries available for Go, Java, Python, and Node.js
- **Google CEL Integration**: Uses Common Expression Language for conditions and parameter references
- **Prepared Statement Generation**: Generates safe prepared statements for execution
- **Literate Programming**: Combine SQL templates with documentation, design notes, and test cases in `.snap.md` files
- **Comprehensive Testing**: Built-in unit testing with input/output examples and mock data generation

## How It Works

SnapSQL works with your existing database infrastructure and table schemas:

1. **Build Time**: The `snapsql` tool (Go) converts SQL templates and markdown files containing SQL into intermediate JSON files containing AST and metadata
2. **Runtime**: Language-specific libraries use the intermediate files to dynamically construct SQL queries based on runtime parameters

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

SnapSQL uses comment-based directives that don't interfere with standard SQL execution:

### Control Flow Directives
- `/*# if condition */` - Conditional blocks
- `/*# elseif condition */` - Alternative conditions
- `/*# else */` - Default condition
- `/*# endif */` - End conditional blocks
- `/*# for variable : list */` - Loop over collections
- `/*# end */` - End loop blocks

### Variable Substitution
- `/*= variable */` - Variable placeholders

### Example Template

```sql
SELECT 
    id,
    name
    /*# if user.permissions.includes("email") */,
    email
    /*# elseif user.permissions.includes("contact") */,
    phone
    /*# else */,
    'hidden' as contact_info
    /*# endif */
    /*# for field : additional_fields */,
    /*= field */
    /*# end */
FROM user_log_/*= table_suffix */
/*# if filters.search != "" */
WHERE name LIKE /*= filters.search */'%example'
/*# endif */
ORDER BY /*= sort_config.field */
```

When comments are removed, this becomes valid SQL:
```sql
SELECT 
    id,
    name,
    email,
    field1,
    field2
FROM user_log_202412
WHERE name LIKE '%example'
ORDER BY created_at
```

## Supported Dynamic Operations

### ✅ Allowed Operations
- Adding/removing WHERE conditions
- Adding/removing ORDER BY clauses  
- Adding/removing SELECT fields
- Table name suffix modification (e.g., `users_test`, `log_202412`)
- Trailing comma and parentheses control
- Conditional removal of entire clauses (WHERE, ORDER BY)

### ❌ Restricted Operations
- Major structural changes to SQL
- Dynamic table name changes (except suffixes)
- Arbitrary SQL injection

## Installation

### Build Tool
```bash
go install github.com/shibukawa/snapsql@latest
```

### Runtime Libraries

#### Go
```bash
go get github.com/shibukawa/snapsql
```

#### Java
```xml
<dependency>
    <groupId>com.github.shibukawa</groupId>
    <artifactId>snapsql-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

#### Python
```bash
pip install snapsql-python
```

#### Node.js
```bash
npm install snapsql-js
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
    /*# endif */
FROM users_/*= env */
/*# if filters.active */
WHERE active = true
/*# endif */
/*# if sort_by != "" */
ORDER BY /*= sort_by */
/*# endif */
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

### 2. Build Intermediate Files

```bash
snapsql build -i queries/ -o generated/
```

### 3. Use Runtime Libraries

#### Go Example
```go
package main

import (
    "github.com/shibukawa/snapsql"
)

func main() {
    engine := snapsql.New("generated/")
    
    params := map[string]interface{}{
        "include_email": true,
        "env": "prod",
        "filters": map[string]interface{}{
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

#### Python Example
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
