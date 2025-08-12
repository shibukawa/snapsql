# Testing Guide

SnapSQL provides comprehensive testing capabilities through mock data, test cases, and dry-run execution. This enables testing SQL templates without database dependencies.

## Mock Data Testing

### Basic Mock Data Structure

Mock data is defined in YAML format and can be embedded in Markdown templates or provided as external files.

```yaml
# Mock data structure
table_name:
  - {column1: value1, column2: value2, column3: value3}
  - {column1: value4, column2: value5, column3: value6}
```

### Example Mock Data

```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", active: true, created_at: "2024-01-15T10:30:00Z"}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", active: true, created_at: "2024-01-14T09:15:00Z"}
  - {id: 3, name: "Bob Wilson", email: "bob@example.com", active: false, created_at: "2024-01-13T08:20:00Z"}

departments:
  - {id: 1, name: "Engineering", description: "Software development team"}
  - {id: 2, name: "Design", description: "UI/UX design team"}
  - {id: 3, name: "Marketing", description: "Marketing and sales team"}

user_departments:
  - {user_id: 1, department_id: 1}
  - {user_id: 2, department_id: 2}
  - {user_id: 3, department_id: 1}
```

## Test Cases in Markdown Templates

### Complete Test Case Example

````markdown
---
function_name: get_active_users_by_department
---

# Get Active Users by Department

## Description
Retrieve active users filtered by department with optional email inclusion.

## Parameters
```yaml
department_id: int
include_email: bool
limit: int
```

## SQL
```sql
SELECT 
    u.id,
    u.name,
    /*# if include_email */
    u.email,
    /*# end */
    u.created_at,
    d.name as department__name
FROM users u
    JOIN user_departments ud ON u.id = ud.user_id
    JOIN departments d ON ud.department_id = d.id
WHERE u.active = true
    AND d.id = /*= department_id */1
    /*# if include_email */
    AND u.email IS NOT NULL
    /*# end */
ORDER BY u.created_at DESC
LIMIT /*= limit */10;
```

## Test Cases

### Test: Engineering department with email

**Fixtures (Pre-test Data):**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", active: true, created_at: "2024-01-15T10:30:00Z"}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", active: true, created_at: "2024-01-14T09:15:00Z"}
  - {id: 3, name: "Bob Wilson", email: "bob@example.com", active: false, created_at: "2024-01-13T08:20:00Z"}

departments:
  - {id: 1, name: "Engineering", description: "Software development team"}
  - {id: 2, name: "Design", description: "UI/UX design team"}

user_departments:
  - {user_id: 1, department_id: 1}
  - {user_id: 2, department_id: 2}
  - {user_id: 3, department_id: 1}
