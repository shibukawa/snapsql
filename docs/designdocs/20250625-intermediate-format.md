# SnapSQL Intermediate Format Specification

**Document Version:** 1.1  
**Date:** 2025-06-28  
**Status:** Updated for new CLI and output format

## Overview

This document defines the intermediate JSON format for SnapSQL templates. The intermediate format serves as a bridge between the SQL template parser and code generators, providing a language-agnostic representation of parsed SQL templates with their metadata, AST, and interface schema.

## Design Goals

### 1. Language Agnostic
- JSON format that can be consumed by any programming language
- No language-specific constructs or assumptions
- Clear separation between SQL structure and language-specific metadata

### 2. Complete Information Preservation
- Full AST representation with position information
- Template metadata and interface schema
- Variable references and their types
- Control flow structure (if/for blocks)

### 3. Code Generation Ready
- Structured data suitable for template-based code generation
- Type information for strongly-typed languages
- Function signature information
- Parameter ordering preservation

### 4. Extensible
- Support for future SnapSQL features
- Versioned format for backward compatibility
- Plugin-friendly structure for custom generators

## Intermediate Format Structure

### Top-Level Structure

```json
{
  "source": {
    "file": "queries/users.snap.sql",
    "content": "SELECT id, name FROM users WHERE active = /*= active */true"
  },
  "interface_schema": { /* InterfaceSchema object */ },
  "ast": { /* AST root node */ }
}
```

### Interface Schema Section

```json
{
  "interface_schema": {
    "name": "UserQuery",
    "description": "Query users with optional filtering",
    "function_name": "queryUsers",
    "parameters": [
      {
        "name": "active",
        "type": "bool"
      },
      {
        "name": "filters",
        "type": "object",
        "children": [
          {
            "name": "department",
            "type": "array",
            "children": [
              {
                "name": "o1",
                "type": "string"
              }
            ]
          },
          {
            "name": "role",
            "type": "string"
          }
        ]
      }
    ]
  }
}
```

### AST Section

```json
{
  "ast": {
    "type": "SELECT_STATEMENT",
    "pos": [1, 1, 0],
    "select_clause": {
      "type": "SELECT_CLAUSE",
      "pos": [1, 1, 0],
      "fields": [
        {
          "type": "IDENTIFIER",
          "pos": [1, 8, 7],
          "name": "id"
        },
        {
          "type": "IDENTIFIER", 
          "pos": [1, 12, 11],
          "name": "name"
        }
      ]
    },
    "from_clause": {
      "type": "FROM_CLAUSE",
      "pos": [1, 17, 16],
      "tables": [
        {
          "type": "IDENTIFIER",
          "pos": [1, 22, 21],
          "name": "users"
        }
      ]
    },
    "where_clause": {
      "type": "WHERE_CLAUSE",
      "pos": [1, 28, 27],
      "condition": {
        "type": "EXPRESSION",
        "pos": [1, 34, 33],
        "left": {
          "type": "IDENTIFIER",
          "name": "active"
        },
        "operator": "=",
        "right": {
          "type": "VARIABLE_SUBSTITUTION",
          "pos": [1, 43, 42],
          "variable_name": "active",
          "dummy_value": "true",
          "variable_type": "bool"
        }
      }
    },
    "implicit_if_block": {
      "type": "IMPLICIT_CONDITIONAL",
      "pos": [-1, -1, -1],
      "condition": "active != null",
      "target_clause": "WHERE_CLAUSE"
    }
  }
}
```

#### Position Information
- **Regular nodes**: `pos: [line, column, offset]` from source code
- **Implicit nodes**: `pos: [-1, -1, -1]` for automatically inserted elements

## JSON Schema Definition

