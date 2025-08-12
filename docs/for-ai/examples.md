# Examples

This document provides practical examples of common SnapSQL patterns and use cases.

## Basic CRUD Operations

### Create (INSERT)

```sql
/*#
function_name: create_user
description: Create a new user account
parameters:
  name: string
  email: string
  department_id: int
response_affinity: none
*/
INSERT INTO users (name, email, department_id)
VALUES (
    /*= name */'John Doe',
    /*= email */'john@example.com',
    /*= department_id */1
);
```

### Read (SELECT)

```sql
/*#
function_name: get_user_by_id
description: Retrieve user by ID with optional profile data
parameters:
  user_id: int
  include_profile: bool
response_affinity: one
*/
SELECT 
    u.id,
    u.name,
    u.email,
    u.created_at,
    /*# if include_profile */
    p.bio AS profile__bio,
    p.avatar_url AS profile__avatar_url,
    p.phone AS profile__phone
    /*# end */
FROM users u
    /*# if include_profile */
    LEFT JOIN profiles p ON u.id = p.user_id
    /*# end */
WHERE u.id = /*= user_id */1;
```

### Update

```sql
/*#
function_name: update_user_profile
description: Update user profile information
parameters:
  user_id: int
  name: string
  email: string
  phone: string
response_affinity: none
*/
UPDATE users 
SET 
    name = /*= name */'John Doe',
    email = /*= email */'john@example.com',
    phone = /*= phone */'+1-555-0123',
    updated_at = NOW()
WHERE id = /*= user_id */1;
```

### Delete

```sql
/*#
function_name: deactivate_user
description: Soft delete user by marking as inactive
parameters:
  user_id: int
  reason: string
response_affinity: none
*/
UPDATE users 
SET 
    active = false,
    deactivated_at = NOW(),
    deactivation_reason = /*= reason */'Account closed by user',
    updated_at = NOW()
WHERE id = /*= user_id */1;
```

## Complex Queries

### Multi-table Joins with Nested Structures

````markdown
---
function_name: get_user_with_department_and_projects
---

# User with Department and Projects

## Description
Get comprehensive user information including department details and associated projects.

## Parameters
```yaml
user_id: int
include_inactive_projects: bool
```

## SQL
```sql
SELECT 
    u.id,
    u.name,
    u.email,
    u.created_at,
    
    -- Department information
    d.id AS department__id,
    d.name AS department__name,
    d.budget AS department__budget,
    
    -- Manager information
    m.id AS department__manager__id,
    m.name AS department__manager__name,
    m.email AS department__manager__email,
    
    -- Project information
    p.id AS projects__id,
    p.name AS projects__name,
    p.status AS projects__status,
    p.start_date AS projects__start_date,
    p.end_date AS projects__end_date
    
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN users m ON d.manager_id = m.id
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
        /*# if !include_inactive_projects */
        AND p.status IN ('active', 'planning')
        /*# end */
WHERE u.id = /*= user_id */1
ORDER BY p.start_date DESC;
```

## Test Cases

### Test: User with active projects only

**Fixtures:**
```yaml
users:
  - {id: 1, name: "Alice Johnson", email: "alice@company.com", department_id: 1}

departments:
  - {id: 1, name: "Engineering", budget: 1000000, manager_id: 2}

users:
  - {id: 2, name: "Bob Manager", email: "bob@company.com", department_id: 1}

projects:
  - {id: 101, name: "Web App", status: "active", start_date: "2024-01-01", end_date: "2024-06-01"}
  - {id: 102, name: "Mobile App", status: "completed", start_date: "2023-06-01", end_date: "2023-12-01"}

user_projects:
  - {user_id: 1, project_id: 101}
  - {user_id: 1, project_id: 102}
```