```

**Parameters:**
```yaml
department_id: 1
include_email: true
limit: 10
```

**Expected Results:**
```yaml
- {id: 1, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z", department__name: "Engineering"}
```

### Test: Design department without email

**Parameters:**
```yaml
department_id: 2
include_email: false
limit: 5
```

**Expected Results:**
```yaml
- {id: 2, name: "Jane Smith", created_at: "2024-01-14T09:15:00Z", department__name: "Design"}
```

### Test: No results for inactive users

**Parameters:**
```yaml
department_id: 1
include_email: false
limit: 10
```

**Expected Results:**
```yaml
[]
```
````

## External Mock Data Files

### Separate Mock Data Files

Create dedicated mock data files for reusable test data:

```yaml
# testdata/users.yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
    department_id: 1
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    active: true
    department_id: 2
    created_at: "2024-01-14T09:15:00Z"
  - id: 3
    name: "Bob Wilson"
    email: null
    active: false
    department_id: 1
    created_at: "2024-01-13T08:20:00Z"

departments:
  - id: 1
    name: "Engineering"
    budget: 1000000.00
  - id: 2
    name: "Design"
    budget: 500000.00
```

### Using External Mock Data

```bash
# Test with external mock data
snapsql query template.snap.md --dry-run --mock-data testdata/users.yaml --params-file params.json
```

## CLI Testing Commands

### Dry Run Execution

```bash
# Basic dry run with inline parameters
snapsql query template.snap.sql --dry-run --param user_id=1 --param include_email=true

# Dry run with parameter file
snapsql query template.snap.md --dry-run --params-file params.json

# Dry run with mock data file
snapsql query template.snap.sql --dry-run --mock-data testdata.yaml --params-file params.json
```

### Parameter Files

```json
// params.json
{
  "user_id": 1,
  "include_email": true,
  "department_id": 1,
  "limit": 10,
  "filters": {
    "active": true,
    "verified": true
  }
}
```

```yaml
# params.yaml
user_id: 1
include_email: true
department_id: 1
limit: 10
filters:
  active: true
  verified: true
```

### Running All Tests

```bash
# Run all test cases in a template
snapsql test template.snap.md

# Run tests for all templates in directory
snapsql test ./queries

# Run tests with verbose output
snapsql test ./queries --verbose

# Run specific test case
snapsql test template.snap.md --test-case "Engineering department with email"
```

## Advanced Testing Patterns

### Testing Complex Data Types

```yaml
# Testing with arrays and objects
users:
  - id: 1
    name: "John Doe"
    tags: ["developer", "senior", "fullstack"]
    preferences:
      theme: "dark"
      language: "en"
      notifications:
        email: true
        push: false
    addresses:
      - type: "home"
        street: "123 Main St"
        city: "New York"
      - type: "work"
        street: "456 Office Blvd"
        city: "San Francisco"
```

### Testing Date and Time

```yaml
users:
  - id: 1
    name: "John Doe"
    created_at: "2024-01-15T10:30:00Z"        # ISO 8601 format
    updated_at: "2024-01-15T10:30:00.123Z"    # With milliseconds
    birth_date: "1990-05-15"                   # Date only
    last_login: null                           # Null timestamp
```

### Testing Decimal and Numeric Types

```yaml
products:
  - id: 1
    name: "Widget"
    price: "19.99"          # String representation for precision
    discount: 0.15          # Float for percentage
    tax_rate: "0.0825"      # String for exact decimal
```

### Testing JSON Fields

```yaml
users:
  - id: 1
    name: "John Doe"
    metadata: |
      {
        "preferences": {
          "theme": "dark",
          "language": "en"
        },
        "features": ["beta", "premium"]
      }
    settings: '{"notifications": true, "privacy": "public"}'
```

## Mock Data Generation

### Realistic Test Data

```yaml
# Use realistic data that matches production patterns
users:
  - id: 1
    name: "Alice Johnson"
    email: "alice.johnson@techcorp.com"
    phone: "+1-555-0123"
    created_at: "2024-01-15T10:30:00Z"
    last_login: "2024-08-09T14:22:00Z"
    
  - id: 2
    name: "Bob Chen"
    email: "bob.chen@techcorp.com"
    phone: "+1-555-0124"
    created_at: "2024-02-20T09:15:00Z"
    last_login: "2024-08-08T16:45:00Z"
```

### Edge Cases Testing

```yaml
# Test edge cases and boundary conditions
users:
  - id: 1
    name: "Normal User"
    email: "normal@example.com"
    active: true
    
  - id: 2
    name: ""                    # Empty string
    email: null                 # Null value
    active: false
    
  - id: 3
    name: "Very Long Name That Exceeds Normal Length Expectations"
    email: "very.long.email.address.that.might.cause.issues@very-long-domain-name.example.com"
    active: true
    
  - id: 2147483647             # Max integer value
    name: "Edge Case User"
    email: "edge@example.com"
    active: true
```

## Integration with Testing Frameworks

### Go Testing Integration

```go
package queries_test

import (
    "context"
    "testing"
    
    "github.com/shibukawa/snapsql/langs/snapsqlgo"
    "github.com/stretchr/testify/assert"
)

func TestGetActiveUsers_WithMockData(t *testing.T) {
    // Create context with mock data
    ctx := snapsqlgo.WithConfig(context.Background(), "get_active_users", 
        snapsqlgo.WithMockData("engineering_department_with_email"))
    
    // Execute query with mock data
    users := make([]User, 0)
    for user, err := range GetActiveUsers(ctx, nil, 1, true, 10) {
        assert.NoError(t, err)
        users = append(users, user)
    }
    
    // Verify results
    assert.Len(t, users, 1)
    assert.Equal(t, "John Doe", users[0].Name)
    assert.Equal(t, "john@example.com", users[0].Email)
    assert.Equal(t, "Engineering", users[0].Department.Name)
}
```

### HTTP API Testing

```go
func TestUserListAPI_WithMockData(t *testing.T) {
    // Create context with mock data
    ctx := snapsqlgo.WithConfig(context.Background(), "get_user_list", 
        snapsqlgo.WithMockData("basic_user_list"))
    
    // Create HTTP request with mock context
    req := httptest.NewRequest("GET", "/api/users?include_email=true", nil)
    req = req.WithContext(ctx)
    
    // Create response recorder
    w := httptest.NewRecorder()
    
    // Call HTTP handler
    userListHandler(w, req)
    
    // Verify response
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response UserListResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Len(t, response.Users, 2)
}
```

## Best Practices

### 1. Comprehensive Test Coverage

```markdown
## Test Cases

### Test: Happy path
# Test the most common, successful scenario

### Test: Edge cases
# Test boundary conditions and limits

### Test: Error conditions
# Test invalid inputs and error scenarios

### Test: Empty results
# Test queries that return no data

### Test: Large datasets
# Test performance with larger mock datasets
```

### 2. Realistic Mock Data

```yaml
# ✅ Good: Realistic, production-like data
users:
  - id: 1
    name: "Alice Johnson"
    email: "alice.johnson@company.com"
    created_at: "2024-01-15T10:30:00Z"

# ❌ Bad: Unrealistic test data
users:
  - id: 1
    name: "test"
    email: "test@test.com"
    created_at: "2000-01-01T00:00:00Z"
```

### 3. Independent Test Cases

```yaml
# ✅ Good: Each test case has its own complete dataset
### Test: Active users
**Fixtures:**
users:
  - {id: 1, name: "John", active: true}
  - {id: 2, name: "Jane", active: false}

### Test: Inactive users  
**Fixtures:**
users:
  - {id: 3, name: "Bob", active: false}
  - {id: 4, name: "Alice", active: false}
```

### 4. Clear Test Names

```markdown
# ✅ Good: Descriptive test names
### Test: Active users in engineering department with email
### Test: Empty result when no users match criteria
### Test: Pagination with large dataset

# ❌ Bad: Vague test names
### Test: Test 1
### Test: Basic case
### Test: Edge case
```
