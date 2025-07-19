# FunctionDefinition Type System Specification

## Overview

This document defines the parameter type system specification for SnapSQL's FunctionDefinition. Parameter types can be flexibly described in YAML, supporting type aliases, arrays, nesting, type inference, dummy value generation, error handling, and more.

## Supported Types and Aliases

### Primitive Types & Aliases

| Type Name | Aliases                      | Go Type      | CEL Type      | Description      |
|-----------|-----------------------------|--------------|---------------|------------------|
| `string`  | `text`, `varchar`, `str`    | string       | string        | String type      |
| `int`     | `integer`, `long`           | int64        | int           | 64-bit integer   |
| `int32`   |                             | int32        | int           | 32-bit integer   |
| `int16`   | `smallint`                  | int16        | int           | 16-bit integer   |
| `int8`    | `tinyint`                   | int8         | int           | 8-bit integer    |
| `float`   | `double`                    | float64      | double        | 64-bit float     |
| `decimal` | `numeric`                   | github.com/shopspring/decimal.Decimal | double | High-precision decimal |
| `float32` |                             | float32      | double        | 32-bit float     |
| `bool`    | `boolean`                   | bool         | bool          | Boolean type     |

### Special Types

| Type Name   | Go Type           | CEL Type         | Description                |
|-------------|-------------------|------------------|----------------------------|
| `date`      | time.Time         | string           | Date (YYYY-MM-DD)          |
| `datetime`  | time.Time         | string           | Datetime                   |
| `timestamp` | time.Time         | string           | Timestamp                  |
| `email`     | string            | string           | Email address              |
| `uuid`      | github.com/google/uuid.UUID | string   | UUID type                  |
| `json`      | map[string]any    | map(string, dyn) | JSON type                  |
| `any`       | any               | dyn              | Any type (inference/unknown) |

### Array Types

- Array types are expressed by appending `[]` to the type name, e.g., `int[]`, `string[]`, `float32[]`.
- YAML list notation `[int]` or array objects like `- id: int\n  name: string` are also supported.

## Type Definition Syntax

### 1. Simple Type Specification

```yaml
parameters:
  id: int
  name: string
  scores: float[]
  flags: [bool]
  prices: decimal[]
```

### 2. Object/Nesting

```yaml
parameters:
  user:
    id: int
    name: string
    profile:
      email: email
      age: int
```

### 3. Array Syntax

- Primitive array: `[int]` → `int[]`
- Object array:

```yaml
parameters:
  users:
    - id: int
      name: string
  tags: [string]
```

### 4. Type Inference

If the type is not explicitly specified, it is inferred from the value:

```yaml
parameters:
  user_id: 123          # → int
  user_name: "John"     # → string
  is_active: true       # → bool
  price: 99.99          # → float
  items:
    - id: 1
      value: 2.0
```

## Type Aliases & Normalization

- `integer`, `long` → `int`
- `double` → `float`
- `decimal`, `numeric` → `decimal`
- `text`, `varchar`, `str` → `string`
- `boolean` → `bool`

## Type Resolution Algorithm

1. **Variable Name Validation**: Must start with alphanumeric or underscore
2. **Type Name Normalization**: Aliases are normalized, array types unified with `[]`
3. **Recursive Array/Object Processing**: Nesting and arrays are resolved recursively
4. **Type Inference**: Type is inferred from value
5. **Fallback to Default Type (string) on Error**

## Dummy Literal Generation

- Dummy values for each type are generated automatically by the implementation (details omitted).

## Error Handling

- Invalid variable names are skipped and errors are returned
- Unknown/undefined types become `any` type and dummy value is `nil`
- Cyclic references in nesting/arrays are detected and result in error

## Usage Example

```yaml
parameters:
  user_id: int
  user_name: string
  is_active: bool
  users:
    - id: int32
      name: string
  tags: [string]
  meta:
    created_at: datetime
    updated_at: datetime
```

## Example Usage in SQL Template

```sql
SELECT /*= user_id */, /*= user_name */ FROM users WHERE is_active = /*= is_active */
```

## Future Extensions

- Custom types (enum, etc.) and constraints (min/max) will be supported in the future
    name:
      type: string
      description: "User name"
    profile:
      email:
        type: email
        description: "Email address"
      age:
        type: int
        description: "Age"
```

### Type Inference

If the type is not explicitly specified, it is inferred from the value:

```yaml
parameters:
  # Example of type inference
  user_id: 123          # → int
  user_name: "John"     # → string
  is_active: true       # → bool
  price: 99.99          # → float
