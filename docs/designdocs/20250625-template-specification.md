# SnapSQL Template Specification

**Document Version:** 1.1  
**Date:** 2025-06-25  
**Status:** Implemented

## Overview

SnapSQL is a SQL template engine that adopts the 2-way SQL format. This document defines the implemented template syntax and features.

## 2-way SQL Format

SnapSQL templates are designed to function as standard SQL when comments are removed:

- **IDE Support**: SQL syntax checking and code completion enabled
- **SQL Linter Compatible**: Validation with standard SQL tools
- **Development Execution**: Possible to execute during development using dummy values
- **Database Compatibility**: Supports PostgreSQL, MySQL, and SQLite

## Template Syntax

### 1. Control Structures

#### Conditional Branching

```sql
/*# if condition */
    -- Content included when condition is true
/*# elseif another_condition */
    -- Content included when another_condition is true
/*# else */
    -- Default content
/*# end */
```

**Features:**
- Uses `elseif` (not `else if`)
- All control structures end with `/*# end */`
- Uses Google CEL for conditions
- Automatically removes entire clause when condition is false

#### Loops

```sql
/*# for variable : list_expression */
    -- Content repeated for each element in the list
    /*= variable.field */
/*# end */
```

**Features:**
- Iteration over arrays and collections
- Loop variable usage within block scope
- Support for nested loops
- Automatic comma handling in SQL lists

### 2. Variable Substitution

#### Basic Variable Substitution

```sql
/*= variable_expression */[dummy_value]
```

Variable substitution supports two formats:

1. **Explicit Dummy Values (Recommended)**
```sql
SELECT * FROM users WHERE id = /*= user_id */123;
SELECT * FROM users_/*= table_suffix */test;
```

2. **Automatic Dummy Values with Type Inference**
```sql
SELECT * FROM users WHERE id = /*= user_id */;
SELECT * FROM users WHERE active = /*= is_active */;
```

#### Benefits of Using Dummy Values

1. **SQL Development Tool Compatibility**
   - Syntax highlighting in SQL editors
   - Code completion enablement
   - SQL formatter proper operation
   - Query plan verification with query planners

2. **Direct Execution During Development**
   - Functions as valid SQL when comments are removed
   - Query testing in development environment
   - Execution plan verification
   - Index usage confirmation

#### Type Conversion

1. **Standard SQL CAST**
```sql
-- Explicit type conversion
CAST(/*= value */123 AS INTEGER)
CAST(/*= date_str */'2024-01-01' AS DATE)

-- Implicit type conversion (compatible types)
WHERE created_at > /*= start_date */'2024-01-01'
```

2. **PostgreSQL-Specific Type Conversion**
```sql
-- PostgreSQL style CAST
/*= value */123::INTEGER
/*= timestamp */'2024-01-01 12:34:56'::TIMESTAMP

-- Automatic conversion
-- PostgreSQL format to standard SQL
WHERE created_at > CAST(/*= start_date */'2024-01-01' AS DATE)
-- Standard SQL to PostgreSQL format
WHERE created_at > /*= start_date */'2024-01-01'::DATE
```

3. **Array and JSON Types**
```sql
-- PostgreSQL arrays
WHERE tags = /*= tag_array */ARRAY['tag1', 'tag2']
WHERE tags @> /*= search_tags */ARRAY['tag1']

-- JSON/JSONB
WHERE data @> /*= json_filter */'{"status": "active"}'::JSONB
```

#### Type Inference Default Values
- Numeric types (int, float): `0`
- String types: `''`
- Boolean types: `false`
- Array types: `[]`
- Object types: `{}`
- DateTime types: `CURRENT_TIMESTAMP`

**Examples:**
```sql
-- Explicit dummy values (recommended)
WHERE department IN (/*= departments */'sales', 'marketing')
  AND created_at > /*= start_date */'2024-01-01'
  AND points > /*= min_points */100

-- Type-inferred dummy values
WHERE status = /*= status */
  AND created_at > /*= start_date */
  AND is_active = /*= is_active */
  AND points > /*= min_points */
```

