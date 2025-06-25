---
name: "user search"
dialect: "postgres"
---

# User Search Query

## Overview
Searches for active users based on various criteria with pagination support.
Supports department filtering and sorting functionality.

## Parameters
```yaml
user_id: int
filters:
  active: bool
  departments: [str]
  name_pattern: str
pagination:
  limit: int
  offset: int
sort_by: str
include_email: bool
table_suffix: str
```

## SQL
```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    department,
    created_at
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
    /*# if filters.name_pattern */
    AND name ILIKE /*= filters.name_pattern */'%john%'
    /*# end */
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'engineering', 'design')
    /*# end */
/*# if sort_by */
ORDER BY /*= sort_by */created_at DESC
/*# end */
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

## Test Cases

### Case 1: Basic Search

**Fixture:**
```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    department: "marketing"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```

**Parameters:**
```yaml
user_id: 123
filters:
  active: true
  departments: ["engineering", "design"]
  name_pattern: null
pagination:
  limit: 20
  offset: 0
sort_by: "name"
include_email: false
table_suffix: "test"
```

**Expected Result:**
```yaml
- id: 1
  name: "John Doe"
  department: "engineering"
  created_at: "2024-01-15T10:30:00Z"
- id: 2
  name: "Jane Smith"
  department: "design"
  created_at: "2024-02-20T14:45:00Z"
```

### Case 2: Full Options with Email

**Fixture:**
```yaml
users:
  - id: 4
    name: "Alice Smith"
    email: "alice@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-10T08:00:00Z"
  - id: 5
    name: "Charlie Smith"
    email: "charlie@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-20T09:00:00Z"
```

**Parameters:**
```yaml
user_id: 456
filters:
  active: true
  departments: ["marketing"]
  name_pattern: "%smith%"
pagination:
  limit: 5
  offset: 0
sort_by: "created_at DESC"
include_email: true
table_suffix: "test"
```

**Expected Result:**
```yaml
- id: 5
  name: "Charlie Smith"
  email: "charlie@example.com"
  department: "marketing"
  created_at: "2024-01-20T09:00:00Z"
- id: 4
  name: "Alice Smith"
  email: "alice@example.com"
  department: "marketing"
  created_at: "2024-01-10T08:00:00Z"
```

## Mock Data
```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
```

## Performance
### Index Requirements
- `users.active` - Required
- `users.department` - Recommended
- `users.name` - For LIKE searches

## Security
### Access Control
- Administrators: Can search all departments
- Regular users: Can only search their own department