**Parameters:**
```yaml
user_id: 1
include_inactive_projects: false
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice Johnson"
  email: "alice@company.com"
  department:
    id: 1
    name: "Engineering"
    budget: 1000000
    manager:
      id: 2
      name: "Bob Manager"
      email: "bob@company.com"
  projects:
    - id: 101
      name: "Web App"
      status: "active"
      start_date: "2024-01-01"
      end_date: "2024-06-01"
```
````

### Aggregation and Analytics

```sql
/*#
function_name: get_department_analytics
description: Get department analytics with user counts and project metrics
parameters:
  department_id: int
  date_range:
    start: timestamp
    end: timestamp
response_affinity: one
*/
SELECT 
    d.id,
    d.name,
    d.budget,
    
    -- User metrics
    COUNT(DISTINCT u.id) AS user_count,
    COUNT(DISTINCT CASE WHEN u.active THEN u.id END) AS active_user_count,
    
    -- Project metrics
    COUNT(DISTINCT p.id) AS total_projects,
    COUNT(DISTINCT CASE WHEN p.status = 'active' THEN p.id END) AS active_projects,
    COUNT(DISTINCT CASE WHEN p.status = 'completed' THEN p.id END) AS completed_projects,
    
    -- Financial metrics
    COALESCE(SUM(p.budget), 0) AS total_project_budget,
    COALESCE(AVG(p.budget), 0) AS avg_project_budget,
    
    -- Time metrics
    AVG(EXTRACT(DAYS FROM (p.end_date - p.start_date))) AS avg_project_duration_days
    
FROM departments d
    LEFT JOIN users u ON d.id = u.department_id
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
        AND p.created_at BETWEEN /*= date_range.start */'2024-01-01' 
                             AND /*= date_range.end */'2024-12-31'
WHERE d.id = /*= department_id */1
GROUP BY d.id, d.name, d.budget;
```

## Dynamic Filtering

### Advanced Search with Multiple Filters

```sql
/*#
function_name: search_users
description: Advanced user search with multiple optional filters
parameters:
  filters:
    name: string
    email: string
    department_ids: "int[]"
    active: bool
    created_after: timestamp
    created_before: timestamp
    has_projects: bool
  sort:
    field: string  # "name", "email", "created_at"
    direction: string  # "asc", "desc"
  pagination:
    limit: int
    offset: int
response_affinity: many
*/
SELECT 
    u.id,
    u.name,
    u.email,
    u.active,
    u.created_at,
    d.name AS department__name,
    COUNT(DISTINCT p.id) AS project_count
FROM users u
    LEFT JOIN departments d ON u.department_id = d.id
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
WHERE 1=1
    /*# if filters.name != "" */
    AND u.name ILIKE /*= "%" + filters.name + "%" */'%john%'
    /*# end */
    
    /*# if filters.email != "" */
    AND u.email ILIKE /*= "%" + filters.email + "%" */'%@company.com%'
    /*# end */
    
    /*# if size(filters.department_ids) > 0 */
    AND u.department_id IN /*= filters.department_ids */[1, 2, 3]
    /*# end */
    
    /*# if filters.active != null */
    AND u.active = /*= filters.active */true
    /*# end */
    
    /*# if filters.created_after != null */
    AND u.created_at >= /*= filters.created_after */'2024-01-01'
    /*# end */
    
    /*# if filters.created_before != null */
    AND u.created_at <= /*= filters.created_before */'2024-12-31'
    /*# end */
    
GROUP BY u.id, u.name, u.email, u.active, u.created_at, d.name

/*# if filters.has_projects != null */
HAVING 
    /*# if filters.has_projects */
    COUNT(DISTINCT p.id) > 0
    /*# else */
    COUNT(DISTINCT p.id) = 0
    /*# end */
/*# end */

ORDER BY 
/*# if sort.field == "name" */
    u.name /*= sort.direction == "desc" ? "DESC" : "ASC" */'ASC'
/*# else if sort.field == "email" */
    u.email /*= sort.direction == "desc" ? "DESC" : "ASC" */'ASC'
/*# else */
    u.created_at /*= sort.direction == "desc" ? "DESC" : "ASC" */'DESC'
/*# end */

LIMIT /*= pagination.limit */20
OFFSET /*= pagination.offset */0;
```