```

## Type Resolution by Variable Reference

### Dot Notation

Nested objects can be accessed using dot notation:

```sql
SELECT /*= user.name */ FROM users WHERE id = /*= user.profile.user_id */
```

### Type Resolution Algorithm

1. **Split Variable Name**: `user.profile.name` → `["user", "profile", "name"]`
2. **Hierarchical Type Resolution**: Get type info at each level
3. **Determine Final Type**: Return the type of the last element

```go
// Example of type resolution
variableName := "user.profile.email"
// user → map[string]any
// user.profile → map[string]any  
// user.profile.email → type: "email" → dummy value: "user@example.com"
```

## Dummy Literal Generation Rules

### Conversion from Type to Dummy Value

```go
func generateDummyValueFromType(paramType string) string {
    switch strings.ToLower(paramType) {
    case "int", "integer", "long":
        return "1"
    case "float", "double", "decimal", "number":
        return "1.0"
    case "bool", "boolean":
        return "true"
    case "string", "text":
        return "'dummy'"
    case "date":
        return "'2024-01-01'"
    case "datetime", "timestamp":
        return "'2024-01-01 00:00:00'"
    case "email":
        return "'user@example.com'"
    case "uuid":
        return "'00000000-0000-0000-0000-000000000000'"
    case "json":
        return "'{}'"
    case "array":
        return "'[]'"
    default:
        return "'dummy'"  // Default is string
    }
}
```

### Token Type Mapping

Mapping from dummy value to TokenType:

```go
func inferTokenTypeFromValue(dummyValue string) tokenizer.TokenType {
    switch {
    case strings.HasPrefix(dummyValue, "'"):
        return tokenizer.STRING
    case dummyValue == "true" || dummyValue == "false":
        return tokenizer.BOOLEAN
    case strings.Contains(dummyValue, "."):
        return tokenizer.NUMBER // Floating point
    default:
        return tokenizer.NUMBER // Integer
    }
}
```

## Error Handling

### Type Resolution Errors

1. **Parameter Not Found**: `ErrParameterNotFound`
2. **Not a Nested Object**: `ErrParameterNotNestedObject`
3. **Invalid Type Info**: Fallback to default type (string)

### Error Handling Policy

```go
// Use default value on error
paramType, err := getParameterType(variableName, functionDef.Parameters)
if err != nil {
    paramType = "string"  // Default type
}
```

## Usage Example

### Complete FunctionDefinition Example

```yaml
# snapsql.yaml
functions:
  getUserProfile:
    description: "Get user profile"
    parameters:
      user_id:
        type: int
        description: "Target user ID"
      include_profile:
        type: bool
        description: "Include profile info"
      filters:
        status:
          type: string
          description: "Status"
        created_after:
          type: date
          description: "Created after filter"
```

### Usage in SQL Template

```sql
-- getUserProfile.snap.sql
SELECT 
    id,
    name,
    /*= filters.status */ as status,
    created_at
FROM users 
WHERE 
    id = /*= user_id */
    /*@ if include_profile */
    AND profile_id IS NOT NULL
    /*@ end */
    /*@ if filters.created_after */
    AND created_at >= /*= filters.created_after */
    /*@ end */
```

## Implementation Stages

### parserstep1 (Implemented)

- Insert DUMMY_LITERAL tokens
- Extract and store variable names
- Basic syntax validation

### parserstep6 (Planned)

- Type resolution using FunctionDefinition
- Replace with actual literal values
- CEL expression evaluation
- Namespace management

## Future Extensions

### Custom Types

```yaml
parameters:
  user_status:
    type: enum
    values: ["active", "inactive", "pending"]
    default: "active"
```

### Detailed Array Specification

```yaml
parameters:
  user_ids:
    type: array
    element_type: int
    description: "Array of user IDs"
```

### Constraints

```yaml
parameters:
  age:
    type: int
    min: 0
    max: 150
    description: "Age (0-150)"
```

This type system improves type safety and development efficiency in SnapSQL templates.# FunctionDefinition Type System Specification

## Overview

This document defines the parameter type system specification for SnapSQL's FunctionDefinition. This type system is used for dummy literal generation in `/*= variable */` directives and actual literal replacement in parserstep6.

## Basic Types

### Primitive Types

| Type Name | Aliases | Dummy Value | Description |
|-----------|---------|-------------|-------------|
| `string` | `text` | `'dummy'` | String type |
| `int` | `integer`, `long` | `1` | Integer type |
| `float` | `double`, `decimal`, `number` | `1.0` | Floating-point number type |
| `bool` | `boolean` | `true` | Boolean type |

### Special Types

| Type Name | Dummy Value | Description |
|-----------|-------------|-------------|
| `date` | `'2024-01-01'` | Date type (YYYY-MM-DD format) |
| `datetime` | `'2024-01-01 00:00:00'` | DateTime type (YYYY-MM-DD HH:MM:SS format) |
| `timestamp` | `'2024-01-01 00:00:00'` | Timestamp type |
| `email` | `'user@example.com'` | Email address type |
| `uuid` | `'00000000-0000-0000-0000-000000000000'` | UUID type |
| `json` | `'{}'` | JSON type |
| `array` | `[]` | Array type |

## Type Definition Syntax

### YAML Format Type Definition

```yaml
parameters:
  # Basic type definitions
  user_id:
    type: int
    description: "User ID"
  
  user_name:
    type: string
    description: "User name"
  
  is_active:
    type: bool
    description: "Active flag"
  
  # Nested objects
  user:
    name:
      type: string
      description: "User name"
    profile:
      email:
        type: email
        description: "Email address"
      age:
        type: int
        description: "Age"
