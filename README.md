# SnapSQL

SnapSQL is a SQL template engine that enables dynamic SQL generation using the **2-way SQL format**. Write SQL templates that work as standard SQL during development while providing runtime flexibility for dynamic query construction.

## Philosophy

### SQL is a Public Interface
SQL queries are not just implementation details—they are the public interface between your application and data. SnapSQL treats SQL as a first-class citizen, making queries visible, testable, and maintainable rather than hiding them behind abstraction layers. SQL itself should be properly tested and guaranteed to be of high quality.

In onion architecture and clean architecture, data access and UI are represented in the same layer. While developers happily expose UI interfaces through OpenAPI, GraphQL, or gRPC specifications, they often insist on hiding database access behind repository patterns. SnapSQL challenges this inconsistency by treating your SQL queries as the explicit, documented interface to your data layer. SnapSQL doesn't hide transactions, unleashing the full power of relational databases.

### Static Typing is a Communication Tool
Types are not just for the compiler—they communicate intent and structure to developers, AI agents, and tooling ecosystems. SnapSQL generates type-safe interfaces from your SQL templates using advanced type inference from SQL schemas and database metadata, making the contract between queries and application code explicit and self-documenting.

Through if/for directive comments, a single SQL template can be flexibly adapted for different use cases while maintaining type safety. Database common system columns (like created_at, updated_at, version) are handled naturally within the type system, providing seamless integration with your application's data models.

### Runtime Should be Thin
Heavy ORMs and query builders add unnecessary complexity and performance overhead. SnapSQL's runtime libraries are lightweight adapters that execute pre-processed templates efficiently, keeping the runtime footprint minimal while maximizing developer productivity.

Smaller dependencies result in smaller program sizes and reduced security attack surfaces. The Go runtime library depends only on the standard library plus `github.com/google/cel-go` and `github.com/shopspring/decimal`, ensuring minimal external dependencies.

### Mocking is Opening Pandora's Box
Wrong mocking makes code complex, creates tight coupling, and makes refactoring difficult. It increases code written for mocks rather than for quality. Incorrect mock data accelerates the divergence between reality and implementation, hiding bugs until integration testing.

SnapSQL enables mocking at the most decoupled point without changing production code, providing authentic mock data that never diverges from reality. This approach is inspired by Mock Service Worker and Prism Contract Testing.

## How It Works

```mermaid
flowchart LR
    DB[(Database<br/>Tables Defined)]
    SQL[SQL Files<br/>*.snap.sql]
    MD[Markdown Files<br/>*.snap.md]
    
    SNAPSQL[SnapSQL<br/>CLI]
    
    GOCODE[Go Code<br/>Type-safe Functions]
    MOCK[Mock Data<br/>JSON]
    
    APP[Application<br/>Program]
    TEST[Test Code]
    
    DB -->|Schema Info| SNAPSQL
    SQL -->|Templates| SNAPSQL
    MD -->|Template, Mock Data| SNAPSQL
    
    SNAPSQL -->|Generate| GOCODE
    SNAPSQL -->|Generate (only from markdown)| MOCK
    
    GOCODE -->|Import| APP
    GOCODE -->|Import| TEST
    MOCK -->|Load| TEST
    
    MD -->|Unit Test| MD
    
    classDef database fill:#e1f5fe
    classDef source fill:#f3e5f5
    classDef generator fill:#e8f5e8
    classDef output fill:#fff3e0
    classDef usage fill:#fce4ec
    
    class DB database
    class SQL,MD source
    class SNAPSQL generator
    class GOCODE,MOCK output
    class APP,TEST usage
```

## Key Features

- **2-way SQL Format**: Templates work as valid SQL when comments are removed
- **Dynamic Query Building**: Add WHERE clauses, ORDER BY, and SELECT fields at runtime
- **Security First**: Controlled modifications prevent SQL injection
- **Multi-Database Support**: PostgreSQL, MySQL, and SQLite
- **Template Engine**: Conditional blocks, loops, and variable substitution
- **CLI Tool**: Generate, validate, and execute SQL templates

