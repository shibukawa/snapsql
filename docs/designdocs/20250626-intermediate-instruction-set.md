# Intermediate Instruction Set Design

## Overview

SnapSQL parses SQL templates to generate an intermediate instruction set, which is then executed by language-specific runtime libraries to generate dynamic SQL queries. This document describes the design of the intermediate instruction set.

## Design Goals

* Optimize the format for runtime processing rather than preserving raw SQL parsing results
* Simplify implementation of language-specific runtime libraries
* Optimize runtime performance
* Ensure security (prevent SQL injection)

## Intermediate Instruction Set Format

The intermediate instruction set is stored in JSON format:

```json
{
  "metadata": {
    "format_version": "1"
  },
  "parameters": {
    "include_email": {"type": "boolean"},
    "env": {"type": "string", "enum": ["dev", "test", "prod"]},
    "filters": {
      "type": "object",
      "properties": {
        "active": {"type": "boolean"},
        "departments": {"type": "array", "items": {"type": "string"}}
      }
    },
    "sort_fields": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "field": {"type": "string"},
          "direction": {"type": "string", "enum": ["ASC", "DESC"]}
        }
      }
    }
  },
  "instructions": [
    {
      "type": "static",
      "pos": [1, 1, 0],
      "content": "SELECT id, name"
    },
    {
      "type": "jump_if_false",
      "pos": [1, 18, 17],
      "condition": "include_email",
      "target": 3
    },
    {
      "type": "static",
      "pos": [1, 42, 41],
      "content": ", email"
    },
    {
      "type": "static",
      "pos": [1, 65, 64],
      "content": " FROM users_"
    },
    {
      "type": "variable",
      "pos": [1, 76, 75],
      "name": "env",
      "validation": {
        "type": "string",
        "pattern": "^[a-z]+$"
      }
    },
    {
      "type": "jump_if_false",
      "pos": [1, 85, 84],
      "condition": "filters.active != null",
      "target": 8
    },
    {
      "type": "static",
      "pos": [1, 110, 109],
      "content": " WHERE active = "
    },
    {
      "type": "variable",
      "pos": [1, 125, 124],
      "name": "filters.active"
    }
  ]
}
```

### Metadata Section

* `format_version`: Format version. Currently only 1.

### Parameters Section

* Defines parameter type information using JSON Schema format
* Can define constraints for parameters
* Supports nested objects and lists

### Instruction Set

Supports the following instruction types:

1. `static`: Static SQL string
   ```json
   {
     "type": "static",
     "pos": [1, 1, 0],
     "content": "SELECT * FROM"
   }
   ```

2. `variable`: Variable expansion
   ```json
   {
     "type": "variable",
     "pos": [1, 15, 14],
     "name": "table_name",
     "validation": {
       "type": "string",
       "pattern": "^[a-z_]+$"
    }
   }
   ```

3. `jump_if_false`: Jump to specified instruction index if condition is false
   ```json
   {
     "type": "jump_if_false",
     "pos": [1, 25, 24],
     "condition": "include_email",
     "target": 5
   }
   ```

4. `loop_start`: Start of loop block
   ```json
   {
     "type": "loop_start",
     "pos": [1, 30, 29],
     "variable": "item",
     "collection": "sort_fields",
     "end_target": 10
   }
   ```

5. `loop_end`: End of loop block (jumps back to loop start)
   ```json
   {
     "type": "loop_end",
     "pos": [1, 100, 99],
     "start_target": 4
   }
   ```

6. `array_expansion`: Array expansion (for IN clauses)
   ```json
   {
     "type": "array_expansion",
     "pos": [1, 50, 49],
     "source": "departments",
     "separator": ", ",
     "quote": true
   }
   ```

7. `emit_if_not_boundary`: Only emit content if not at a boundary (commas, AND, OR, etc.)
   ```json
   {
     "type": "emit_if_not_boundary",
     "pos": [1, 60, 59],
     "content": ", "
   }
   ```

8. `emit_static_boundary`: Static text that represents a boundary (closing parentheses, clause boundaries, etc.)
   ```json
   {
     "type": "emit_static_boundary",
     "pos": [1, 70, 69],
     "content": ") FROM"
   }
   ```

### Security Features

1. Parameter Validation
   * Type checking
   * Pattern matching
   * Enumeration restrictions
   * Custom validation functions

2. SQL Injection Prevention
   * Prohibit complete dynamic table name generation (only suffixes allowed)
   * All values are converted to placeholders
   * Special character escaping

## Runtime Library Implementation

Language-specific runtime libraries must implement the following features:

1. Loading and parsing of intermediate instruction set
2. Parameter validation
3. Instruction execution and SQL string generation
4. Placeholder management
5. Database-specific adjustments (placeholder format, etc.)

### Error Handling

The following errors must be handled appropriately:

1. Parameter validation errors
2. Invalid instruction set
3. Runtime errors (undefined variables, etc.)
4. Database-specific errors

## Future Extensions

1. More advanced conditional branching (else if, else)
2. Support for custom validation functions
3. Database-specific optimizations
4. Caching functionality
5. Debugging support features

## Limitations

1. Complete dynamic table name generation not allowed
2. Dynamic subquery generation is restricted
3. Column names in ORDER BY and GROUP BY must be predefined
