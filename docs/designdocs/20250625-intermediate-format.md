# SnapSQL Intermediate Format Specification

**Document Version:** 2.0  
**Date:** 2025-07-25  
**Status:** Implemented

## Overview

This document defines the intermediate JSON format for SnapSQL templates. The intermediate format serves as a bridge between SQL template parsers and code generators, providing a language-agnostic representation of parsed SQL templates with metadata, CEL expressions, and function definitions.

## Design Goals

### 1. Language Agnostic
- JSON format usable by any programming language
- No language-specific structures or assumptions
- Clear separation between SQL structure and language-specific metadata

### 2. Complete Information Preservation
- Template metadata and function definitions
- Complete extraction of CEL expressions and type information
- Instruction representation of control flow structures (if/for blocks)

### 3. Code Generation Ready
- Structured data suitable for template-based code generation
- Type information for strongly typed languages
- Function signature information
- Parameter order preservation

### 4. Extensible
- Support for future SnapSQL features
- Versioned format for backward compatibility
- Plugin-friendly structure for custom generators

## Intermediate Format Structure

### Top-Level Structure

```json
{
  "format_version": "1",
  "name": "get_user_by_id",
  "function_name": "get_user_by_id",
  "parameters": [/* parameter definitions */],
  "instructions": [/* instruction sequence */],
  "expressions": [/* CEL expression list */],
  "envs": [/* environment variable hierarchy */]
}
```

## CEL Expression Extraction

SnapSQL extracts all CEL expressions from templates. This includes:

1. **Variable substitution**: Expressions in `/*= expression */` format
2. **Conditional expressions**: Condition parts of `/*# if condition */` and `/*# elseif condition */`
3. **Loop expressions**: Collection parts of `/*# for variable : collection */`

### CEL Expression Examples

```sql
-- Variable substitution
SELECT * FROM users WHERE id = /*= user_id */123

-- Conditional expressions
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */

-- Loop expressions
/*# for dept : departments */
SELECT /*= dept.name */'Engineering'
/*# end */

-- Complex expressions
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name
```

### Intermediate Format Representation

```json
{
  "expressions": [
    "user_id",
    "min_age > 0",
    "min_age",
    "departments",
    "dept",
    "dept.name",
    "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")"
  ],
  "envs": [
    [{"name": "dept", "type": "any"}]
  ]
}
```

The `envs` section contains the hierarchical structure of loop variables. Each level contains a list of loop variables defined at that level.

## Function Definition Section

Function definitions contain metadata extracted from template header comments.

```json
{
  "name": "get_user_by_id",
  "function_name": "get_user_by_id",
  "parameters": [
    {"name": "user_id", "type": "int"},
    {"name": "include_details", "type": "bool"}
  ]
}
```

### Parameter Definitions

Parameter definitions are extracted from template header comments or Markdown Parameters sections.

```yaml
# SQL file header comment
/*#
function_name: get_user_by_id
parameters:
  user_id: int
  include_details: bool
*/
```

```markdown
# Markdown file Parameters section
## Parameters

```yaml
user_id: int
include_details: bool
```
```

## Instruction Set

The instruction set is an executable representation of SQL templates. The current implementation supports the following instruction types.

### Instruction Types

#### Basic Output Instructions
- **EMIT_STATIC**: Output static SQL text
- **EMIT_EVAL**: Evaluate CEL expression and output parameter

#### Control Flow Instructions
- **IF**: Start of conditional branch
- **ELSE_IF**: Else if condition
- **ELSE**: Else branch
- **END**: End of control block

#### Loop Instructions
- **LOOP_START**: Start of for loop
- **LOOP_END**: End of for loop

#### System Instructions
- **EMIT_SYSTEM_LIMIT**: Output system LIMIT clause
- **EMIT_SYSTEM_OFFSET**: Output system OFFSET clause

### Instruction Examples

```json
{"op": "EMIT_STATIC", "value": "SELECT id, name FROM users WHERE ", "pos": "1:1"}
{"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
{"op": "IF", "condition": "min_age > 0", "pos": "2:1"}
{"op": "EMIT_STATIC", "value": " AND age >= ", "pos": "3:1"}
{"op": "EMIT_EVAL", "param": "min_age", "pos": "3:12"}
{"op": "ELSE_IF", "condition": "max_age > 0", "pos": "4:1"}
{"op": "EMIT_STATIC", "value": " AND age <= ", "pos": "5:1"}
{"op": "EMIT_EVAL", "param": "max_age", "pos": "5:12"}
{"op": "ELSE", "pos": "6:1"}
{"op": "EMIT_STATIC", "value": " -- No age filter", "pos": "7:1"}
{"op": "END", "pos": "8:1"}
{"op": "LOOP_START", "variable": "dept", "collection": "departments", "pos": "9:1"}
{"op": "EMIT_EVAL", "param": "dept.name", "pos": "10:5"}
{"op": "LOOP_END", "pos": "11:1"}
{"op": "EMIT_SYSTEM_LIMIT", "default_value": "100", "pos": "12:1"}
{"op": "EMIT_SYSTEM_OFFSET", "default_value": "0", "pos": "13:1"}
```