## Batch Operations

### Bulk Insert with Loop

```sql
/*#
function_name: create_multiple_users
description: Create multiple users in a single transaction
parameters:
  users:
    - name: string
      email: string
      department_id: int
response_affinity: none
*/
INSERT INTO users (name, email, department_id)
VALUES
/*# for user : users */
(
    /*= user.name */'John Doe',
    /*= user.email */'john@example.com',
    /*= user.department_id */1
)/*# if !last(users, user) */,/*# end */
/*# end */;
```

### Conditional Bulk Update

```sql
/*#
function_name: update_user_statuses
description: Update multiple users with different status changes
parameters:
  updates:
    - user_id: int
      active: bool
      reason: string
response_affinity: none
*/
/*# for update : updates */
UPDATE users 
SET 
    active = /*= update.active */true,
    status_change_reason = /*= update.reason */'Bulk update',
    updated_at = NOW()
WHERE id = /*= update.user_id */1;
/*# end */
```

## Reporting Queries

### Time-Series Analytics

````markdown
---
function_name: get_user_registration_trends
---

# User Registration Trends

## Description
Get user registration trends over time with department breakdown.

## Parameters
```yaml
date_range:
  start: timestamp
  end: timestamp
group_by: string  # "day", "week", "month"
department_ids: "int[]"
```

## SQL
```sql
SELECT 
    /*# if group_by == "day" */
    DATE_TRUNC('day', u.created_at) AS period,
    /*# else if group_by == "week" */
    DATE_TRUNC('week', u.created_at) AS period,
    /*# else */
    DATE_TRUNC('month', u.created_at) AS period,
    /*# end */
    
    d.id AS department__id,
    d.name AS department__name,
    
    COUNT(*) AS registration_count,
    COUNT(CASE WHEN u.active THEN 1 END) AS active_count,
    COUNT(CASE WHEN NOT u.active THEN 1 END) AS inactive_count,
    
    -- Running totals
    SUM(COUNT(*)) OVER (
        PARTITION BY d.id 
        ORDER BY DATE_TRUNC(/*= group_by */'month', u.created_at)
    ) AS cumulative_registrations

FROM users u
    JOIN departments d ON u.department_id = d.id
WHERE u.created_at BETWEEN /*= date_range.start */'2024-01-01' 
                       AND /*= date_range.end */'2024-12-31'
    /*# if size(department_ids) > 0 */
    AND d.id IN /*= department_ids */[1, 2, 3]
    /*# end */
    
GROUP BY 
    /*# if group_by == "day" */
    DATE_TRUNC('day', u.created_at),
    /*# else if group_by == "week" */
    DATE_TRUNC('week', u.created_at),
    /*# else */
    DATE_TRUNC('month', u.created_at),
    /*# end */
    d.id, d.name
    
ORDER BY period ASC, d.name ASC;
```
````

### Dashboard Summary

```sql
/*#
function_name: get_dashboard_summary
description: Get high-level dashboard metrics
parameters:
  date_range:
    start: timestamp
    end: timestamp
response_affinity: one
*/
SELECT 
    -- User metrics
    (SELECT COUNT(*) FROM users WHERE active = true) AS active_users,
    (SELECT COUNT(*) FROM users WHERE created_at >= /*= date_range.start */'2024-01-01') AS new_users_period,
    
    -- Project metrics
    (SELECT COUNT(*) FROM projects WHERE status = 'active') AS active_projects,
    (SELECT COUNT(*) FROM projects WHERE status = 'completed' 
     AND end_date >= /*= date_range.start */'2024-01-01') AS completed_projects_period,
    
    -- Department metrics
    (SELECT COUNT(*) FROM departments WHERE active = true) AS active_departments,
    (SELECT AVG(budget) FROM departments WHERE active = true) AS avg_department_budget,
    
    -- Financial metrics
    (SELECT SUM(budget) FROM projects WHERE status IN ('active', 'planning')) AS total_active_budget,
    (SELECT SUM(budget) FROM projects WHERE status = 'completed' 
     AND end_date >= /*= date_range.start */'2024-01-01') AS completed_budget_period,
    
    -- Activity metrics
    (SELECT COUNT(DISTINCT user_id) FROM user_activity 
     WHERE activity_date >= /*= date_range.start */'2024-01-01') AS active_users_period,
    (SELECT AVG(login_count) FROM user_activity 
     WHERE activity_date >= /*= date_range.start */'2024-01-01') AS avg_logins_per_user;
```

