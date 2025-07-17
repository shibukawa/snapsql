# Type Inference System Design

## Overview

The type inference system in SnapSQL determines the types of variables and expressions in SQL templates. It ensures type safety during template compilation and runtime execution while maintaining compatibility with different database systems.

## Design Goals

1. **Type Safety**
   - Prevent type-related runtime errors
   - Early detection of type mismatches
   - Consistent type handling across databases

2. **Developer Experience**
   - Clear error messages for type mismatches
   - Intelligent type inference with minimal annotations
   - IDE support through type information

3. **Database Compatibility**
   - Support for PostgreSQL, MySQL, and SQLite types
   - Consistent type mapping across databases
   - Handling of database-specific type features

## Type System

### Basic Types

```go
type BasicType string

const (
    TypeString   BasicType = "string"
    TypeInt      BasicType = "int"
    TypeFloat    BasicType = "float"
    TypeBool     BasicType = "bool"
    TypeDate     BasicType = "date"
    TypeTime     BasicType = "time"
    TypeDateTime BasicType = "datetime"
    TypeJSON     BasicType = "json"
    TypeBinary   BasicType = "binary"
)
```

### Complex Types

```go
type ComplexType struct {
    Kind       TypeKind
    ElementType *Type        // For arrays
    Fields     map[string]*Type  // For objects
    Nullable   bool
}

type TypeKind int

const (
    KindBasic TypeKind = iota
    KindArray
    KindObject
    KindEnum
)
```

### Type Context

```go
type TypeContext struct {
    Variables map[string]*Type
    Functions map[string]*FunctionType
    Tables    map[string]*TableType
}

type FunctionType struct {
    Parameters []*Type
    ReturnType *Type
}

type TableType struct {
    Columns map[string]*ColumnType
}
```

## Type Inference Process

### 1. Schema Analysis
- Extract type information from database schema
- Build initial type context
- Handle database-specific types

### 2. Template Analysis
- Parse template variables and expressions
- Analyze SQL structure for type context
- Identify type constraints from SQL operations

### 3. Type Resolution
- Resolve variable types
- Infer expression types
- Handle type conversions
- Validate type constraints

### 4. Type Validation
- Check type compatibility
- Validate SQL operations
- Ensure type safety
- Generate type errors

## Type Inference Rules

### 1. Variable Types

```sql
-- String type inference
WHERE name = /*= user_name */'John'
WHERE email LIKE /*= email_pattern */'%@example.com'

-- Numeric type inference
WHERE age > /*= min_age */18
WHERE price BETWEEN /*= min_price */10.0 AND /*= max_price */20.0

-- Boolean type inference
WHERE is_active = /*= active */true

-- Date/Time type inference
WHERE created_at > /*= start_date */'2024-01-01'
```

### 2. Expression Types

```sql
-- Arithmetic expressions
WHERE price * /*= quantity */5 > /*= min_total */100

-- String concatenation
WHERE name LIKE /*= prefix */'A%' || /*= suffix */'son'

-- Boolean expressions
WHERE active = /*= is_active */true AND age > /*= min_age */18
```

### 3. Array Types

```sql
-- Array in IN clause
WHERE department IN (/*= departments */'sales', 'marketing')

-- Array functions
WHERE tags @> /*= search_tags */ARRAY['tag1', 'tag2']
```

### 4. Object Types

```sql
-- Object field access
WHERE metadata->>'status' = /*= filters.status */'active'
  AND metadata->>'type' = /*= filters.type */'user'
```

## Type Conversion Rules

### 1. Implicit Conversions

```sql
-- String to number
WHERE id = /*= user_id */'123'  -- Converted to number if compatible

-- Number to string
WHERE code = /*= product_code */123  -- Converted to string if required

-- Date/time conversions
WHERE created_at > /*= date_str */'2024-01-01'  -- Converted to date
```

### 2. Explicit Conversions

```sql
-- Using CAST
WHERE created_at > CAST(/*= timestamp */'2024-01-01' AS DATE)

-- Using database-specific syntax
WHERE created_at > /*= timestamp */'2024-01-01'::DATE  -- PostgreSQL
```

## Error Handling

### 1. Type Mismatch Errors

```go
type TypeError struct {
    Position    Position
    Expected    *Type
    Actual      *Type
    Context     string
}
```

Example error messages:
```
Error at line 5, column 10:
Expected type: number
Actual type: string
In expression: user_id = 'abc'
```

### 2. Validation Errors

```go
type ValidationError struct {
    Position    Position
    Message     string
    Details     map[string]interface{}
}
```

Example error messages:
```
Error at line 7, column 15:
Invalid operation: cannot compare date with string
Details: created_at > 'invalid-date'
```

## Implementation Details

### 1. Type Inference Engine

```go
type InferenceEngine struct {
    typeContext *TypeContext
    schema      *SchemaInfo
}

func (e *InferenceEngine) InferTypes(template *Template) (*TypeInfo, error)
func (e *InferenceEngine) ValidateTypes(typeInfo *TypeInfo) error
```

### 2. Type Resolution

```go
type TypeResolver struct {
    context *TypeContext
}

func (r *TypeResolver) ResolveType(expr Expr) (*Type, error)
func (r *TypeResolver) UnifyTypes(types []*Type) (*Type, error)
```

### 3. Type Validation

```go
type TypeValidator struct {
    context *TypeContext
}

func (v *TypeValidator) ValidateExpr(expr Expr, expectedType *Type) error
func (v *TypeValidator) ValidateOperation(op Operation, args []*Type) error
```

## Database-Specific Handling

### 1. PostgreSQL Types

```sql
-- Array types
WHERE tags = /*= tag_array */ARRAY['tag1', 'tag2']

-- JSONB operations
WHERE data @> /*= json_filter */'{"status": "active"}'::JSONB

-- Custom types
WHERE status = /*= status */'active'::user_status
```

### 2. MySQL Types

```sql
-- ENUM types
WHERE status = /*= status */'active'

-- SET types
WHERE permissions = /*= permissions */'read,write'

-- JSON operations
WHERE JSON_CONTAINS(data, /*= json_filter */'{"status": "active"}')
```

### 3. SQLite Types

```sql
-- Dynamic typing
WHERE id = /*= id */123  -- Handles both string and number

-- JSON operations (if JSON1 extension is enabled)
WHERE json_extract(data, '$.status') = /*= status */'active'
```

## Best Practices

1. **Explicit Type Annotations**
   - Use dummy values that match expected types
   - Add comments for complex type requirements
   - Document custom type conversions

2. **Type Safety**
   - Validate types early in development
   - Use explicit conversions for clarity
   - Handle NULL values consistently

3. **Database Compatibility**
   - Use standard SQL types when possible
   - Document database-specific type features
   - Test type handling across databases

4. **Error Handling**
   - Provide clear error messages
   - Include context in type errors
   - Guide developers to correct type usage
