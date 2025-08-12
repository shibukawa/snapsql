# Best Practices

This guide covers recommended patterns, conventions, and practices for creating maintainable and efficient SnapSQL templates.

## Template Organization

### File Naming Conventions

Use descriptive, action-oriented names:

```
✅ Good:
- get_user_by_id.snap.sql
- create_order_with_items.snap.md
- update_user_profile.snap.sql
- list_active_products.snap.md

❌ Bad:
- user.snap.sql
- order.snap.md
- update.snap.sql
- list.snap.md
```

### Function Naming

Use consistent, descriptive function names:

```sql
/*#
function_name: get_user_by_id          # ✅ Clear action and target
function_name: find_active_orders      # ✅ Descriptive verb
function_name: create_user_account     # ✅ Specific action
function_name: update_order_status     # ✅ Clear what's being updated

function_name: user                    # ❌ Too vague
function_name: getData                 # ❌ Generic
function_name: process                 # ❌ Unclear action
*/
```

## SQL Template Design

### 2-Way SQL Principle

Always ensure templates work as valid SQL:

```sql
-- ✅ Good: Works as valid SQL during development
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    created_at
FROM users_/*= table_suffix */dev
WHERE active = /*= active */true
LIMIT /*= limit */10;

-- ❌ Bad: Breaks SQL syntax
SELECT 
    id,
    name,
    /*# if include_email */
    email
    /*# end */
FROM users_/*= table_suffix */
WHERE active = /*= active */
LIMIT /*= limit */;
```

### Meaningful Dummy Values

Use realistic dummy values that help during development:

```sql
-- ✅ Good: Realistic dummy values
WHERE user_id = /*= user_id */12345
AND email LIKE /*= email_pattern */'%@company.com'
AND created_at >= /*= start_date */'2024-01-01'
AND status IN /*= status_list */('active', 'pending')

-- ❌ Bad: Meaningless dummy values
WHERE user_id = /*= user_id */1
AND email LIKE /*= email_pattern */'%'
AND created_at >= /*= start_date */'1900-01-01'
AND status IN /*= status_list */('x', 'y')
```

### Parameter Design

Design parameters with clear types and validation:

```yaml
# ✅ Good: Clear, well-structured parameters
parameters:
  user_id: int                    # Simple type
  filters:                        # Grouped related parameters
    email_domain: string
    active_only: bool
    created_after: timestamp
  pagination:                     # Logical grouping
    limit: int
    offset: int
  sort_options:                   # Enumerated options
    field: string                 # "name", "email", "created_at"
    direction: string             # "asc", "desc"

# ❌ Bad: Flat, unclear parameters
parameters:
  user_id: int
  email_domain: string
  active_only: bool
  created_after: timestamp
  limit: int
  offset: int
  sort_field: string
  sort_dir: string
```

## Query Performance

### Efficient Joins

Structure joins for optimal performance:

```sql
-- ✅ Good: Efficient join structure
SELECT 
    u.id,
    u.name,
    u.email,
    d.name AS department__name,
    m.name AS department__manager__name
FROM users u
    INNER JOIN departments d ON u.department_id = d.id
    LEFT JOIN users m ON d.manager_id = m.id
WHERE u.active = true
    AND d.active = true;

-- ❌ Bad: Inefficient subqueries
SELECT 
    u.id,
    u.name,
    u.email,
    (SELECT name FROM departments WHERE id = u.department_id) AS department__name,
    (SELECT name FROM users WHERE id = (
        SELECT manager_id FROM departments WHERE id = u.department_id
    )) AS department__manager__name
FROM users u
WHERE u.active = true;
```

### Index-Friendly Conditions

Write conditions that can use database indexes:

```sql
-- ✅ Good: Index-friendly conditions
WHERE user_id = /*= user_id */123
AND created_at >= /*= start_date */'2024-01-01'
AND status = /*= status */'active'

-- ❌ Bad: Index-unfriendly conditions
WHERE UPPER(email) = /*= email */'USER@EXAMPLE.COM'
AND YEAR(created_at) = /*= year */2024
AND status || '_suffix' = /*= status_with_suffix */'active_suffix'
```

