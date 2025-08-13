# Directives Reference

SnapSQL directives are embedded in SQL comments and provide dynamic query generation capabilities while maintaining valid SQL syntax.

## Variable Substitution

### Basic Syntax
```sql
/*= expression */dummy_value
```

The expression is evaluated at runtime, and `dummy_value` is used when the SQL is executed directly during development.

### Examples

```sql
-- Simple variable substitution
WHERE user_id = /*= user_id */1

-- String concatenation
FROM users_/*= table_suffix */'dev'

-- Complex expressions with CEL
WHERE created_at >= /*= start_date != null ? start_date : "2024-01-01" */'2024-01-01'

-- Array expansion in IN clause
WHERE status IN /*= status_list */('active', 'pending')

-- Object field access
WHERE department_id = /*= user.department_id */1
```

## Conditional Blocks

### If Directive
```sql
/*# if condition */
SQL content
/*# end */
```

### If-Else Directive
```sql
/*# if condition */
SQL for true condition
/*# else */
SQL for false condition
/*# end */
```

### Examples

```sql
-- Simple boolean condition
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    created_at
FROM users;

-- Complex condition with CEL expression
/*# if user_role == "admin" || user_role == "moderator" */
SELECT * FROM sensitive_data
/*# else */
SELECT id, name FROM public_data
/*# end */

-- Null checking
/*# if filters.department_id != null */
WHERE department_id = /*= filters.department_id */1
/*# end */

-- String comparison
/*# if sort_order == "desc" */
ORDER BY created_at DESC
/*# else */
ORDER BY created_at ASC
/*# end */
```

## Loop Directives

### For Loop Syntax
```sql
/*# for item : collection */
SQL content using item
/*# end */
```

### Examples

```sql
-- Simple array iteration
INSERT INTO tags (name) VALUES
/*# for tag : tag_list */
(/*= tag */'example')
/*# end */;

-- Object array iteration
INSERT INTO users (id, name, email) VALUES
/*# for user : users */
(/*= user.id */1, /*= user.name */'John', /*= user.email */'john@example.com')
/*# end */;

-- Nested loops
INSERT INTO department_users (dept_code, user_id) VALUES
/*# for dept : departments */
    /*# for user : dept.users */
    (/*= dept.code */'ENG', /*= user.id */1)
    /*# end */
/*# end */;

-- Loop with conditions
/*# for item : items */
    /*# if item.active */
    INSERT INTO active_items (id, name) VALUES (/*= item.id */1, /*= item.name */'Item');
    /*# end */
/*# end */
```

## CEL Expressions