## API Integration Patterns

### Paginated List with Metadata

````markdown
---
function_name: get_paginated_users
---

# Paginated User List

## Description
Get paginated user list with total count and metadata for API responses.

## Parameters
```yaml
filters:
  search: string
  department_id: int
  active: bool
pagination:
  page: int
  page_size: int
sort:
  field: string
  direction: string
```

## SQL
```sql
WITH filtered_users AS (
    SELECT u.*
    FROM users u
        LEFT JOIN departments d ON u.department_id = d.id
    WHERE 1=1
        /*# if filters.search != "" */
        AND (u.name ILIKE /*= "%" + filters.search + "%" */'%john%' 
             OR u.email ILIKE /*= "%" + filters.search + "%" */'%john%')
        /*# end */
        
        /*# if filters.department_id != null */
        AND u.department_id = /*= filters.department_id */1
        /*# end */
        
        /*# if filters.active != null */
        AND u.active = /*= filters.active */true
        /*# end */
),
user_count AS (
    SELECT COUNT(*) as total_count
    FROM filtered_users
)
SELECT 
    -- User data
    u.id,
    u.name,
    u.email,
    u.active,
    u.created_at,
    d.name AS department__name,
    
    -- Metadata for pagination
    uc.total_count,
    /*= pagination.page */1 AS current_page,
    /*= pagination.page_size */20 AS page_size,
    CEIL(uc.total_count::float / /*= pagination.page_size */20) AS total_pages,
    
    -- Navigation helpers
    CASE 
        WHEN /*= pagination.page */1 > 1 THEN /*= pagination.page - 1 */0
        ELSE NULL 
    END AS previous_page,
    CASE 
        WHEN /*= pagination.page */1 < CEIL(uc.total_count::float / /*= pagination.page_size */20) 
        THEN /*= pagination.page + 1 */2
        ELSE NULL 
    END AS next_page

FROM filtered_users u
    LEFT JOIN departments d ON u.department_id = d.id
    CROSS JOIN user_count uc
ORDER BY 
    /*# if sort.field == "name" */
    u.name /*= sort.direction == "desc" ? "DESC" : "ASC" */'ASC'
    /*# else if sort.field == "email" */
    u.email /*= sort.direction == "desc" ? "DESC" : "ASC" */'ASC'
    /*# else */
    u.created_at /*= sort.direction == "desc" ? "DESC" : "ASC" */'DESC'
    /*# end */
LIMIT /*= pagination.page_size */20
OFFSET /*= (pagination.page - 1) * pagination.page_size */0;
```
````

### Hierarchical Data