The intermediate format includes a JSON Schema definition for validation:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL Intermediate Format",
  "description": "Intermediate JSON format for SnapSQL templates",
  "type": "object",
  "required": ["source", "ast"],
  "properties": {
    "source": {
      "type": "object",
      "required": ["file", "content"],
      "properties": {
        "file": { "type": "string" },
        "content": { "type": "string" }
      }
    },
    "interface_schema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "description": { "type": "string" },
        "function_name": { "type": "string" },
        "parameters": { "type": "array" }
      }
    },
    "ast": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": { "type": "string" },
        "pos": {
          "type": "array",
          "items": { "type": "integer" },
          "minItems": 3,
          "maxItems": 3
        }
      }
    }
  }
}
```

## Implementation Plan

### Phase 1: Core Structure
1. Define intermediate format data structures
2. Implement JSON serialization for AST nodes
3. Create basic conversion from parser output to intermediate format

### Phase 2: Schema Integration
1. Include InterfaceSchema in intermediate format
2. Add recursive parameters structure
3. Implement parameter type analysis

### Phase 3: AST Enhancement
1. Add implicit node position handling (pos: [-1, -1, -1])
2. Enhance AST serialization with complete node information
3. Include variable substitution details in AST nodes

### Phase 4: Validation
1. Implement JSON Schema validation
2. Add format validation utilities
3. Create validation error reporting

### Phase 5: CLI Integration
1. Create `rawparse` subcommand (replaces previous parse/generate separation)
2. Output is always per-file (no date in filename)
3. Output directory is specified with `--output-dir`
4. Output format is always JSON (pretty-print with `--pretty`)
5. Validation is performed with `--validate`
6. No output format option for pull (see database-pull doc)

## Usage Examples

### Command Line Usage
```bash
# Parse single file to intermediate format (JSON)
snapsql rawparse queries/users.snap.sql

# Parse with pretty printing
snapsql rawparse --pretty queries/users.snap.sql

# Parse multiple files to output directory (one JSON per input)
snapsql rawparse queries/*.snap.sql --output-dir generated/

# Validate against schema
snapsql rawparse --validate queries/users.snap.sql
```

### Programmatic Usage
```go
package main

import (
    "github.com/shibukawa/snapsql/intermediate"
    "github.com/shibukawa/snapsql/parser"
)

func main() {
    // Parse SQL template
    ast, schema, err := parser.ParseTemplate(sqlContent)
    if err != nil {
        panic(err)
    }
    
    // Convert to intermediate format
    format := intermediate.NewFormat()
    format.SetSource("queries/users.snap.sql", sqlContent)
    format.SetAST(ast)
    format.SetInterfaceSchema(schema)
    
    // Serialize to JSON
    jsonData, err := format.ToJSON()
    if err != nil {
        panic(err)
    }
    
    // Validate against schema
    if err := format.Validate(); err != nil {
        panic(err)
    }
}
```

## Benefits

### For Code Generators
- Consistent input format across all target languages
- Rich type information for code generation
- Position information for error reporting
- Complete template structure for analysis

### For Tooling
- Language-agnostic template analysis
- IDE integration possibilities
- Template validation and linting
- Documentation generation

### For Debugging
- Complete parse tree visualization
- Variable reference tracking
- Control flow analysis
- Template complexity metrics

## Future Extensions

### Template Analysis
- Dead code detection
- Variable usage analysis
- Performance estimation
- Security analysis

### Multi-Language Support
- Language-specific metadata sections
- Custom type mappings
- Framework-specific annotations
- Code style preferences

### Optimization
- Template simplification
- Query optimization hints
- Caching strategies
- Performance metrics

## Conclusion

The intermediate format provides a robust foundation for SnapSQL's code generation pipeline. By separating parsing from code generation, we enable:

1. **Flexibility**: Support for multiple target languages and frameworks
2. **Maintainability**: Clear separation of concerns
3. **Extensibility**: Easy addition of new features and generators
4. **Tooling**: Rich ecosystem of analysis and development tools

This format will serve as the cornerstone for SnapSQL's multi-language code generation capabilities.