SnapSQL uses [Common Expression Language (CEL)](https://github.com/google/cel-spec) for complex expressions.

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition/Concatenation | `"Hello " + name` |
| `-` | Subtraction | `end_date - start_date` |
| `*` | Multiplication | `price * quantity` |
| `/` | Division | `total / count` |
| `%` | Modulo | `id % 10` |
| `==` | Equality | `status == "active"` |
| `!=` | Inequality | `role != "guest"` |
| `<`, `<=`, `>`, `>=` | Comparison | `age >= 18` |
| `&&` | Logical AND | `active && verified` |
| `\|\|` | Logical OR | `role == "admin" \|\| role == "mod"` |
| `!` | Logical NOT | `!deleted` |
| `in` | Membership | `status in ["active", "pending"]` |

### Built-in Functions

```sql
-- String functions
WHERE name.startsWith(/*= prefix */'John')
WHERE email.endsWith(/*= domain */'@example.com')
WHERE description.contains(/*= keyword */'important')

-- Size/length functions
WHERE size(tags) > /*= min_tags */0
WHERE len(description) <= /*= max_length */255

-- Type checking
/*# if type(user_id) == int */
WHERE id = /*= user_id */1
/*# end */

-- Null checking
/*# if user.email != null */
WHERE email = /*= user.email */'user@example.com'
/*# end */
```

### Complex Expressions

```sql
-- Conditional expressions (ternary operator)
LIMIT /*= page_size > 0 ? page_size : 10 */10

-- Multiple conditions
/*# if user.role == "admin" && user.active && user.verified */
SELECT * FROM admin_panel
/*# end */

-- Array operations
WHERE id IN /*= user_ids.filter(id, id > 0) */[1, 2, 3]

-- Object field access with defaults
WHERE department_id = /*= user.department?.id ?? 1 */1
```

## System Fields

SnapSQL automatically handles common system fields like timestamps and versioning.

### Automatic Field Injection

```sql
-- INSERT automatically adds system fields
INSERT INTO users (name, email) 
VALUES (/*= name */'John', /*= email */'john@example.com');
-- Automatically becomes:
-- INSERT INTO users (name, email, created_at, updated_at, version) 
-- VALUES ('John', 'john@example.com', NOW(), NOW(), 1);

-- UPDATE automatically updates system fields
UPDATE users 
SET name = /*= name */'John'
WHERE id = /*= user_id */1;
-- Automatically becomes:
-- UPDATE users 
-- SET name = 'John', updated_at = NOW(), version = version + 1
-- WHERE id = 1;
```

### Explicit System Field Control

```sql
-- Explicit system field values
INSERT INTO users (name, email, created_at, version) 
VALUES (/*= name */'John', /*= email */'john@example.com', /*= created_at */'2024-01-01', /*= version */1);

-- Disable automatic system fields
/*# system_fields: false */
INSERT INTO users (name, email) 
VALUES (/*= name */'John', /*= email */'john@example.com');
```

## Advanced Directives

### Response Affinity Hints

```sql
/*# response_affinity: one */
SELECT * FROM users WHERE id = /*= user_id */1;

/*# response_affinity: many */
SELECT * FROM users WHERE active = true;

/*# response_affinity: none */
UPDATE users SET last_login = NOW() WHERE id = /*= user_id */1;
```

### Custom Function Names

```sql
/*# function_name: findUserById */
SELECT * FROM users WHERE id = /*= user_id */1;
```

### Table and Column Aliases

```sql
-- Structured field names with double underscores
SELECT 
    u.id,
    u.name,
    d.id as department__id,
    d.name as department__name,
    p.bio as profile__bio
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN profiles p ON u.id = p.user_id;
```

## Error Handling

### Common Errors and Solutions

```sql
-- ❌ Unterminated block comment
/*# if condition
SELECT * FROM users;
/*# end */

-- ✅ Properly terminated
/*# if condition */
SELECT * FROM users;
/*# end */

-- ❌ Invalid CEL expression
WHERE id = /*= user_id + */1

-- ✅ Valid CEL expression
WHERE id = /*= user_id + 1 */1

-- ❌ Missing dummy value
WHERE status = /*= status */

-- ✅ With dummy value
WHERE status = /*= status */'active'
```

### Validation

Use the CLI to validate templates:

```bash
# Validate single file
snapsql validate template.snap.sql

# Validate directory
snapsql validate ./queries

# Validate with verbose output
snapsql validate ./queries --verbose
```

## Best Practices

### 1. Always Provide Dummy Values
```sql
-- ✅ Good: SQL works during development
WHERE user_id = /*= user_id */1

-- ❌ Bad: SQL breaks during development
WHERE user_id = /*= user_id */
```

### 2. Use Meaningful Dummy Values
```sql
-- ✅ Good: Realistic dummy data
WHERE email LIKE /*= email_pattern */'%@example.com'

-- ❌ Bad: Meaningless dummy data
WHERE email LIKE /*= email_pattern */'xxx'
```

### 3. Keep Conditions Simple
```sql
-- ✅ Good: Simple, readable condition
/*# if include_deleted */
WHERE deleted_at IS NOT NULL
/*# end */

-- ❌ Bad: Complex condition in directive
/*# if user.role == "admin" && user.permissions.includes("delete") && user.active */
```

### 4. Use Consistent Naming
```sql
-- ✅ Good: Consistent snake_case
/*# for user_item : user_items */

-- ❌ Bad: Mixed naming conventions
/*# for userItem : user_items */
```
