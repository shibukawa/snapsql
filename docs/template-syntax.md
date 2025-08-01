# Template Syntax

SnapSQL uses a 2-way SQL format that allows templates to work as standard SQL while providing runtime flexibility.

## Basic Concepts

### 2-way SQL Format

The core principle is that your SQL templates should be valid SQL that can be executed directly during development:

```sql
-- This works as standard SQL
SELECT id, name, email
FROM users_dev
WHERE active = true
ORDER BY created_at DESC
LIMIT 10;
```

The same template with SnapSQL syntax:

```sql
-- This also works as standard SQL (comments are ignored)
SELECT 
    id, 
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
ORDER BY created_at DESC
LIMIT /*= pagination.limit */10;
```

## Template Directives

### Variable Substitution

Use `/*= expression */dummy_value` for variable substitution:

```sql
-- Basic substitution
FROM users_/*= table_suffix */dev

-- With parameters
WHERE active = /*= filters.active */true
LIMIT /*= pagination.limit */10
```

### Conditional Blocks

Use `/*# if condition */` and `/*# end */` for conditional content:

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# if include_profile */
    profile_data,
    /*# end */
    created_at
FROM users
```

### Loops (Planned)

```sql
/*# for field in selected_fields */
    /*= field */,
/*# end */
```

## Parameter Types

### Simple Parameters

```json
{
  "table_suffix": "prod",
  "limit": 50,
  "active": true
}
```

### Nested Parameters

```json
{
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

### Array Parameters

```json
{
  "departments": ["engineering", "design", "product"],
  "user_ids": [1, 2, 3, 4, 5]
}
```

## Template Metadata

Each template can include metadata in a comment block at the top:

```sql
/*#
name: getUserList
description: Get list of users with optional filtering
function_name: getUserList
parameters:
  include_email: bool
  table_suffix: string
  filters:
    active: bool
    departments:
      - string
  pagination:
    limit: int
    offset: int
*/

SELECT id, name FROM users;
```

## Expression Language

SnapSQL uses a simple expression language for parameter references:

- `table_suffix` - Simple parameter
- `filters.active` - Nested parameter
- `pagination.limit` - Nested parameter
- `departments` - Array parameter

## Security Features

SnapSQL provides controlled modifications to prevent SQL injection:

- **Allowed**: Field selection, simple conditions, table suffix changes
- **Prevented**: Arbitrary SQL injection, complex WHERE clause modifications
- **Validated**: All parameters are type-checked and validated

## Best Practices

1. **Always provide defaults**: Ensure templates work as standard SQL
2. **Use meaningful parameter names**: Make templates self-documenting
3. **Keep conditions simple**: Complex logic should be in application code
4. **Test with dry-run**: Use `--dry-run` to verify template processing
5. **Document parameters**: Include metadata for better maintainability

## Examples

### User Query with Filtering

```sql
/*#
name: searchUsers
description: Search users with various filters
parameters:
  search_term: string
  department: string
  active_only: bool
  limit: int
*/

SELECT 
    u.id,
    u.name,
    u.email,
    u.department
FROM users u
WHERE 1=1
    /*# if search_term */
    AND (u.name ILIKE /*= search_term */'%john%' OR u.email ILIKE /*= search_term */'%john%')
    /*# end */
    /*# if department */
    AND u.department = /*= department */'engineering'
    /*# end */
    /*# if active_only */
    AND u.active = true
    /*# end */
ORDER BY u.created_at DESC
LIMIT /*= limit */50;
```

### Dynamic Table Selection

```sql
/*#
name: getTableData
description: Get data from different table variants
parameters:
  environment: string
  include_archived: bool
*/

SELECT *
FROM data_/*= environment */prod
/*# if include_archived */
UNION ALL
SELECT * FROM data_archive_/*= environment */prod
/*# end */
ORDER BY created_at DESC;
```