### Pagination Best Practices

Implement efficient pagination:

```sql
-- ✅ Good: Offset-based pagination for small offsets
SELECT id, name, email, created_at
FROM users
WHERE active = true
ORDER BY created_at DESC, id DESC
LIMIT /*= limit */20
OFFSET /*= offset */0;

-- ✅ Good: Cursor-based pagination for large datasets
SELECT id, name, email, created_at
FROM users
WHERE active = true
    /*# if cursor_id != null */
    AND (created_at, id) < (/*= cursor_date */'2024-01-01', /*= cursor_id */1000)
    /*# end */
ORDER BY created_at DESC, id DESC
LIMIT /*= limit */20;
```

## Type Safety and Validation

### Strong Parameter Types

Use specific types for better validation:

```yaml
# ✅ Good: Specific types
parameters:
  user_id: int                    # Positive integer
  email: string                   # Email format
  birth_date: date               # Date only
  salary: decimal                # Precise decimal
  tags: "string[]"               # Array of strings
  preferences: object            # Structured object

# ❌ Bad: Generic types
parameters:
  user_id: string                # Should be int
  email: string                  # No format validation
  birth_date: string             # Should be date
  salary: float                  # Precision issues
  tags: string                   # Should be array
  preferences: string            # Should be object
```

### Null Handling

Be explicit about nullable fields:

```sql
-- ✅ Good: Explicit null handling
SELECT 
    id,
    name,                          -- NOT NULL
    email,                         -- NULLABLE
    COALESCE(phone, '') AS phone,  -- Default for nullable
    last_login                     -- NULLABLE timestamp
FROM users;

-- ❌ Bad: Unclear null semantics
SELECT 
    id,
    name,
    email,
    phone,
    last_login
FROM users;
```

### Response Structure

Design clear response structures:

```sql
-- ✅ Good: Well-structured response
SELECT 
    u.id,
    u.name,
    u.email,
    u.created_at,
    d.id AS department__id,
    d.name AS department__name,
    d.budget AS department__budget,
    COUNT(p.id) AS project_count
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
GROUP BY u.id, u.name, u.email, u.created_at, d.id, d.name, d.budget;

-- ❌ Bad: Flat, unclear structure
SELECT 
    u.id AS user_id,
    u.name AS user_name,
    u.email AS user_email,
    u.created_at AS user_created,
    d.id AS dept_id,
    d.name AS dept_name,
    d.budget AS dept_budget,
    COUNT(p.id) AS proj_count
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
GROUP BY u.id, u.name, u.email, u.created_at, d.id, d.name, d.budget;
```

## Testing Strategies

### Comprehensive Test Coverage

Create tests for all scenarios:

````markdown
## Test Cases

### Test: Happy path - normal user retrieval
**Parameters:**
```yaml
user_id: 1
include_profile: true
```

### Test: Edge case - user without profile
**Parameters:**
```yaml
user_id: 2
include_profile: true
```

### Test: Error case - non-existent user
**Parameters:**
```yaml
user_id: 99999
include_profile: false
```

### Test: Boundary case - maximum limit
**Parameters:**
```yaml
limit: 1000
offset: 0
```

### Test: Empty result set
**Parameters:**
```yaml
filters:
  active: false
  created_after: "2030-01-01"
```
````

### Realistic Mock Data

Use production-like test data:

```yaml
# ✅ Good: Realistic mock data
users:
  - id: 1
    name: "Alice Johnson"
    email: "alice.johnson@techcorp.com"
    phone: "+1-555-0123"
    created_at: "2024-01-15T10:30:00Z"
    last_login: "2024-08-09T14:22:00Z"
    department_id: 1
    
  - id: 2
    name: "Bob Chen"
    email: "bob.chen@techcorp.com"
    phone: null
    created_at: "2024-02-20T09:15:00Z"
    last_login: "2024-08-08T16:45:00Z"
    department_id: 2

departments:
  - id: 1
    name: "Engineering"
    budget: 1500000.00
    manager_id: 1
    
  - id: 2
    name: "Design"
    budget: 800000.00
    manager_id: null

# ❌ Bad: Unrealistic mock data
users:
  - id: 1
    name: "test"
    email: "test@test.com"
    created_at: "2000-01-01T00:00:00Z"
    
departments:
  - id: 1
    name: "dept1"
```

## Documentation Standards

### Template Documentation

Use Markdown format for complex queries:

````markdown
---
function_name: get_user_analytics
description: Get comprehensive user analytics with performance metrics
---

# User Analytics Query

## Description

This query retrieves comprehensive user analytics including:
- Basic user information
- Activity metrics (login frequency, last activity)
- Performance indicators (tasks completed, success rate)
- Department and team associations

## Use Cases

- Admin dashboard user overview
- Performance review data collection
- User engagement analysis

## Parameters

```yaml
user_id: int                    # Target user ID
date_range:                     # Analysis period
  start: timestamp              # Start of analysis period
  end: timestamp                # End of analysis period
include_sensitive: bool         # Include salary and personal data
metrics:                        # Which metrics to include
  activity: bool                # Login and activity metrics
  performance: bool             # Task and success metrics
  social: bool                  # Team and collaboration metrics
```

## Performance Notes

- Query uses indexes on user_id, created_at, and department_id
- Expected execution time: < 50ms for single user
- Memory usage: ~1KB per user record

## SQL

```sql
-- Complex query implementation here
```
````

### Inline Comments

Add helpful comments in SQL:

```sql
-- User base query with department join
SELECT 
    u.id,
    u.name,
    u.email,
    
    -- Department information (always present)
    d.id AS department__id,
    d.name AS department__name,
    
    /*# if include_metrics */
    -- Performance metrics (optional)
    COALESCE(m.tasks_completed, 0) AS metrics__tasks_completed,
    COALESCE(m.success_rate, 0.0) AS metrics__success_rate,
    /*# end */
    
    -- Audit fields
    u.created_at,
    u.updated_at
    
FROM users u
    INNER JOIN departments d ON u.department_id = d.id
    /*# if include_metrics */
    LEFT JOIN user_metrics m ON u.id = m.user_id 
        AND m.period_start >= /*= date_range.start */'2024-01-01'
        AND m.period_end <= /*= date_range.end */'2024-12-31'
    /*# end */
WHERE u.id = /*= user_id */1;
```

## Security Considerations

### Parameter Validation

Always validate input parameters:

```sql
/*#
function_name: get_user_data
parameters:
  user_id: int                  # Must be positive
  access_level: string          # Must be: "public", "internal", "admin"
  limit: int                    # Must be 1-1000
*/
SELECT 
    id,
    name,
    /*# if access_level == "internal" || access_level == "admin" */
    email,
    phone,
    /*# end */
    /*# if access_level == "admin" */
    ssn,
    salary,
    /*# end */
    created_at
FROM users
WHERE id = /*= user_id */1
    AND active = true
LIMIT /*= limit > 0 && limit <= 1000 ? limit : 10 */10;
```

### SQL Injection Prevention

SnapSQL prevents SQL injection through parameterization, but follow these practices:

```sql
-- ✅ Good: Parameterized values
WHERE name = /*= user_name */'John Doe'
AND status IN /*= status_list */('active', 'pending')

-- ✅ Good: Safe string operations in CEL
WHERE email LIKE /*= "%" + email_domain + "%" */'%@example.com'

-- ❌ Bad: Don't try to build raw SQL (won't work anyway)
-- WHERE /*= "name = '" + user_name + "'" */"name = 'John'"
```

### Access Control

