# SnapSQL Template Specification

**Document Version:** 1.0  
**Date:** 2025-06-25  
**Status:** Implementation Complete

## Overview

SnapSQL is a SQL template engine that enables dynamic SQL generation using the **2-way SQL format**. This specification defines the complete template syntax, features, and usage patterns for creating dynamic SQL templates that work as standard SQL during development while providing runtime flexibility.

## Core Principles

### 2-Way SQL Format

SnapSQL templates are designed to work as **valid SQL** when comments are removed, enabling:

- **IDE Support**: Full syntax highlighting and IntelliSense
- **SQL Linting**: Standard SQL tools can validate basic syntax  
- **Development Testing**: Execute templates with dummy values during development
- **Database Compatibility**: Works with PostgreSQL, MySQL, and SQLite

### Comment-Based Directives

All SnapSQL functionality is implemented through SQL comments, ensuring templates remain valid SQL:

- Control flow: `/*# if */`, `/*# for */`, `/*# end */`
- Variable substitution: `/*= variable */`
- Environment references: `/*$ constant */`

## Template Syntax

### 1. Control Flow Directives

#### Conditional Blocks

```sql
/*# if condition */
    -- Content included when condition is true
/*# elseif another_condition */
    -- Content included when another_condition is true
/*# else */
    -- Default content
/*# end */
```

**Key Features:**
- `elseif` (not `else if`) for alternative conditions
- Unified `/*# end */` terminator for all control structures
- Conditions use Google CEL (Common Expression Language)
- Automatic clause removal when conditions are false

#### Loop Blocks

```sql
/*# for variable : list_expression */
    -- Content repeated for each item in list
    /*= variable.field */
/*# end */
```

**Key Features:**
- Iterate over arrays and collections
- Loop variable available within block scope
- Nested loops supported
- Automatic comma handling in SQL lists

### 2. Variable Substitution

#### Basic Variable Substitution

```sql
/*= variable_expression */dummy_value
```

**Examples:**
```sql
SELECT * FROM users WHERE id = /*= user_id */123;
SELECT * FROM users_/*= table_suffix */test;
WHERE department IN (/*= departments */'sales', 'marketing');
```

#### Nested Variable Access

```sql
/*= object.field */
/*= array[0].property */
/*= nested.object.deep.field */
```

### 3. Automatic Runtime Adjustments

SnapSQL automatically handles common SQL construction challenges:

#### Trailing Comma Removal
```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,  -- Trailing comma automatically removed if condition is false
    /*# end */
FROM users;
```

#### Empty Clause Removal

Entire clauses are automatically removed when all conditions are null/empty:

- **WHERE clause**: Removed when all conditions are null/empty
- **ORDER BY clause**: Removed when sort fields are null/empty  
- **LIMIT clause**: Removed when limit is null or negative
- **OFFSET clause**: Removed when offset is null or negative
- **AND/OR conditions**: Individual conditions removed when variables are null/empty

```sql
SELECT * FROM users
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales')
ORDER BY /*= sort_field */name
LIMIT /*= page_size */10
OFFSET /*= offset */0;
```

#### Array Expansion

Arrays automatically expand to comma-separated quoted values:

```sql
-- Template
WHERE department IN (/*= departments */'sales', 'marketing')

-- Runtime with departments = ['engineering', 'design', 'product']
WHERE department IN ('engineering', 'design', 'product')
```

## SQL Statement Support

### SELECT Statements

#### Basic Structure
```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# for field : additional_fields */
        /*= field */,
    /*# end */dummy
FROM users
WHERE active = /*= filters.active */true
    /*# if filters.department */
    AND department = /*= filters.department */'sales'
    /*# end */
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */,
    /*# end */
    created_at DESC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

#### Advanced Features
- **JOINs**: Full support for INNER, LEFT, RIGHT, FULL OUTER joins
- **Subqueries**: Nested SELECT statements with variable substitution
- **Window Functions**: OVER clauses with dynamic partitioning
- **CTEs (Common Table Expressions)**: WITH clauses with recursive support
- **Aggregation**: GROUP BY, HAVING clauses with dynamic grouping

### INSERT Statements

#### Standard INSERT
```sql
INSERT INTO products_/*= env */prod (name, price, category_id)
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### Bulk INSERT (Multiple VALUES)
```sql
INSERT INTO products (name, price, category_id)
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2),
    (/*= product3.name */'Product C', /*= product3.price */150.25, /*= product3.category_id */1);
```

#### Conditional Bulk INSERT
```sql
INSERT INTO orders (user_id, product_id, quantity)
VALUES 
    (/*= order.user_id */1, /*= order.product_id */1, /*= order.quantity */2)
    /*# if include_bulk_orders */
    , (/*= bulk_order1.user_id */2, /*= bulk_order1.product_id */2, /*= bulk_order1.quantity */1)
    , (/*= bulk_order2.user_id */3, /*= bulk_order2.product_id */3, /*= bulk_order2.quantity */5)
    /*# end */;
```