```

### Type Inference

When type is not explicitly specified, it's inferred from the value:

```yaml
parameters:
  # Type inference examples
  user_id: 123          # → int
  user_name: "John"     # → string
  is_active: true       # → bool
  price: 99.99          # → float
```

## Type Resolution for Variable References

### Dot Notation

Nested objects are accessed using dot notation:

```sql
SELECT /*= user.name */ FROM users WHERE id = /*= user.profile.user_id */
```

### Type Resolution Algorithm

1. **Variable Name Decomposition**: `user.profile.name` → `["user", "profile", "name"]`
2. **Hierarchical Type Resolution**: Get type information at each level
3. **Final Type Determination**: Return the type of the last element

```go
// Type resolution example
variableName := "user.profile.email"
// user → map[string]any
// user.profile → map[string]any  
// user.profile.email → type: "email" → dummy value: "user@example.com"
```

## Dummy Literal Generation Rules

### Type to Dummy Value Conversion

```go
func generateDummyValueFromType(paramType string) string {
    switch strings.ToLower(paramType) {
    case "int", "integer", "long":
        return "1"
    case "float", "double", "decimal", "number":
        return "1.0"
    case "bool", "boolean":
        return "true"
    case "string", "text":
        return "'dummy'"
    case "date":
        return "'2024-01-01'"
    case "datetime", "timestamp":
        return "'2024-01-01 00:00:00'"
    case "email":
        return "'user@example.com'"
    case "uuid":
        return "'00000000-0000-0000-0000-000000000000'"
    case "json":
        return "'{}'"
    case "array":
        return "'[]'"
    default:
        return "'dummy'"  // Default to string
    }
}
```

### Token Type Mapping

Mapping from dummy values to TokenType:

```go
func inferTokenTypeFromValue(dummyValue string) tokenizer.TokenType {
    switch {
    case strings.HasPrefix(dummyValue, "'"):
        return tokenizer.STRING
    case dummyValue == "true" || dummyValue == "false":
        return tokenizer.BOOLEAN
    case strings.Contains(dummyValue, "."):
        return tokenizer.NUMBER // Floating-point number
    default:
        return tokenizer.NUMBER // Integer
    }
}
```

## Error Handling

### Type Resolution Errors

1. **Parameter Not Found**: `ErrParameterNotFound`
2. **Not a Nested Object**: `ErrParameterNotNestedObject`
3. **Invalid Type Information**: Fallback to default type (string)

### Error Handling Policy

```go
// Use default value on error
paramType, err := getParameterType(variableName, functionDef.Parameters)
if err != nil {
    paramType = "string"  // Default type
}
```

## Usage Examples

### Complete FunctionDefinition Example

```yaml
# snapsql.yaml
functions:
  getUserProfile:
    description: "Get user profile"
    parameters:
      user_id:
        type: int
        description: "Target user ID"
      include_profile:
        type: bool
        description: "Whether to include profile information"
      filters:
        status:
          type: string
          description: "Status filter"
        created_after:
          type: date
          description: "Creation date filter"
```

### Usage in SQL Templates

```sql
-- getUserProfile.snap.sql
SELECT 
    id,
    name,
    /*= filters.status */ as status,
    created_at
FROM users 
WHERE 
    id = /*= user_id */
    /*@ if include_profile */
    AND profile_id IS NOT NULL
    /*@ end */
    /*@ if filters.created_after */
    AND created_at >= /*= filters.created_after */
    /*@ end */
```

## Implementation Phases

### parserstep1 (Currently Implemented)

- DUMMY_LITERAL token insertion
- Variable name extraction and storage
- Basic syntax validation

### parserstep6 (To Be Implemented)

- Type resolution using FunctionDefinition
- Replacement with actual literal values
- CEL expression evaluation
- Namespace management

## Future Extensions

### Custom Types

```yaml
parameters:
  user_status:
    type: enum
    values: ["active", "inactive", "pending"]
    default: "active"
```

### Detailed Array Type Specification

```yaml
parameters:
  user_ids:
    type: array
    element_type: int
    description: "Array of user IDs"
```

### Constraints

```yaml
parameters:
  age:
    type: int
    min: 0
    max: 150
    description: "Age (0-150)"
```

This type system improves type safety and development efficiency in SnapSQL templates.