Implement access control in templates:

```sql
-- Role-based data access
SELECT 
    u.id,
    u.name,
    /*# if user_role == "admin" || user_role == "manager" */
    u.email,
    u.phone,
    u.salary,
    /*# end */
    /*# if user_role == "admin" */
    u.ssn,
    u.notes,
    /*# end */
    u.created_at
FROM users u
WHERE u.id = /*= target_user_id */1
    /*# if user_role != "admin" */
    -- Non-admins can only see users in their department
    AND u.department_id = /*= current_user_department_id */1
    /*# end */;
```

## Error Handling

### Graceful Degradation

Handle missing data gracefully:

```sql
SELECT 
    u.id,
    u.name,
    u.email,
    
    -- Graceful handling of optional relationships
    COALESCE(d.name, 'No Department') AS department__name,
    COALESCE(m.name, 'No Manager') AS manager__name,
    
    -- Default values for metrics
    COALESCE(s.login_count, 0) AS stats__login_count,
    COALESCE(s.last_login, u.created_at) AS stats__last_activity
    
FROM users u
    LEFT JOIN departments d ON u.department_id = d.id
    LEFT JOIN users m ON d.manager_id = m.id
    LEFT JOIN user_stats s ON u.id = s.user_id;
```

### Validation in Templates

Add validation logic where appropriate:

```sql
/*#
function_name: transfer_funds
parameters:
  from_account_id: int
  to_account_id: int
  amount: decimal
*/
-- Validate accounts exist and amount is positive
SELECT 
    CASE 
        WHEN /*= from_account_id */1 = /*= to_account_id */2 
        THEN 'ERROR: Cannot transfer to same account'
        WHEN /*= amount */100.00 <= 0 
        THEN 'ERROR: Amount must be positive'
        ELSE 'OK'
    END AS validation_result,
    
    -- Only proceed if validation passes
    /*# if from_account_id != to_account_id && amount > 0 */
    fa.balance AS from_balance,
    ta.balance AS to_balance
    /*# end */
    
FROM accounts fa
    /*# if from_account_id != to_account_id && amount > 0 */
    CROSS JOIN accounts ta
    /*# end */
WHERE fa.id = /*= from_account_id */1
    /*# if from_account_id != to_account_id && amount > 0 */
    AND ta.id = /*= to_account_id */2
    /*# end */;
```

## Maintenance and Evolution

### Version Control

Structure templates for clean version control:

```sql
-- Keep related logic together for better diffs
/*#
function_name: get_user_profile
description: Get user profile with optional sections
parameters:
  user_id: int
  include_preferences: bool
  include_activity: bool
*/

-- Base user data (stable)
SELECT 
    u.id,
    u.name,
    u.email,
    u.created_at,
    
    -- Optional preferences section
    /*# if include_preferences */
    p.theme AS preferences__theme,
    p.language AS preferences__language,
    p.timezone AS preferences__timezone,
    /*# end */
    
    -- Optional activity section  
    /*# if include_activity */
    a.last_login AS activity__last_login,
    a.login_count AS activity__login_count,
    a.page_views AS activity__page_views
    /*# end */
    
FROM users u
    /*# if include_preferences */
    LEFT JOIN user_preferences p ON u.id = p.user_id
    /*# end */
    /*# if include_activity */
    LEFT JOIN user_activity a ON u.id = a.user_id
    /*# end */
WHERE u.id = /*= user_id */1;
```

### Backward Compatibility

When evolving templates, maintain backward compatibility:

```sql
/*#
function_name: get_user_list
parameters:
  # Legacy parameter (deprecated but supported)
  active_only: bool
  
  # New structured filters (preferred)
  filters:
    active: bool
    verified: bool
    created_after: timestamp
    
  # Pagination (enhanced)
  pagination:
    limit: int
    offset: int
*/

SELECT id, name, email, active, verified, created_at
FROM users
WHERE 1=1
    -- Support both old and new parameter styles
    /*# if active_only != null */
    AND active = /*= active_only */true
    /*# else if filters.active != null */
    AND active = /*= filters.active */true
    /*# end */
    
    /*# if filters.verified != null */
    AND verified = /*= filters.verified */true
    /*# end */
    
    /*# if filters.created_after != null */
    AND created_at >= /*= filters.created_after */'2024-01-01'
    /*# end */
    
ORDER BY created_at DESC
LIMIT /*= pagination.limit ?? 10 */10
OFFSET /*= pagination.offset ?? 0 */0;
```

## System Columns Best Practices

### Consistent Configuration

Use standardized system column definitions across your application:

```yaml
# Standard system columns configuration
system:
  fields:
    # Audit timestamps
    - name: created_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
    - name: updated_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
      on_update:
        parameter: implicit
        default: "NOW()"
    
    # User tracking
    - name: created_by
      type: int
      on_insert:
        parameter: implicit
    - name: updated_by
      type: int
      on_update:
        parameter: implicit
    
    # Optimistic locking
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1
      on_update:
        parameter: implicit
    
    # Soft delete support
    - name: deleted_at
      type: timestamp
      on_update:
        parameter: implicit  # Only set when deleting
```

### Context Management

Set system values at application boundaries:

```go
// ✅ Good: Set context in middleware
func withUserContext(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserIDFromToken(r)
        ctx := snapsqlgo.WithSystemValue(r.Context(), "created_by", userID)
        ctx = snapsqlgo.WithSystemValue(ctx, "updated_by", userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// ✅ Good: Set context for background jobs
func processBackgroundJob(ctx context.Context) {
    ctx = snapsqlgo.WithSystemValue(ctx, "created_by", SYSTEM_USER_ID)
    ctx = snapsqlgo.WithSystemValue(ctx, "updated_by", SYSTEM_USER_ID)
    // ... perform operations
}

// ❌ Bad: Setting context values in business logic
func createUser(ctx context.Context, name, email string) error {
    // Don't do this - context should be set at boundaries
    ctx = snapsqlgo.WithSystemValue(ctx, "created_by", getCurrentUser())
    return CreateUser(ctx, db, name, email)
}
```

### Template Design

Keep templates clean by relying on system column automation:

```sql
-- ✅ Good: Let system columns be handled automatically
/*#
function_name: create_user
parameters:
  name: string
  email: string
*/
INSERT INTO users (name, email) 
VALUES (/*= name */'John', /*= email */'john@example.com');

-- ✅ Good: Focus on business logic, not system columns
/*#
function_name: update_user_profile
parameters:
  user_id: int
  name: string
  email: string
*/
UPDATE users 
SET name = /*= name */'John', email = /*= email */'john@example.com'
WHERE id = /*= user_id */1;

-- ❌ Bad: Manually handling system columns
/*#
function_name: create_user_manual
parameters:
  name: string
  email: string
  created_by: int
  created_at: timestamp
*/
INSERT INTO users (name, email, created_by, created_at) 
VALUES (/*= name */'John', /*= email */'john@example.com', /*= created_by */1, /*= created_at */'2024-01-01');
```

### Testing Strategy

Use mock data that includes realistic system column values:

```yaml
# ✅ Good: Realistic system column values in tests
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    created_at: "2024-01-15T10:30:00Z"
    updated_at: "2024-01-15T10:30:00Z"
    created_by: 123
    updated_by: 123
    version: 1
    
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    created_at: "2024-01-14T09:15:00Z"
    updated_at: "2024-01-16T14:20:00Z"  # Updated later
    created_by: 456
    updated_by: 789  # Updated by different user
    version: 3  # Multiple updates

# ❌ Bad: Unrealistic or missing system column values
users:
  - id: 1
    name: "test user"
    email: "test@test.com"
    # Missing system columns
```

### Migration Strategy

When adding system columns to existing tables:

```sql
-- ✅ Good: Add system columns with appropriate defaults
ALTER TABLE users ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN created_by INTEGER;
ALTER TABLE users ADD COLUMN version INTEGER DEFAULT 1;

-- Update existing records with reasonable defaults
UPDATE users 
SET created_by = 0,  -- System user ID for existing records
    updated_by = 0
WHERE created_by IS NULL;

-- Add constraints after data migration
ALTER TABLE users ALTER COLUMN created_by SET NOT NULL;
```

### Error Handling

Handle system column errors gracefully:

```go
// ✅ Good: Graceful error handling
func createUser(ctx context.Context, name, email string) error {
    defer func() {
        if r := recover(); r != nil {
            if strings.Contains(fmt.Sprint(r), "required system field") {
                log.Error("Missing required system field - check context setup")
                // Handle gracefully or re-panic with better context
            }
        }
    }()
    
    _, err := CreateUser(ctx, db, name, email)
    return err
}

// ✅ Good: Validate context before operations
func validateSystemContext(ctx context.Context) error {
    if snapsqlgo.GetSystemValueFromContext(ctx, "created_by") == nil {
        return errors.New("created_by not set in context")
    }
    return nil
}
```

### Performance Considerations

System columns can impact performance - optimize accordingly:

```sql
-- ✅ Good: Index system columns used in queries
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_created_by ON users(created_by);
CREATE INDEX idx_users_updated_at ON users(updated_at) WHERE deleted_at IS NULL;

-- ✅ Good: Use system columns for efficient filtering
/*#
function_name: get_recent_users
parameters:
  days: int
  created_by: int
*/
SELECT id, name, email, created_at
FROM users
WHERE created_at >= NOW() - INTERVAL /*= days */7 DAY
    /*# if created_by != null */
    AND created_by = /*= created_by */123
    /*# end */
ORDER BY created_at DESC;
```

### Audit Trail Implementation

Use system columns for comprehensive audit trails:

```sql
-- ✅ Good: Audit-friendly update pattern
/*#
function_name: update_user_with_audit
parameters:
  user_id: int
  name: string
  email: string
  change_reason: string
*/
-- First, capture the old values for audit
WITH old_values AS (
    SELECT name, email, version
    FROM users 
    WHERE id = /*= user_id */1
),
-- Update the record (system columns handled automatically)
updated AS (
    UPDATE users 
    SET name = /*= name */'John', 
        email = /*= email */'john@example.com'
    WHERE id = /*= user_id */1
    RETURNING id, name, email, version, updated_by, updated_at
)
-- Create audit trail entry
INSERT INTO audit_log (
    table_name, record_id, action, 
    old_values, new_values, change_reason
)
SELECT 
    'users',
    /*= user_id */1,
    'UPDATE',
    json_build_object('name', ov.name, 'email', ov.email, 'version', ov.version),
    json_build_object('name', u.name, 'email', u.email, 'version', u.version),
    /*= change_reason */'Profile update'
FROM old_values ov
CROSS JOIN updated u;
```

### Multi-Tenant Applications

Handle system columns in multi-tenant scenarios:

```yaml
# Multi-tenant system columns
system:
  fields:
    - name: tenant_id
      type: string
      on_insert:
        parameter: implicit
      on_update:
        parameter: implicit  # Ensure tenant consistency
    - name: created_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
    - name: created_by
      type: int
      on_insert:
        parameter: implicit
```

```go
// Set tenant context
func withTenantContext(ctx context.Context, tenantID string, userID int) context.Context {
    ctx = snapsqlgo.WithSystemValue(ctx, "tenant_id", tenantID)
    ctx = snapsqlgo.WithSystemValue(ctx, "created_by", userID)
    ctx = snapsqlgo.WithSystemValue(ctx, "updated_by", userID)
    return ctx
}
```

These system column best practices ensure consistent, maintainable, and secure handling of common database patterns while leveraging SnapSQL's automation capabilities.