### 3. Automatic Adjustments

#### Automatic Comma Removal
```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,  -- Comma automatically removed when condition is false
    /*# end */
FROM users;
```

#### Empty Clause Removal

Entire clauses are automatically removed when:

- **WHERE clause**: All conditions are null or empty
- **ORDER BY clause**: Sort fields are null or empty
- **LIMIT clause**: Limit is null or negative
- **OFFSET clause**: Offset is null or negative
- **AND/OR conditions**: Variables are null or empty

```sql
SELECT * FROM users
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales')
ORDER BY /*= sort_field */name
LIMIT /*= page_size */10
OFFSET /*= offset */0;
```

#### Array Expansion

Arrays are automatically expanded into comma-separated quoted values:

```sql
-- Template
WHERE department IN (/*= departments */'sales', 'marketing')

-- Runtime (departments = ['engineering', 'design', 'product'])
WHERE department IN ('engineering', 'design', 'product')
```

## Supported SQL Statements

### SELECT Statement

```sql
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
    /*# end */
ORDER BY created_at DESC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

### INSERT Statement

#### Single Row INSERT
```sql
INSERT INTO products (name, price, category_id)
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### Multiple Row INSERT
```sql
INSERT INTO products (name, price, category_id)
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2);
```

#### Array-Based Multiple Row INSERT
```sql
-- When products is []map[string]any, automatically expands to multiple VALUES
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);
```

### UPDATE Statement

```sql
UPDATE products
SET 
    name = /*= updates.name */'Updated Product',
    price = /*= updates.price */150.00,
    /*# if updates.category_id */
    category_id = /*= updates.category_id */2,
    /*# end */
    updated_at = NOW()
WHERE id = /*= product_id */123;
```

### DELETE Statement

```sql
DELETE FROM logs
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if level_filter */
    AND level = /*= level_filter */'DEBUG'
    /*# end */;
```

## Table Name Suffixes

Control environment-specific table names with suffixes:

```sql
-- Template
SELECT * FROM users_/*= env */test;

-- Runtime
SELECT * FROM users_dev;     -- env = "dev"
SELECT * FROM users_staging; -- env = "staging"  
SELECT * FROM users_prod;    -- env = "prod"
```

## Security Restrictions

### ✅ Allowed Operations
- Adding/removing WHERE conditions
- Adding/removing ORDER BY clauses
- Adding/removing SELECT fields
- Table name suffix modification (e.g., `users_test`, `log_202412`)
- Array expansion in IN clauses
- Conditional clause removal
- Multiple row INSERT operations
- DML operations using SnapSQL variables

### ❌ Restricted Operations
- Structural SQL changes
- Dynamic table name changes (except suffixes)
- Arbitrary SQL injection
- Schema modification statements (DDL)

## Error Handling

### Template Validation
- Syntax error detection
- Directive format checking
- Variable reference validation
- Control block scope checking

### Runtime Validation
- Type checking
- NULL value handling
- Parameter validation

## File Formats

### `.snap.sql` Files
Files containing only SQL templates:

```sql
-- queries/users.snap.sql
SELECT id, name
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

### `.snap.md` Files
Files containing documentation and test cases:

````markdown
# User Query Template

## SQL Template
```sql
SELECT id, name
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

## Test Cases
```json
{
    "env": "prod",
    "filters": {"active": true}
}
```
````

## Best Practices

1. **Start Simple**: Begin with basic templates and gradually add complexity
2. **Document Intent**: Add comments for complex logic
3. **Create Test Cases**: Provide comprehensive test cases
4. **Consider Indexes**: Consider index usage in dynamic WHERE clauses
5. **Consistent Patterns**: Use consistent naming conventions and structure across templates