```sql
/*#
function_name: get_department_hierarchy
description: Get department hierarchy with employee counts
parameters:
  root_department_id: int
  max_depth: int
response_affinity: many
*/
WITH RECURSIVE department_tree AS (
    -- Base case: root department
    SELECT 
        d.id,
        d.name,
        d.parent_id,
        d.manager_id,
        0 as depth,
        ARRAY[d.id] as path
    FROM departments d
    WHERE d.id = /*= root_department_id */1
    
    UNION ALL
    
    -- Recursive case: child departments
    SELECT 
        d.id,
        d.name,
        d.parent_id,
        d.manager_id,
        dt.depth + 1,
        dt.path || d.id
    FROM departments d
        JOIN department_tree dt ON d.parent_id = dt.id
    WHERE dt.depth < /*= max_depth */5
)
SELECT 
    dt.id,
    dt.name,
    dt.parent_id,
    dt.depth,
    dt.path,
    
    -- Manager information
    m.name AS manager__name,
    m.email AS manager__email,
    
    -- Employee counts
    COUNT(DISTINCT u.id) AS employee_count,
    COUNT(DISTINCT CASE WHEN u.active THEN u.id END) AS active_employee_count,
    
    -- Child department count
    COUNT(DISTINCT child.id) AS child_department_count

FROM department_tree dt
    LEFT JOIN users m ON dt.manager_id = m.id
    LEFT JOIN users u ON dt.id = u.department_id
    LEFT JOIN departments child ON dt.id = child.parent_id
GROUP BY dt.id, dt.name, dt.parent_id, dt.depth, dt.path, m.name, m.email
ORDER BY dt.path;
```

## Performance Optimization Examples

### Efficient Pagination with Cursor

```sql
/*#
function_name: get_users_cursor_paginated
description: Cursor-based pagination for large datasets
parameters:
  cursor:
    created_at: timestamp
    id: int
  limit: int
  direction: string  # "next", "prev"
response_affinity: many
*/
SELECT 
    id,
    name,
    email,
    created_at,
    active
FROM users
WHERE 
    /*# if direction == "next" */
        /*# if cursor.created_at != null && cursor.id != null */
        (created_at, id) > (/*= cursor.created_at */'2024-01-01', /*= cursor.id */1)
        /*# end */
    /*# else */
        /*# if cursor.created_at != null && cursor.id != null */
        (created_at, id) < (/*= cursor.created_at */'2024-01-01', /*= cursor.id */1)
        /*# end */
    /*# end */
ORDER BY 
    /*# if direction == "next" */
    created_at ASC, id ASC
    /*# else */
    created_at DESC, id DESC
    /*# end */
LIMIT /*= limit */20;
```

### Optimized Aggregation

```sql
/*#
function_name: get_user_stats_optimized
description: Optimized user statistics with selective aggregation
parameters:
  department_ids: "int[]"
  include_project_stats: bool
  include_activity_stats: bool
response_affinity: many
*/
SELECT 
    d.id,
    d.name,
    
    -- Basic user counts (always included)
    COUNT(DISTINCT u.id) AS total_users,
    COUNT(DISTINCT CASE WHEN u.active THEN u.id END) AS active_users,
    
    /*# if include_project_stats */
    -- Project statistics (expensive, optional)
    COUNT(DISTINCT p.id) AS total_projects,
    COUNT(DISTINCT CASE WHEN p.status = 'active' THEN p.id END) AS active_projects,
    COALESCE(AVG(p.budget), 0) AS avg_project_budget,
    /*# end */
    
    /*# if include_activity_stats */
    -- Activity statistics (expensive, optional)
    COALESCE(AVG(a.login_count), 0) AS avg_monthly_logins,
    COUNT(DISTINCT CASE WHEN a.last_login >= NOW() - INTERVAL '30 days' THEN u.id END) AS recently_active_users
    /*# end */

FROM departments d
    LEFT JOIN users u ON d.id = u.department_id
    /*# if include_project_stats */
    LEFT JOIN user_projects up ON u.id = up.user_id
    LEFT JOIN projects p ON up.project_id = p.id
    /*# end */
    /*# if include_activity_stats */
    LEFT JOIN user_activity a ON u.id = a.user_id 
        AND a.month = DATE_TRUNC('month', NOW())
    /*# end */
WHERE d.id IN /*= department_ids */[1, 2, 3]
GROUP BY d.id, d.name
ORDER BY d.name;
```

## System Columns Examples

### Basic INSERT with System Columns

````markdown
---
function_name: create_user_with_system_columns
---