## Implementation Examples

### Simple Variable Substitution

```sql
SELECT id, name, email FROM users WHERE id = /*= user_id */123
```

Intermediate format:

```json
{
  "format_version": "1",
  "name": "get_user_by_id",
  "function_name": "get_user_by_id",
  "parameters": [{"name": "user_id", "type": "int"}],
  "expressions": ["user_id"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name, email FROM users WHERE id = ", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
  ]
}
```

### Conditional Query

```sql
SELECT id, name, age, department 
FROM users
WHERE 1=1
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */
/*# if max_age > 0 */
AND age <= /*= max_age */65
/*# end */
```

Intermediate format:

```json
{
  "format_version": "1",
  "name": "get_filtered_users",
  "function_name": "get_filtered_users",
  "parameters": [
    {"name": "min_age", "type": "int"},
    {"name": "max_age", "type": "int"}
  ],
  "expressions": ["min_age > 0", "min_age", "max_age > 0", "max_age"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name, age, department \nFROM users\nWHERE 1=1", "pos": "1:1"},
    {"op": "IF", "condition": "min_age > 0", "pos": "4:1"},
    {"op": "EMIT_STATIC", "value": "\nAND age >= ", "pos": "5:1"},
    {"op": "EMIT_EVAL", "param": "min_age", "pos": "5:11"},
    {"op": "END", "pos": "6:1"},
    {"op": "IF", "condition": "max_age > 0", "pos": "7:1"},
    {"op": "EMIT_STATIC", "value": "\nAND age <= ", "pos": "8:1"},
    {"op": "EMIT_EVAL", "param": "max_age", "pos": "8:11"},
    {"op": "END", "pos": "9:1"}
  ]
}
```

### IF-ELSE_IF-ELSE Structure

```sql
SELECT 
    id,
    name,
    /*# if user_type == "admin" */
    'Administrator' as role
    /*# elseif user_type == "manager" */
    'Manager' as role
    /*# else */
    'User' as role
    /*# end */
FROM users
WHERE age >= /*= age */18
```

Intermediate format:

```json
{
  "format_version": "1",
  "name": "get_user_with_role",
  "function_name": "get_user_with_role",
  "parameters": [
    {"name": "user_type", "type": "string"},
    {"name": "age", "type": "int"}
  ],
  "expressions": ["user_type == \"admin\"", "user_type == \"manager\"", "age"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name,", "pos": "1:1"},
    {"op": "IF", "condition": "user_type == \"admin\"", "pos": "4:5"},
    {"op": "EMIT_STATIC", "value": "'Administrator' as role", "pos": "5:5"},
    {"op": "ELSE_IF", "condition": "user_type == \"manager\"", "pos": "6:5"},
    {"op": "EMIT_STATIC", "value": "'Manager' as role", "pos": "7:5"},
    {"op": "ELSE", "pos": "8:5"},
    {"op": "EMIT_STATIC", "value": "'User' as role", "pos": "9:5"},
    {"op": "END", "pos": "10:5"},
    {"op": "EMIT_STATIC", "value": "FROM users WHERE age >= ", "pos": "11:1"},
    {"op": "EMIT_EVAL", "param": "age", "pos": "12:14"},
    {"op": "EMIT_STATIC", "value": "18", "pos": "12:24"}
  ]
}
```

### Nested Loops

```sql
INSERT INTO sub_departments (id, name, department_code, department_name)
VALUES
/*# for dept : departments */
    /*# for sub : dept.sub_departments */
    (/*= dept.department_code + "-" + sub.id */'1-101', /*= sub.name */'Engineering Team A', /*= dept.department_code */'1', /*= dept.department_name */'Engineering')
    /*# end */
/*# end */;
```

Intermediate format:

```json
{
  "format_version": "1",
  "name": "insert_all_sub_departments",
  "function_name": "insert_all_sub_departments",
  "parameters": [{"name": "departments", "type": "any"}],
  "expressions": [
    "departments", "dept", "dept.sub_departments", "sub",
    "dept.department_code + \"-\" + sub.id", "sub.name",
    "dept.department_code", "dept.department_name"
  ],
  "envs": [
    [{"name": "dept", "type": "any"}],
    [{"name": "sub", "type": "any"}]
  ],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "INSERT INTO sub_departments (id, name, department_code, department_name) VALUES (", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "dept.department_code + \"-\" + sub.id", "pos": "5:6"},
    {"op": "EMIT_STATIC", "value": "'1-101',", "pos": "5:48"},
    {"op": "EMIT_EVAL", "param": "sub.name", "pos": "5:57"},
    {"op": "EMIT_STATIC", "value": "'Engineering Team A',", "pos": "5:72"},
    {"op": "EMIT_EVAL", "param": "dept.department_code", "pos": "5:94"},
    {"op": "EMIT_STATIC", "value": "'1',", "pos": "5:121"},
    {"op": "EMIT_EVAL", "param": "dept.department_name", "pos": "5:126"},
    {"op": "EMIT_STATIC", "value": "'Engineering') ;", "pos": "5:153"}
  ]
}
```

