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
    "source_file": "queries/users.snap.sql",
    "hash": "sha256:...",
    "timestamp": "2025-06-26T10:00:00Z"
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
      "content": "SELECT id, name"
    },
    {
      "type": "conditional",
      "condition": "include_email",
      "instructions": [
        {
          "type": "static",
          "content": ", email"
        }
      ]
    },
    {
      "type": "static",
      "content": " FROM users_"
    },
    {
      "type": "variable",
      "name": "env",
      "validation": {
        "type": "string",
        "pattern": "^[a-z]+$"
      }
    },
    {
      "type": "conditional",
      "condition": "filters.active != null",
      "instructions": [
        {
          "type": "static",
          "content": " WHERE active = "
        },
        {
          "type": "variable",
          "name": "filters.active"
        }
      ]
    }
  ]
}
```

### Metadata Section

* `source_file`: Path to the original SQL template file
* `hash`: Hash of the source file (for change detection)
* `timestamp`: Generation timestamp

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
     "content": "SELECT * FROM"
   }
   ```

2. `variable`: Variable expansion
   ```json
   {
     "type": "variable",
     "name": "table_name",
     "validation": {
       "type": "string",
       "pattern": "^[a-z_]+$"
    }
   }
   ```

3. `conditional`: Conditional branching
   ```json
   {
     "type": "conditional",
     "condition": "include_email",
     "instructions": [...]
   }
   ```

4. `loop`: Iteration
   ```json
   {
     "type": "loop",
     "source": "sort_fields",
     "separator": ", ",
     "instructions": [...]
   }
   ```

5. `array_expansion`: Array expansion (for IN clauses)
   ```json
   {
     "type": "array_expansion",
     "source": "departments",
     "separator": ", ",
     "quote": true
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