#### Map Array Bulk INSERT (Automatic Expansion)

**Map Array (Bulk):**
```sql
-- When 'products' is []map[string]any, automatically expands to multiple VALUES
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);
```

**Single Map (Regular):**
```sql
-- When 'product' is map[string]any, treated as regular variables
INSERT INTO products (name, price, category_id) 
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### INSERT...SELECT
```sql
INSERT INTO archive_users_/*= env */prod (id, name, email)
SELECT id, name, email 
FROM users_/*= env */prod
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if department_filter */
    AND department = /*= department_filter */'sales'
    /*# end */;
```

### UPDATE Statements

#### Basic UPDATE
```sql
UPDATE products_/*= env */prod
SET 
    name = /*= updates.name */'Updated Product',
    price = /*= updates.price */150.00,
    /*# if updates.category_id */
    category_id = /*= updates.category_id */2,
    /*# end */
    updated_at = NOW()
WHERE id = /*= product_id */123
    /*# if additional_filters */
    AND status = /*= additional_filters.status */'active'
    /*# end */;
```

#### Conditional Field Updates
```sql
UPDATE users_/*= env */prod
SET 
    /*# if updates.name */
    name = /*= updates.name */'New Name',
    /*# end */
    /*# if updates.email */
    email = /*= updates.email */'new@example.com',
    /*# end */
    updated_at = NOW()
WHERE id = /*= user_id */123;
```

#### Bulk UPDATE with Dynamic SET Clauses
```sql
UPDATE products_/*= env */prod
SET 
    /*# for update : field_updates */
    /*= update.field */ = /*= update.value */,
    /*# end */
    updated_at = NOW()
WHERE id IN (/*= product_ids */1, 2, 3);
```

### DELETE Statements

#### Basic DELETE
```sql
DELETE FROM logs_/*= env */prod
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if level_filter */
    AND level = /*= level_filter */'DEBUG'
    /*# end */;
```

#### Conditional DELETE
```sql
DELETE FROM users_/*= env */prod
WHERE 
    /*# if delete_inactive */
    status = 'inactive'
    /*# end */
    /*# if delete_old */
    /*# if delete_inactive */AND /*# end */last_login < /*= cutoff_date */'2023-01-01'
    /*# end */;
```

#### DELETE with Subquery
```sql
DELETE FROM orders_/*= env */prod
WHERE user_id IN (
    SELECT id FROM users_/*= env */prod
    WHERE status = /*= user_status */'deleted'
        /*# if department_filter */
        AND department = /*= department_filter */'sales'
        /*# end */
);
```

## Environment and Table Management

### Table Suffix Pattern

SnapSQL supports environment-specific table naming through suffix patterns:

```sql
-- Template
SELECT * FROM users_/*= env */test;

-- Runtime examples
SELECT * FROM users_dev;     -- env = "dev"
SELECT * FROM users_staging; -- env = "staging"  
SELECT * FROM users_prod;    -- env = "prod"
```

### Environment-Specific Configurations

```sql
-- Different database schemas per environment
SELECT * FROM /*= schema */public.users_/*= env */prod;

-- Environment-specific connection parameters
CONNECT TO /*= database_url */postgresql://localhost/myapp_/*= env */dev;
```

## Advanced Features

### Implicit Conditional Generation

SnapSQL automatically generates conditional blocks for variables that might be null or empty:

```sql
-- Template
WHERE status = /*= filters.status */'active'

-- Automatically becomes
/*# if filters.status != null && filters.status != "" */
WHERE status = /*= filters.status */'active'
/*# end */
```

### Nested Variable References

Support for complex object navigation:

```sql
SELECT 
    /*= user.profile.personal.first_name */'John',
    /*= user.profile.personal.last_name */'Doe',
    /*= user.settings.preferences.theme */'dark'
FROM users;
```

### Dynamic Field Selection

```sql
SELECT 
    id,
    /*# for field : dynamic_fields */
    /*= field */,
    /*# end */
    created_at
FROM /*= table_name */users;
```

### Complex Conditional Logic

```sql
WHERE 1=1
    /*# if filters.status */
    AND status = /*= filters.status */'active'
    /*# end */
    /*# if filters.date_range */
        /*# if filters.date_range.start */
        AND created_at >= /*= filters.date_range.start */'2024-01-01'
        /*# end */
        /*# if filters.date_range.end */
        AND created_at <= /*= filters.date_range.end */'2024-12-31'
        /*# end */
    /*# end */;
```

## Template Formatting Guidelines

### Indentation Rules

1. **Control Block Content**: Indent content inside `/*# if */` and `/*# for */` blocks
2. **Consistent Structure**: Maintain consistent indentation for readability
3. **Line Breaks**: Start control blocks on new lines for visibility

```sql
-- Good formatting
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# for field : additional_fields */
        /*= field */,
    /*# end */