### Complex Expressions and System Instructions

```sql
SELECT 
  id, 
  name,
  /*= display_name ? username : "Anonymous" */'Anonymous'
FROM users
WHERE 
  /*# if start_date != "" && end_date != "" */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if sort_field != "" */
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name
  /*# end */
LIMIT /*= page_size != 0 ? page_size : 10 */10
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */0
```

Intermediate format:

```json
{
  "format_version": "1",
  "name": "getComplexData",
  "function_name": "getComplexData",
  "parameters": [
    {"name": "user_id", "type": "int"},
    {"name": "username", "type": "string"},
    {"name": "display_name", "type": "bool"},
    {"name": "start_date", "type": "string"},
    {"name": "end_date", "type": "string"},
    {"name": "sort_field", "type": "string"},
    {"name": "sort_direction", "type": "string"},
    {"name": "page_size", "type": "int"},
    {"name": "page", "type": "int"}
  ],
  "expressions": [
    "display_name ? username : \"Anonymous\"",
    "start_date != \"\" && end_date != \"\"",
    "start_date", "end_date",
    "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")",
    "page_size != 0 ? page_size : 10",
    "page > 0 ? (page - 1) * page_size : 0"
  ],
  "instructions": [
    {"op": "IF", "condition": "page > 0 ? (page - 1) * page_size : 0 != null", "pos": "0:0"},
    {"op": "IF", "condition": "page_size != 0 ? page_size : 10 != null", "pos": "0:0"},
    {"op": "IF", "condition": "sort_field != \"\"", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": "SELECT id, name,", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "display_name ? username : \"Anonymous\"", "pos": "4:3"},
    {"op": "EMIT_STATIC", "value": "FROM users WHERE created_at BETWEEN", "pos": "4:3"},
    {"op": "EMIT_EVAL", "param": "start_date", "pos": "8:22"},
    {"op": "EMIT_STATIC", "value": "'2023-01-01' AND", "pos": "8:39"},
    {"op": "EMIT_EVAL", "param": "end_date", "pos": "8:56"},
    {"op": "EMIT_STATIC", "value": "'2023-12-31' ORDER BY", "pos": "8:71"},
    {"op": "EMIT_EVAL", "param": "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")", "pos": "11:10"},
    {"op": "EMIT_STATIC", "value": "name", "pos": "11:83"},
    {"op": "EMIT_SYSTEM_LIMIT", "default_value": "10", "pos": "13:1"},
    {"op": "EMIT_STATIC", "value": "10", "pos": "13:45"},
    {"op": "EMIT_SYSTEM_OFFSET", "default_value": "0", "pos": "14:1"},
    {"op": "END", "pos": "0:0"},
    {"op": "END", "pos": "0:0"},
    {"op": "END", "pos": "0:0"}
  ]
}
```

## JSON Schema Definition

The intermediate format includes a JSON schema definition for validation:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL Intermediate Format",
  "description": "Intermediate JSON format for SnapSQL templates",
  "type": "object",
  "properties": {
    "format_version": {"type": "string", "enum": ["1"]},
    "name": {"type": "string"},
    "function_name": {"type": "string"},
    "parameters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"}
        },
        "required": ["name", "type"]
      }
    },
    "instructions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "op": {
            "type": "string",
            "enum": ["EMIT_STATIC", "EMIT_EVAL", "IF", "ELSE_IF", "ELSE", "END", "LOOP_START", "LOOP_END", "EMIT_SYSTEM_LIMIT", "EMIT_SYSTEM_OFFSET"]
          },
          "value": {"type": "string"},
          "param": {"type": "string"},
          "condition": {"type": "string"},
          "variable": {"type": "string"},
          "collection": {"type": "string"},
          "default_value": {"type": "string"},
          "pos": {"type": "string"}
        },
        "required": ["op"]
      }
    },
    "expressions": {
      "type": "array",
      "items": {"type": "string"}
    },
    "envs": {
      "type": "array",
      "items": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "type": {"type": "string"}
          },
          "required": ["name", "type"]
        }
      }
    }
  },
  "required": ["format_version", "instructions"]
}
```

## Position Information (pos)

Each instruction includes position information (`pos`) indicating the line and column in the original SQL template. The format is `"line:column"`.

- `"1:1"`: Line 1, column 1
- `"5:43"`: Line 5, column 43
- `"0:0"`: System-generated instruction (no position information)

This information is used for debugging and error reporting.

## Future Extensions

The current intermediate format focuses on basic CEL expression extraction and instruction set implementation. Future features may include:

1. **Instruction Set Optimization**: Optimization of instruction sequences for improved execution efficiency
2. **Enhanced Type Inference**: More accurate type information
3. **Response Type Definitions**: Type information for query results
4. **Table Schema Integration**: Integration with database schema information

These extensions will enable more powerful code generation and runtime optimization.