## Quick Start

### Installation

```bash
go install github.com/shibukawa/snapsql@latest
```

### Create a Project

```bash
# Initialize a new SnapSQL project
snapsql init my-project
cd my-project

# Generate intermediate files
snapsql generate

# Test a query with dry-run
snapsql query queries/users.snap.sql --dry-run --params-file params.json
```

### Example Template

There are two types format. Primary one is Markdown. Another is SQL. It includes SQL and parameter list, directives (in comment)

````markdown
# Get Project User List

## Description

Get project user list with department information.

## Parameters

```yaml
project_id: int
include_profile: bool
page_size: int
page: int
```

## SQL

```sql
SELECT 
    u.id, 
    u.name,
    /*# if include_profile */
        p.bio,
        p.avatar_url,
    /*# end */
    d.id as departments__id,
    d.name as departments__name,
FROM users AS u
    JOIN departments AS d ON u.department_id = d.id
    /*# if include_profile */
        LEFT JOIN profiles AS p ON u.id = p.user_id
    /*# end */
WHERE
  u.project_id = /*= project_id */ AND
  u.active = /*= active */
LIMIT /*= page_size != 0 ? page_size : 10 */
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */
```
````

This template works as valid SQL when comments are removed, while providing runtime flexibility. Fields with double underscores (`departments__id`, `departments__name`) are automatically structured into nested objects. And it supports CEL expression.

### Generated Code Usage

```go
// Generated type-safe function
func GetProjectUserList(ctx context.Context, db *sql.DB, projectId bool, includeProfile bool, pageSize int, page int) iter.Seq2[User, error] {
    // SnapSQL generates this function from the template above
}

// Usage in your application
func main() {
    db, _ := sql.Open("postgres", "connection_string")
    
    for user, err := range GetProjectUserList(ctx, db, 
        10,    // includeEmail
        false, // includeProfile
        10,    // pageSize
        20,    // page
    ) {
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("User: %s (%s)\n", user.Name, user.Departments[0].Name)
    }
}
```

This is a basic example, but SnapSQL provides many additional features:

* **Smart Return Types**: Analyzes queries and generates appropriate return types - `iter.Seq2[Res, error]` for multiple results, `Res, error` for single results, and `sql.Result, error` for queries without return values.
* **Context-based System Data**: Automatically expands system data embedded in context into parameters (similar to Java's logging API MDC mechanism)
* **Loop Constructs**: Support for `for` loops in templates for dynamic query generation
* **Constant Data Expansion**: Template expansion with predefined constant data
* **Complex Conditional Expressions**: Advanced conditional logic using CEL (Common Expression Language)

### Markdown Testing Example

The above markdown can have test cases. SnapSQL runs unit tests without host programming language (Like Java or Python).

````markdown
## Test Cases

### Test: Basic user list

**Fixtures (Pre-test Data):**
```yaml
# users table
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1, active: true, created_at: "2024-01-15T10:30:00Z"}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2, active: true, created_at: "2024-01-14T09:15:00Z"}
  - {id: 3, name: "Bob Wilson", email: "bob@example.com", department_id: 1, active: false, created_at: "2024-01-13T08:20:00Z"}

# departments table  
departments:
  - {id: 1, name: "Engineering", description: "Software development team"}
  - {id: 2, name: "Design", description: "UI/UX design team"}
```

**Parameters:**
```yaml
project_id: 15
include_profile: true
page_size: 3
page: 1
```

**Expected Results:**
```yaml
- {id: 1, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z", departments__id: 1, departments__name: "Engineering"}
- {id: 2, name: "Jane Smith", email: "jane@example.com", created_at: "2024-01-14T09:15:00Z", departments__id: 2, departments__name: "Design"}
```

### Test: Empty result

**Fixtures (Pre-test Data):**
```yaml
# users table (only inactive users)
users:
  - {id: 3, name: "Bob Wilson", email: "bob@example.com", department_id: 1, active: false, created_at: "2024-01-13T08:20:00Z"}

# departments table  
departments:
  - {id: 1, name: "Engineering", description: "Software development team"}
```

**Parameters:**
```yaml
active: false
limit: 10
```

**Expected Results:**
```yaml
[]
```
````

This Markdown file serves as both documentation and executable test, providing mock data for unit testing without database dependencies.

**Additional Testing Features:**

* **dbtestify Integration**: Test framework compliant with [dbtestify](https://github.com/shibukawa/dbtestify) library for comprehensive database testing
* **Pre-test Data Setup**: Ability to configure initial table data before test execution, not just expected results
* **Multiple Data Formats**: Support for test data in YAML, JSON, CSV, and dbunit XML formats
* **Comprehensive Test Scenarios**: Define complex test cases with setup, execution, and verification phases
* **Database-agnostic Testing**: Run the same tests across different database engines

### Mock Testing Example

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/shibukawa/snapsql/langs/snapsqlgo"
    "github.com/alecthomas/assert/v2"
)

func TestGetUserList_WithMockData(t *testing.T) {
    // Create context with mock data from the "basic_user_list" test case
    ctx := snapsqlgo.WithConfig(context.Background(), "user_list_query", 
        snapsqlgo.WithMockData("basic_user_list"))
    
    // The same function call, but now returns mock data instead of hitting the database
    users := make([]User, 0)
    for user, err := range GetUserList(ctx, nil, // db can be nil when using mock
        true,  // includeEmail
        false, // includeProfile
        "prod", // tableSuffix
        Pagination{Limit: 10, Offset: 0},
    ) {
        assert.NoError(t, err)
        users = append(users, user)
    }
    
    // Verify mock data matches expectations
    assert.Equal(t, 2, len(users))
    assert.Equal(t, "John Doe", users[0].Name)
    assert.Equal(t, "Engineering", users[0].Departments[0].Name)
    assert.Equal(t, "Jane Smith", users[1].Name)
    assert.Equal(t, "Design", users[1].Departments[0].Name)
}

func TestUserListAPI_WithMockData(t *testing.T) {
    // Create context with mock data for HTTP service testing
    ctx := snapsqlgo.WithConfig(context.Background(), "user_list_query", 
        snapsqlgo.WithMockData("basic_user_list"))
    
    // Create HTTP request with mock context
    req := httptest.NewRequest("GET", "/api/users?include_email=true", nil)
    req = req.WithContext(ctx)
    
    // Create response recorder
    w := httptest.NewRecorder()
    
    // Call your HTTP handler (which internally calls GetUserList)
    userListHandler(w, req)
    
    // Verify HTTP response
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response struct {
        Users []User `json:"users"`
        Count int    `json:"count"`
    }
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    
    // Verify mock data was used
    assert.Equal(t, 2, response.Count)
    assert.Equal(t, "John Doe", response.Users[0].Name)
    assert.Equal(t, "Engineering", response.Users[0].Departments[0].Name)
}
```

This approach enables testing without database dependencies while ensuring mock data never diverges from reality, as it's generated from the same templates used in production.

## Documentation

- [Template Syntax](docs/template-syntax.md) - Complete guide to SnapSQL template syntax
- [Configuration](docs/configuration.md) - Project configuration and database setup
- [CLI Commands](docs/cli-commands.md) - Command-line tool reference
- [Installation Guide](docs/installation.md) - Detailed installation instructions
- [Development Guide](docs/development.md) - Contributing and development setup

## Current Status

🚧 **Under Development** - Core functionality is implemented and working. Runtime libraries for multiple languages are planned.

**Working Features:**
- ✅ SQL template parsing and validation
- ✅ CLI tool with generate, validate, and query commands
- ✅ 2-way SQL format support
- ✅ Dry-run mode for testing templates

**Planned Features:**
- 🔄 Runtime libraries (Go, Python, TypeScript, Java)
- 🔄 Type-safe query generation
- 🔄 Mock data support for testing

## License

- **CLI Tool**: AGPL-3.0
- **Runtime Libraries**: Apache-2.0 (planned)

## Repository

[https://github.com/shibukawa/snapsql](https://github.com/shibukawa/snapsql)