FROM users
WHERE active = /*= filters.active */true
    /*# if filters.department */
    AND department = /*= filters.department */'sales'
    /*# end */;
```

### Comma Placement

1. **Trailing Commas**: Write `field,` not `,field` for easier template authoring
2. **Automatic Cleanup**: SnapSQL removes trailing commas automatically
3. **Conditional Fields**: Don't worry about comma management in conditional blocks

### Dummy Value Guidelines

1. **Always Include**: Provide realistic dummy values for 2-way SQL compatibility
2. **Type Matching**: Ensure dummy values match expected data types
3. **Realistic Data**: Use representative values for better development experience

```sql
-- Good dummy values
WHERE user_id = /*= user_id */123
AND email = /*= email */'user@example.com'
AND created_at > /*= start_date */'2024-01-01'
AND price BETWEEN /*= min_price */10.00 AND /*= max_price */100.00
```

## Security Considerations

### Allowed Operations

SnapSQL restricts modifications to safe operations:

✅ **Permitted:**
- Adding/removing WHERE conditions
- Adding/removing ORDER BY clauses
- Adding/removing SELECT fields
- Table name suffix modification
- Array expansion in IN clauses
- Conditional clause removal
- Bulk INSERT operations
- Dynamic DML operations with SnapSQL variables

❌ **Restricted:**
- Major structural SQL changes
- Dynamic table name changes (except suffixes)
- Arbitrary SQL injection
- Schema modification statements (DDL)

### Parameter Binding

All variable substitutions generate parameterized queries with proper escaping:

```sql
-- Template
WHERE name = /*= user_name */'John'

-- Generated (conceptual)
WHERE name = ?  -- with parameter: "John"
```

## Error Handling

### Template Validation

SnapSQL validates templates at build time:

- **Syntax Errors**: Invalid SQL structure detection
- **Directive Errors**: Malformed control flow directives
- **Variable Errors**: Undefined variable references
- **Clause Violations**: Control blocks spanning multiple SQL clauses

### Runtime Validation

- **Type Checking**: Variable type validation against schema
- **Null Handling**: Automatic null/empty value handling
- **Parameter Validation**: Required parameter checking

## File Organization

### Template Files

#### `.snap.sql` Files
Pure SQL templates with SnapSQL directives:

```sql
-- queries/users.snap.sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

#### `.snap.md` Files (Literate Programming)
Combine templates with documentation and test cases:

````markdown
# User Query Template

## Purpose
Retrieve user data with optional email inclusion and environment-specific tables.

## Parameters
- `include_email`: boolean - Include email field in results
- `env`: string - Environment suffix for table name
- `filters.active`: boolean - Filter by active status

## SQL Template

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

## Test Cases

### Basic Query
**Input:**
```json
{
    "include_email": false,
    "env": "prod",
    "filters": {"active": true}
}
```

**Expected Output:**
```sql
SELECT id, name FROM users_prod WHERE active = true;
```
````

## Best Practices

### Template Design

1. **Start Simple**: Begin with basic templates and add complexity gradually
2. **Use Meaningful Names**: Choose descriptive variable and parameter names
3. **Document Intent**: Include comments explaining complex logic
4. **Test Thoroughly**: Provide comprehensive test cases

### Performance Considerations

1. **Index Awareness**: Consider database indexes when designing dynamic WHERE clauses
2. **Query Planning**: Test generated queries with EXPLAIN/ANALYZE
3. **Parameter Limits**: Be mindful of database parameter limits in bulk operations

### Maintainability

1. **Consistent Patterns**: Use consistent naming and structure across templates
2. **Modular Design**: Break complex templates into smaller, reusable components
3. **Version Control**: Track template changes with meaningful commit messages
4. **Code Reviews**: Review templates like application code

## Migration and Compatibility

### Database Compatibility

SnapSQL templates work across database systems with minor dialect considerations:

- **PostgreSQL**: Full feature support
- **MySQL**: Full feature support with dialect-specific syntax
- **SQLite**: Full feature support with limitations on advanced features

### Version Compatibility

- **Backward Compatibility**: New SnapSQL versions maintain template compatibility
- **Deprecation Policy**: Features deprecated with advance notice and migration paths
- **Upgrade Guidance**: Clear upgrade instructions for breaking changes

## Conclusion

SnapSQL provides a powerful, safe, and maintainable approach to dynamic SQL generation. By following the 2-way SQL format and leveraging comment-based directives, developers can create flexible SQL templates that work seamlessly in development and production environments while maintaining the benefits of standard SQL tooling and practices.

The template specification ensures that SnapSQL templates are:

- **Developer Friendly**: Work with existing SQL tools and workflows
- **Runtime Flexible**: Adapt to changing requirements and conditions
- **Security Focused**: Prevent SQL injection through controlled modifications
- **Performance Aware**: Generate efficient, parameterized queries
- **Maintainable**: Support long-term application evolution

This specification serves as the definitive guide for creating and maintaining SnapSQL templates across all supported runtime environments.