# Create User with System Columns

## Description
Create a new user with automatic system column handling.

## Parameters
```yaml
name: string
email: string
```

## SQL
```sql
INSERT INTO users (name, email) VALUES (/*= name */'John Doe', /*= email */'john@example.com')
```
````

**Configuration (snapsql.yaml):**
```yaml
system:
  fields:
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
    - name: created_by
      type: int
      on_insert:
        parameter: implicit
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1
```

**Generated SQL:**
```sql
INSERT INTO users (name, email, created_at, updated_at, created_by, version) 
VALUES ($1, $2, $3, $4, $5, $6)
```

**Generated Go Code:**
```go
func CreateUserWithSystemColumns(ctx context.Context, executor snapsqlgo.DBExecutor, name string, email string) (sql.Result, error) {
    // Extract implicit parameters
    implicitSpecs := []snapsqlgo.ImplicitParamSpec{
        {Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
        {Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
        {Name: "created_by", Type: "int", Required: true},
        {Name: "version", Type: "int", Required: false, DefaultValue: 1},
    }
    systemValues := snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)
    
    // Execute with system values
    args := []any{name, email, systemValues["created_at"], systemValues["updated_at"], systemValues["created_by"], systemValues["version"]}
    // ... execute query
}
```

### UPDATE with System Columns

```sql
/*#
function_name: update_user_profile
description: Update user profile with automatic version increment
parameters:
  user_id: int
  name: string
  email: string
*/
UPDATE users 
SET 
    name = /*= name */'John Doe',
    email = /*= email */'john@example.com'
WHERE id = /*= user_id */1;
```

**Generated SQL:**
```sql
UPDATE users 
SET name = $1, email = $2, updated_at = $3, version = $4
WHERE id = $5
```

### Context-Based System Values

**Application Code:**
```go
// Set system values in context
ctx := context.Background()
ctx = snapsqlgo.WithSystemValue(ctx, "created_by", 123)

// Create user - system columns automatically handled
result, err := CreateUserWithSystemColumns(ctx, db, "John Doe", "john@example.com")
```

### Explicit System Columns

**Configuration for explicit parameters:**
```yaml
system:
  fields:
    - name: created_by
      type: int
      on_insert:
        parameter: explicit  # Must be provided as function parameter
```

**Generated Function Signature:**
```go
func CreateUser(ctx context.Context, executor snapsqlgo.DBExecutor, name string, email string, createdBy int) (sql.Result, error)
```

### System Columns in Test Cases

````markdown
## Test Cases

### Test: Create user with system columns

**Parameters:**
```yaml
name: "John Doe"
email: "john@example.com"
```

**Expected Results:**
```yaml
- {id: 1, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z", created_by: 123, version: 1}
```
````

### Conditional System Columns

**Configuration for different operations:**
```yaml
system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:  # Only applied to INSERT statements
        parameter: implicit
        default: "NOW()"
    - name: updated_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
      on_update:  # Applied to both INSERT and UPDATE
        parameter: implicit
        default: "NOW()"
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1
      on_update:
        parameter: implicit  # Version increment handled by application logic
```

### Audit Trail with System Columns

```sql
/*#
function_name: create_audit_entry
description: Create audit trail entry with system columns
parameters:
  table_name: string
  record_id: int
  action: string
  old_values: string
  new_values: string
*/
INSERT INTO audit_log (
    table_name, 
    record_id, 
    action, 
    old_values, 
    new_values
) VALUES (
    /*= table_name */'users',
    /*= record_id */123,
    /*= action */'UPDATE',
    /*= old_values */'{"name": "Old Name"}',
    /*= new_values */'{"name": "New Name"}'
);
```

**With system columns, this becomes:**
```sql
INSERT INTO audit_log (
    table_name, record_id, action, old_values, new_values,
    created_at, created_by, version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
```

These examples demonstrate common patterns and best practices for building robust, efficient SnapSQL templates that can handle real-world application requirements.
