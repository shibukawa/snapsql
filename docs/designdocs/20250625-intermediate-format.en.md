# SnapSQL Intermediate Format Specification

**Document Version:** 2.0  
**Date:** 2025-07-28  
**Status:** Implemented

## Overview

This document defines the intermediate JSON format for SnapSQL templates. The intermediate format serves as a bridge between SQL template parsers and code generators, providing a language-independent representation of parsed SQL template metadata, CEL expressions, and function definitions.

## Design Goals

### 1. Language Independence
- JSON format usable by any programming language
- No language-specific structures or assumptions
- Clear separation of SQL structure and language-specific metadata

### 2. Complete Information Preservation
- Template metadata and function definitions
- Complete extraction of CEL expressions and type information

### 3. Code Generation Support
- Structured data suitable for template-based code generation
- Type information for strongly typed languages
- Function signature information
- Parameter order preservation

### 4. Extensibility
- Support for future SnapSQL features
- Versioned format for backward compatibility
- Plugin-friendly structure for custom generators

## Items Included in Intermediate Format

### Top-Level Structure

```json
{
  "format_version": "1",
  "description": "Retrieve user information by user ID",
  "function_name": "get_user_by_id", 
  "parameters": [/* Parameter definitions */],
  "implicit_parameters": [/* Implicit parameter definitions */],
  "instructions": [/* Instruction sequence */],
  "expressions": [/* CEL expression list */],
  "envs": [/* Environment variable hierarchy */],
  "responses": [/* Response type definitions */],
  "response_affinity": {/* Response affinity information */}
}
```

### Item Details

#### 1. **format_version** (string)
- **Purpose**: Version management of intermediate format
- **Value**: Currently `"1"`
- **Usage**: Backward compatibility guarantee, parser version compatibility check

#### 2. **description** (string)
- **Purpose**: Template description/overview
- **Source**: Function definition
- **Usage**: Documentation generation, comment output, developer explanation

#### 3. **function_name** (string)
- **Purpose**: Name of the generated function
- **Source**: Function definition
- **Usage**: Function name for code generation, snake_case format

#### 4. **parameters** (array)
- **Purpose**: Definition of explicit template parameters
- **Structure**: `{"name": string, "type": string}`
- **Source**: Function definition
- **Usage**: Type-safe function signature generation, validation

#### 5. **implicit_parameters** (array)
- **Purpose**: Parameters automatically provided by the system (system fields, etc.)
- **Structure**: `{"name": string, "type": string, "default": any}`
- **Source**: System field configuration, LIMIT/OFFSET clause analysis
- **Usage**: Automatic parameter injection at runtime

#### 6. **instructions** (array)
- **Purpose**: Executable representation of SQL template
- **Structure**: Array of instruction objects
- **Source**: Token analysis, control flow analysis
- **Usage**: Dynamic SQL generation at runtime

#### 7. **expressions** (array)
- **Purpose**: All CEL expressions in the template
- **Structure**: Array of CEL expression strings
- **Source**: Directive comments in SQL
- **Usage**: CEL environment construction, expression pre-compilation

#### 8. **envs** (array)
- **Purpose**: Hierarchical structure of loop variables
- **Structure**: `[[{"name": string, "type": string}]]`
- **Source**: Directive comments in SQL
- **Usage**: Variable scope management in nested loops

#### 9. **responses** (array)
- **Purpose**: Type definition of query results
- **Structure**: Array of response type objects
- **Source**: Type inference
- **Usage**: Result type generation, type-safe response handling

#### 10. **response_affinity** (object)
- **Purpose**: Structural information of query results
- **Structure**: Table, column, and type information
- **Source**: Type inference
- **Usage**: Result type generation, ORM integration

## Information Source Mapping

| Intermediate Format Item | Primary Information Source |
|--------------------------|----------------------------|
| `format_version` | Fixed value |
| `description` | Function definition |
| `function_name` | Function definition |
| `parameters` | Function definition |
| `implicit_parameters` | System field configuration |
| `instructions` | Token sequence |
| `expressions` | Directive comments in SQL |
| `envs` | Directive comments in SQL |
| `responses` | Type inference |
| `response_affinity` | Type inference |

### Detailed Information Sources

#### Function Definition Format (SQL File)
```sql
/*#
function_name: get_user_by_id
description: Retrieve user information by user ID
parameters:
  user_id: int
  include_details: bool
*/
```

#### Function Definition Format (Markdown File)
```markdown
## Function Definition
- **Name**: get_user_by_id
- **Description**: Retrieve user information by user ID

## Parameters
```yaml
user_id: int
include_details: bool
```
```
```

#### Directive Comments in SQL
```sql
-- Variable substitution
/*= user_id */

-- Conditional branching
/*# if min_age > 0 */

-- Loop
/*# for dept : departments */
```

#### System Field Configuration (snapsql.yaml)
```yaml
system:
  fields:
    - name: updated_at
      type: timestamp
      on_update:
        parameter: implicit
```

## Processing Flow

### 1. Parser Phase (parser package)

#### parserstep2: Basic Structure Analysis
- Tokenize SQL string
- Build basic structure of StatementNode (AST)
- Identify WITH clause and various clauses

#### parserstep3: Syntax Validation
- Detect basic syntax errors

#### parserstep4: Detailed Clause Analysis
Execute detailed analysis and validation for each clause:

**SELECT Statement Processing Order:**
1. `finalizeSelectClause()` - Detailed analysis of SELECT clause
2. `finalizeFromClause()` - Detailed analysis of FROM clause  
3. `emptyCheck(WHERE)` - Empty check for WHERE clause
4. `finalizeGroupByClause()` - Detailed analysis of GROUP BY clause
5. `finalizeHavingClause()` - Detailed analysis of HAVING clause (relationship check with GROUP BY)
6. `finalizeOrderByClause()` - Detailed analysis of ORDER BY clause
7. `finalizeLimitOffsetClause()` - Detailed analysis of LIMIT/OFFSET clause
8. `emptyCheck(FOR)` - Empty check for FOR clause

**INSERT Statement Processing Order:**
1. `finalizeInsertIntoClause()` - Detailed analysis of INSERT INTO clause (column list analysis)
2. Reflection to `InsertIntoStatement.Columns`
3. `emptyCheck(WITH)` - Empty check for WITH clause
4. If SELECT part exists, execute above SELECT statement processing
5. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

**UPDATE Statement Processing Order:**
1. `finalizeUpdateClause()` - Detailed analysis of UPDATE clause
2. `finalizeSetClause()` - Detailed analysis of SET clause
3. `emptyCheck(WHERE)` - Empty check for WHERE clause
4. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

**DELETE Statement Processing Order:**
1. `finalizeDeleteFromClause()` - Detailed analysis of DELETE FROM clause
2. `emptyCheck(WHERE)` - Empty check for WHERE clause
3. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

#### parserstep5: Advanced Processing
1. `expandArraysInValues()` - Array expansion in VALUES clause
2. `detectDummyRanges()` - Dummy value detection
3. `applyImplicitIfConditions()` - Apply implicit if conditions to LIMIT/OFFSET clause
4. `validateAndLinkDirectives()` - Directive validation and linking

### 2. Intermediate Format Generation Phase (intermediate package)

#### Preprocessing
1. `ExtractFromStatement()` - Extract CEL expressions and environment variables
2. Parameter extraction from function definition
3. `extractSystemFieldsInfo()` - Extract system field information

#### System Field Processing
1. `CheckSystemFields()` - System field validation and implicit parameter generation
2. System field addition by statement type:
   - UPDATE statement: `AddSystemFieldsToUpdate()` - Add to SET clause
   - INSERT statement: `AddSystemFieldsToInsert()` - Add to column list and VALUES clause

#### Instruction Generation
1. `extractTokensFromStatement()` - Extract token sequence from StatementNode
2. `detectDialectPatterns()` - Detect dialect-specific patterns
3. `generateInstructions()` - Generate execution instructions
4. `detectResponseAffinity()` - Detect response affinity

## Clause-Specific Processing Details

### INSERT INTO Clause
- **Processing**: Column list analysis, table name extraction
- **Output**: `InsertIntoClause.Columns[]`, `InsertIntoClause.Table`
- **Subsequent Processing**: Reflection to `InsertIntoStatement.Columns`

### VALUES Clause  
- **Processing**: Value list analysis, array expansion
- **Output**: Structured value tokens
- **Subsequent Processing**: System field value addition

### SELECT Clause
- **Processing**: Selected field analysis, function call validation
- **Output**: Structured field list

### FROM Clause
- **Processing**: Table reference, JOIN syntax analysis
- **Output**: Structured table references

### WHERE/HAVING Clause
- **Processing**: Conditional expression validation (empty check only)
- **Output**: Syntax error detection

### GROUP BY Clause
- **Processing**: Grouping field analysis
- **Output**: Structured grouping conditions

### ORDER BY Clause
- **Processing**: Sort condition analysis, ASC/DESC specification validation
- **Output**: Structured sort conditions

### LIMIT/OFFSET Clause
- **Processing**: Limit value validation, implicit if condition application
- **Output**: Structured limit conditions

### SET Clause (UPDATE)
- **Processing**: Update field and value analysis
- **Output**: Structured update conditions
- **Subsequent Processing**: System field addition

### RETURNING Clause
- **Processing**: Return value field analysis
- **Output**: Structured return values

## Database Dialect Support

### Overview

SnapSQL supports multiple database dialects (PostgreSQL, MySQL, SQLite) and automatically converts dialect-specific syntax. In the intermediate format, dialect-specific processing is represented as instructions.

### Supported Dialect-Specific Syntax

#### 1. **PostgreSQL Cast Operator (`::`)**
```sql
-- PostgreSQL-specific
SELECT price::DECIMAL(10,2) FROM products

-- Standard SQL conversion
SELECT CAST(price AS DECIMAL(10,2)) FROM products
```

#### 2. **MySQL LIMIT Syntax**
```sql
-- MySQL-specific
SELECT * FROM users LIMIT 10, 20

-- Standard SQL conversion  
SELECT * FROM users LIMIT 20 OFFSET 10
```

#### 3. **SQLite Type Affinity**
```sql
-- SQLite-specific
CREATE TABLE users (id INTEGER PRIMARY KEY)

-- Other DB conversion
CREATE TABLE users (id SERIAL PRIMARY KEY)
```

### Representation in Intermediate Format

Dialect-specific processing is represented as dedicated instructions:

```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT ", "pos": "1:1"},
  {"op": "EMIT_STATIC", "value": "CAST(price AS DECIMAL(10,2))", "pos": "1:8"},
    {"op": "EMIT_STATIC", "value": " FROM products", "pos": "1:25"}
  ]
}
```

### Dialect Detection Patterns

#### PostgreSQL Cast Detection
```go
// detectPostgreSQLCast function
// Pattern: expression::type
if token.Type == tok.DOUBLE_COLON {
    // Parse preceding expression and following type
    // Convert to CAST(expression AS type)
}
```

#### MySQL Function Detection
```go
// detectMySQLFunctions function  
// Detect MySQL-specific functions like NOW(), RAND()
// Convert to equivalent functions in other dialects
```

### Runtime Dialect Selection

Runtime libraries execute appropriate dialect instructions based on the connected database:

```go
// Runtime dialect selection example
switch currentDialect {
case "postgresql":
    executePostgreSQLInstructions()
case "mysql": 
    executeMySQLInstructions()
case "sqlite":
    executeSQLiteInstructions()
}
```

## Trailing Delimiter Handling

### Overview

SnapSQL automatically handles trailing delimiters (commas, AND, OR) for elements that are dynamically added or removed through conditional branching. This allows developers to write templates without worrying about syntax errors.

### Target Delimiters

#### 1. **Comma (`,`)**
- Field list in SELECT clause
- Column list in INSERT statement
- Value list in VALUES clause

#### 2. **AND Operator**
- Condition joining in WHERE clause
- Condition joining in HAVING clause

#### 3. **OR Operator**
- Condition joining in WHERE clause (selective)

### Processing Examples

#### Trailing Comma Processing in SELECT Clause

**Template:**
```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# if include_phone */
    phone,
    /*# end */
    created_at
FROM users
```

**Output by conditions:**
```sql
-- When include_email=true, include_phone=false
SELECT id, name, email, created_at FROM users

-- When include_email=false, include_phone=true  
SELECT id, name, phone, created_at FROM users

-- When both are false
SELECT id, name, created_at FROM users
```

#### AND Processing in WHERE Clause

**Template:**
```sql
SELECT * FROM users 
WHERE active = true
    /*# if min_age > 0 */
    AND age >= /*= min_age */18
    /*# end */
    /*# if department != "" */
    AND department = /*= department */'Engineering'
    /*# end */
```

**Output by conditions:**
```sql
-- When min_age=25, department="Sales"
SELECT * FROM users WHERE active = true AND age >= 25 AND department = 'Sales'

-- When min_age=0, department=""
SELECT * FROM users WHERE active = true

-- When min_age=25, department=""
SELECT * FROM users WHERE active = true AND age >= 25
```

### Representation in Intermediate Format

Trailing delimiter processing is represented by boundary detection (`BOUNDARY`) instructions:

```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name", "pos": "1:1"},
    {"op": "IF", "condition": "include_email", "pos": "4:5"},
    {"op": "BOUNDARY", "pos": "4:5"},
    {"op": "EMIT_STATIC", "value": ", email", "pos": "5:5"},
    {"op": "END", "pos": "6:5"},
    {"op": "IF", "condition": "include_phone", "pos": "7:5"},
    {"op": "BOUNDARY", "pos": "7:5"},
    {"op": "EMIT_STATIC", "value": ", phone", "pos": "8:5"},
    {"op": "END", "pos": "9:5"},
    {"op": "EMIT_STATIC", "value": ", created_at FROM users", "pos": "11:5"}
  ]
}
```

### Boundary Detection Algorithm

#### 1. **Boundary Token Identification**
```go
// Token types that serve as boundaries
switch token.Type {
case tok.FROM, tok.WHERE, tok.ORDER, tok.GROUP, 
     tok.HAVING, tok.LIMIT, tok.OFFSET, tok.UNION,
     tok.CLOSED_PARENS:
    return true
}
```

#### 2. **Delimiter Type Determination**
```go
// Determine delimiter based on context
func detectBoundaryDelimiter(context string) string {
    switch context {
    case "SELECT_FIELDS":
        return ","
    case "WHERE_CONDITIONS":
        return "AND"
    case "VALUES_LIST":
        return ","
    }
}
```

#### 3. **Runtime Delimiter Insertion**
```go
// Runtime dynamic delimiter processing
if hasNextElement && !isLastElement {
    output += delimiter // Insert comma or AND/OR
}
```

### Special Cases

#### 1. **Conditional Output of LIMIT/OFFSET Clause**
```sql
-- When LIMIT/OFFSET clause is absent, system automatically adds
SELECT * FROM users

-- Output: Added only when system LIMIT/OFFSET is available
SELECT * FROM users LIMIT 10 OFFSET 0
```

Representation in intermediate format:
```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT * FROM users", "pos": "1:1"},
    {"op": "IF_SYSTEM_LIMIT", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": " LIMIT ", "pos": "0:0"},
    {"op": "EMIT_SYSTEM_LIMIT", "pos": "0:0"},
    {"op": "END", "pos": "0:0"},
    {"op": "IF_SYSTEM_OFFSET", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": " OFFSET ", "pos": "0:0"},
    {"op": "EMIT_SYSTEM_OFFSET", "pos": "0:0"},
    {"op": "END", "pos": "0:0"}
  ]
}
```

#### 2. **Processing in Nested Conditions**
```sql
SELECT *
FROM users
WHERE (
    /*# if condition1 */
    field1 = 'value1'
    /*# end */
    /*# if condition2 */
    AND field2 = 'value2'  
    /*# end */
)
```

#### 3. **OR Operator Processing**
```sql
WHERE (
    /*# if search_name */
    name LIKE /*= search_name */'%John%'
    /*# end */
    /*# if search_email */
    OR email LIKE /*= search_email */'%john%'
    /*# end */
)
```

### Benefits

1. **Syntax Error Prevention**: No need for manual delimiter management
2. **Improved Readability**: Templates can be written in a more natural form
3. **Maintainability**: Reduced risk of syntax errors when adding/removing conditions
4. **Flexibility**: Generates correct SQL even with complex conditional branching
5. **System Integration**: Pagination support through automatic LIMIT/OFFSET clause addition

## Issues and Improvement Proposals

### Current Problems
1. **Complexity of StatementNode Updates**: After adding system fields, StatementNode changes require token re-extraction
2. **Distributed Processing**: Clause-specific processing is scattered between parserstep4 and intermediate
3. **Complexity of Token-Level Operations**: Direct token manipulation of InsertIntoClause is required

### Improvement Proposal: Pipeline Processing
```
SQL String
↓
tokenizer: SQL → Token sequence
↓  
parser: Token sequence → StatementNode (structure analysis only)
↓
System field analysis: StatementNode → ImplicitParameter[]
↓
Token processing pipeline:
  Token sequence → clause-specific conversion → system field insertion → instruction generation
↓
Intermediate format JSON
```

**Benefits:**
- StatementNode used only for structure analysis
- Define token conversion rules for each clause
- Testable independent pipeline stages
- Flexible token-level operations
- Instruction representation of control flow structures (if/for blocks)

### 1. Parser Phase (parser package)

#### parserstep2: Basic Structure Analysis
- Tokenize SQL string
- Build basic structure of StatementNode (AST)
- Identify WITH clause and various clauses

#### parserstep3: Syntax Validation
- Detect basic syntax errors

#### parserstep4: Detailed Clause Analysis
Execute detailed analysis and validation for each clause:

**SELECT Statement Processing Order:**
1. `finalizeSelectClause()` - Detailed analysis of SELECT clause
2. `finalizeFromClause()` - Detailed analysis of FROM clause  
3. `emptyCheck(WHERE)` - Empty check for WHERE clause
4. `finalizeGroupByClause()` - Detailed analysis of GROUP BY clause
5. `finalizeHavingClause()` - Detailed analysis of HAVING clause (relationship check with GROUP BY)
6. `finalizeOrderByClause()` - Detailed analysis of ORDER BY clause
7. `finalizeLimitOffsetClause()` - Detailed analysis of LIMIT/OFFSET clause
8. `emptyCheck(FOR)` - Empty check for FOR clause

**INSERT Statement Processing Order:**
1. `finalizeInsertIntoClause()` - Detailed analysis of INSERT INTO clause (column list analysis)
2. Reflection to `InsertIntoStatement.Columns`
3. `emptyCheck(WITH)` - Empty check for WITH clause
4. If SELECT part exists, execute above SELECT statement processing
5. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

**UPDATE Statement Processing Order:**
1. `finalizeUpdateClause()` - Detailed analysis of UPDATE clause
2. `finalizeSetClause()` - Detailed analysis of SET clause
3. `emptyCheck(WHERE)` - Empty check for WHERE clause
4. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

**DELETE Statement Processing Order:**
1. `finalizeDeleteFromClause()` - Detailed analysis of DELETE FROM clause
2. `emptyCheck(WHERE)` - Empty check for WHERE clause
3. `finalizeReturningClause()` - Detailed analysis of RETURNING clause

#### parserstep5: Advanced Processing
1. `expandArraysInValues()` - Array expansion in VALUES clause
2. `detectDummyRanges()` - Dummy value detection
3. `applyImplicitIfConditions()` - Apply implicit if conditions to LIMIT/OFFSET clause
4. `validateAndLinkDirectives()` - Directive validation and linking

### 2. Intermediate Format Generation Phase (intermediate package)

#### Preprocessing
1. `ExtractFromStatement()` - Extract CEL expressions and environment variables
2. Parameter extraction from function definition
3. `extractSystemFieldsInfo()` - Extract system field information

#### System Field Processing
1. `CheckSystemFields()` - System field validation and implicit parameter generation
2. System field addition by statement type:
   - UPDATE statement: `AddSystemFieldsToUpdate()` - Add to SET clause
   - INSERT statement: `AddSystemFieldsToInsert()` - Add to column list and VALUES clause

#### Instruction Generation
1. `extractTokensFromStatement()` - Extract token sequence from StatementNode
2. `detectDialectPatterns()` - Detect dialect-specific patterns
3. `generateInstructions()` - Generate execution instructions
4. `detectResponseAffinity()` - Detect response affinity

## Clause-Specific Processing Details

### INSERT INTO Clause
- **Processing**: Column list analysis, table name extraction
- **Output**: `InsertIntoClause.Columns[]`, `InsertIntoClause.Table`
- **Subsequent Processing**: Reflection to `InsertIntoStatement.Columns`

### VALUES Clause  
- **Processing**: Value list analysis, array expansion
- **Output**: Structured value tokens
- **Subsequent Processing**: System field value addition

### SELECT Clause
- **Processing**: Selected field analysis, function call validation
- **Output**: Structured field list

### FROM Clause
- **Processing**: Table reference, JOIN syntax analysis
- **Output**: Structured table references

### WHERE/HAVING Clause
- **Processing**: Conditional expression validation (empty check only)
- **Output**: Syntax error detection

### GROUP BY Clause
- **Processing**: Grouping field analysis
- **Output**: Structured grouping conditions

### ORDER BY Clause
- **Processing**: Sort condition analysis, ASC/DESC specification validation
- **Output**: Structured sort conditions

### LIMIT/OFFSET Clause
- **Processing**: Limit value validation, implicit if condition application
- **Output**: Structured limit conditions
### SET Clause (UPDATE)
- **Processing**: Update field and value analysis
- **Output**: Structured update conditions
- **Subsequent Processing**: System field addition

### RETURNING Clause
- **Processing**: Return value field analysis
- **Output**: Structured return values

## Issues and Improvement Proposals

### Current Problems
1. **Complexity of StatementNode Updates**: After adding system fields, StatementNode changes require token re-extraction
2. **Distributed Processing**: Clause-specific processing is scattered between parserstep4 and intermediate
3. **Complexity of Token-Level Operations**: Direct token manipulation of InsertIntoClause is required

### Improvement Proposal: Pipeline Processing
```
SQL String
↓
tokenizer: SQL → Token sequence
↓  
parser: Token sequence → StatementNode (structure analysis only)
↓
System field analysis: StatementNode → ImplicitParameter[]
↓
Token processing pipeline:
  Token sequence → clause-specific conversion → system field insertion → instruction generation
↓
Intermediate format JSON
```

**Benefits:**
- StatementNode used only for structure analysis
- Define token conversion rules for each clause
- Testable independent pipeline stages
- Flexible token-level operations
- Instruction representation of control flow structures (if/for blocks)

### 3. Code Generation Support
- Structured data suitable for template-based code generation
- Type information for strongly typed languages
- Function signature information
- Parameter order preservation

### 4. Extensibility
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
  "parameters": [/* Parameter definitions */],
  "implicit_parameters": [/* Implicit parameter definitions */],
  "instructions": [/* Instruction sequence */],
  "expressions": [/* CEL expression list */],
  "envs": [/* Environment variable hierarchy */]
}
```

## CEL Expression Extraction

SnapSQL extracts all CEL expressions from templates. This includes:

1. **Variable Substitution**: Expressions in `/*= expression */` format
2. **Conditional Expressions**: Condition parts of `/*# if condition */` and `/*# elseif condition */`
3. **Loop Expressions**: Collection parts of `/*# for variable : collection */`

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

### Representation in Intermediate Format

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
  "description": "Retrieve user information by user ID",
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
# Header comment in SQL file
/*#
function_name: get_user_by_id
parameters:
  user_id: int
  include_details: bool
*/
```

````markdown
# Parameters section in Markdown file
## Parameters

```yaml
user_id: int
include_details: bool
```
````

## Instruction Set

The instruction set is an executable representation of SQL templates. The current implementation supports the following instruction types.

### Instruction Types

#### Basic Output Instructions
- **EMIT_STATIC**: Output static SQL text
- **EMIT_EVAL**: Evaluate CEL expression and output parameter

#### Control Flow Instructions
- **IF**: Start of conditional branching
- **ELSE_IF**: else if condition
- **ELSE**: else branch
- **END**: End of control block

#### Loop Instructions
- **LOOP_START**: Start of for loop
- **LOOP_END**: End of for loop

#### System Instructions
- **EMIT_SYSTEM_LIMIT**: Output system LIMIT clause
- **EMIT_SYSTEM_OFFSET**: Output system OFFSET clause
- **EMIT_SYSTEM_VALUE**: Output system field value

#### Boundary Processing Instructions
- **BOUNDARY**: Handle trailing delimiters (comma, AND, OR)

#### Dialect Support Instructions
-- Dialect-specific switching instruction: Output database dialect-specific SQL fragments (the pipeline typically resolves these into final fragments)

### Instruction Examples

```json
{"op": "EMIT_STATIC", "value": "SELECT id, name FROM users WHERE ", "pos": "1:1"}
{"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
{"op": "IF", "condition": "min_age > 0", "pos": "2:1"}
{"op": "BOUNDARY", "pos": "2:1"}
{"op": "EMIT_STATIC", "value": " AND age >= ", "pos": "3:1"}
{"op": "EMIT_EVAL", "param": "min_age", "pos": "3:12"}
{"op": "ELSE_IF", "condition": "max_age > 0", "pos": "4:1"}
{"op": "BOUNDARY", "pos": "4:1"}
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
{"op": "EMIT_STATIC", "value": "price::DECIMAL(10,2)", "pos": "14:1"}
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
  "description": "Retrieve user by user ID",
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
  "description": "User search by age conditions",
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

The intermediate format includes JSON schema definition for validation:

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
            "enum": [
              "EMIT_STATIC", "EMIT_EVAL", "IF", "ELSE_IF", "ELSE", "END", 
              "LOOP_START", "LOOP_END", "EMIT_SYSTEM_LIMIT", "EMIT_SYSTEM_OFFSET", 
              "EMIT_SYSTEM_VALUE", "BOUNDARY"
            ]
          },
          "value": {"type": "string"},
          "param": {"type": "string"},
          "condition": {"type": "string"},
          "variable": {"type": "string"},
          "collection": {"type": "string"},
          "default_value": {"type": "string"},
          "system_field": {"type": "string"},
          "dialect": {"type": "string"},
          "sql_fragment": {"type": "string"},
          "pos": {"type": "string"}
        },
        "required": ["op"]
      }
    },
    "expressions": {
      "type": "array",
      "items": {"type": "string"}
    },
    "implicit_parameters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"},
          "default": {"type": ["string", "number", "boolean", "null"]}
        },
        "required": ["name", "type"]
      }
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
    },
    "responses": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"},
          "nullable": {"type": "boolean"}
        },
        "required": ["name", "type"]
      }
    },
    "response_affinity": {
      "type": "object",
      "properties": {
        "tables": {
          "type": "array",
          "items": {"type": "string"}
        },
        "columns": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "type": {"type": "string"},
              "table": {"type": "string"}
            },
            "required": ["name"]
          }
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

## Response Affinity

### Overview

Response affinity is information determined by type inference that analyzes the structure of SQL query results. This information is used for result type definition during code generation and ORM integration.

### Calculation Logic

#### 1. **Affinity Type Determination**

```go
type ResponseAffinity string

const (
    ResponseAffinityOne  ResponseAffinity = "one"  // Single record
    ResponseAffinityMany ResponseAffinity = "many" // Multiple records
    ResponseAffinityNone ResponseAffinity = "none" // No records
)
```

#### 2. **Determination Algorithm**

**Single Record (`one`) Conditions:**
- SELECT
  - Complete match search by PRIMARY KEY
  - Complete match search by UNIQUE constraint fields
  - `LIMIT 1` explicitly specified
  - Aggregate functions like COUNT(), SUM(), AVG() (return single values)
  - Contains JOIN but driving table element is uniquely determined by LEFT INNER JOIN or INNER JOIN, and joined tables are specified to be processed as arrays
- INSERT
  - Single element INSERT with RETURNING clause
- UPDATE
  - UPDATE with RETURNING clause and complete match search by PRIMARY KEY
- DELETE
  - DELETE with RETURNING clause and complete match search by PRIMARY KEY

**Multiple Records (`many`) Conditions:**
- SELECT statements other than above
- INSERT with RETURNING clause and multiple elements
- UPDATE/DELETE with RETURNING clause but WHERE clause does not have complete match by primary key

**No Response (`none`) Conditions:**
- INSERT statement without RETURNING clause
- UPDATE statement without RETURNING clause  
- DELETE statement without RETURNING clause

#### 3. **Implementation Example**

```go
func detectResponseAffinity(stmt parser.StatementNode, tableInfo *TableInfo) ResponseAffinity {
    switch s := stmt.(type) {
    case *parser.SelectStatement:
        return detectSelectAffinity(s, tableInfo)
    case *parser.InsertStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    case *parser.UpdateStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    case *parser.DeleteStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    default:
        return ResponseAffinityNone
    }
}

func detectSelectAffinity(stmt *parser.SelectStatement, tableInfo *TableInfo) ResponseAffinity {
    // When LIMIT 1 is explicitly specified
    if hasExplicitLimitOne(stmt.Limit) {
        return ResponseAffinityOne
    }
    
    // Search by PRIMARY KEY or UNIQUE constraint
    if hasUniqueKeyCondition(stmt, tableInfo) {
        return ResponseAffinityOne
    }
    
    // Multiple rows possible when GROUP BY exists
    if stmt.GroupBy != nil {
        return ResponseAffinityMany
    }
    
    // Default is multiple records
    return ResponseAffinityMany
}

func detectReturningAffinity(returning *parser.ReturningClause, tableInfo *TableInfo) ResponseAffinity {
    // INSERT RETURNING depends on number of inserted records
    if stmt, ok := returning.Parent.(*parser.InsertStatement); ok {
        if isMultiRowInsert(stmt) {
            return ResponseAffinityMany // Multi-row INSERT
        }
        return ResponseAffinityOne // Single-row INSERT
    }
    
    // UPDATE/DELETE RETURNING depends on conditions
    if stmt, ok := returning.Parent.(*parser.UpdateStatement); ok {
        if hasUniqueKeyCondition(stmt, tableInfo) {
            return ResponseAffinityOne
        }
        return ResponseAffinityMany
    }
    
    if stmt, ok := returning.Parent.(*parser.DeleteStatement); ok {
        if hasUniqueKeyCondition(stmt, tableInfo) {
            return ResponseAffinityOne
        }
        return ResponseAffinityMany
    }
    
    return ResponseAffinityMany
}

func isMultiRowInsert(stmt *parser.InsertStatement) bool {
    // When multiple rows are specified in VALUES clause
    if stmt.ValuesList != nil && len(stmt.ValuesList.Values) > 1 {
        return true
    }
    
    // INSERT from SELECT statement (INSERT INTO ... SELECT)
    if stmt.SelectStatement != nil {
        return true // SELECT results may have multiple rows
    }
    
    return false
}
```

#### 4. **Utilizing Table Information**

```go
type TableInfo struct {
    Name        string
    PrimaryKeys []string
    UniqueKeys  [][]string // Support for composite UNIQUE constraints
    Columns     []ColumnInfo
}

type ColumnInfo struct {
    Name     string
    Type     string
    Nullable bool
}
```

#### 5. **WHERE Clause Analysis**

```go
func hasUniqueKeyCondition(stmt *parser.SelectStatement, tableInfo *TableInfo) bool {
    whereConditions := extractWhereConditions(stmt.Where)
    
    // PRIMARY KEY complete match check
    if matchesAllPrimaryKeys(whereConditions, tableInfo.PrimaryKeys) {
        return true
    }
    
    // UNIQUE constraint complete match check
    for _, uniqueKey := range tableInfo.UniqueKeys {
        if matchesAllUniqueKeys(whereConditions, uniqueKey) {
            return true
        }
    }
    
    return false
}
```

### Representation in Intermediate Format

```json
{
  "response_affinity": {
    "type": "one",
    "tables": ["users"],
    "columns": [
      {"name": "id", "type": "int", "table": "users"},
      {"name": "name", "type": "string", "table": "users"},
      {"name": "email", "type": "string", "table": "users"}
    ],
    "reasoning": "PRIMARY KEY condition detected: users.id = ?"
  }
}
```

### Calculation Examples

#### Example 1: Single Record (PRIMARY KEY Search)
```sql
SELECT id, name, email FROM users WHERE id = /*= user_id */123
```

**Calculation Result:**
- **Type**: `one`
- **Reason**: Complete match search by PRIMARY KEY (`id`)
- **Table**: `users`
- **Columns**: `id`, `name`, `email`

#### Example 2: Single Record (Aggregate Function)
```sql
SELECT COUNT(*) as total FROM users WHERE active = true
```

**Calculation Result:**
- **Type**: `one`
- **Reason**: Aggregate functions return single values
- **Table**: `users`
- **Columns**: `total` (calculated field)

#### Example 3: Multiple Records
```sql
SELECT id, name FROM users WHERE department = /*= dept */'Engineering'
```

**Calculation Result:**
- **Type**: `many`
- **Reason**: Search by non-UNIQUE condition
- **Table**: `users`
- **Columns**: `id`, `name`

#### Example 4: No Response (INSERT Statement)
```sql
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com')
```

**Calculation Result:**
- **Type**: `none`
- **Reason**: INSERT statement without RETURNING clause
- **Table**: `users`
- **Columns**: None

#### Example 5: Single Record (Single-row INSERT with RETURNING)
```sql
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com') RETURNING id, created_at
```

**Calculation Result:**
- **Type**: `one`
- **Reason**: Single-row INSERT statement
- **Table**: `users`
- **Columns**: `id`, `created_at`

#### Example 6: Multiple Records (Multi-row INSERT with RETURNING)
```sql
INSERT INTO users (name, email) VALUES 
    (/*= user1.name */'John', /*= user1.email */'john@example.com'),
    (/*= user2.name */'Jane', /*= user2.email */'jane@example.com')
RETURNING id, created_at
```

**Calculation Result:**
- **Type**: `many`
- **Reason**: Multi-row INSERT statement
- **Table**: `users`
- **Columns**: `id`, `created_at`

#### Example 7: Multiple Records (INSERT from SELECT with RETURNING)
```sql
INSERT INTO users_backup (name, email) 
SELECT name, email FROM users WHERE active = false 
RETURNING id, created_at
```

**Calculation Result:**
- **Type**: `many`
- **Reason**: INSERT from SELECT statement (multiple rows possible)
- **Table**: `users_backup`
- **Columns**: `id`, `created_at`

#### Example 8: Multiple Records (UPDATE with RETURNING)
```sql
UPDATE users SET active = false WHERE department = /*= dept */'Engineering' RETURNING id, name
```

**Calculation Result:**
- **Type**: `many`
- **Reason**: Multiple rows may be updated
- **Table**: `users`
- **Columns**: `id`, `name`

### Use Cases

#### 1. **Code Generation Applications**
```go
// Go language generation example
func GetUser(ctx context.Context, userID int) (*User, error)             // one (SELECT)
func CountUsers(ctx context.Context) (int, error)                       // one (COUNT)
func GetUsers(ctx context.Context, dept string) ([]*User, error)        // many (SELECT)
func CreateUser(ctx context.Context, user *User) error                  // none (single INSERT)
func CreateUserWithID(ctx context.Context, user *User) (*User, error)   // one (single INSERT RETURNING)
func CreateUsers(ctx context.Context, users []*User) ([]*User, error)   // many (multiple INSERT RETURNING)
func UpdateUsers(ctx context.Context, dept string) ([]*User, error)     // many (UPDATE RETURNING)
func DeleteUser(ctx context.Context, userID int) error                  // none (DELETE)
```

#### 2. **Type Safety Improvement**
- **`one`**: Returns single object or null
- **`many`**: Returns array (empty array possible)
- **`none`**: Returns only error (no response data)

#### 3. **ORM Integration**
- **`one`**: Generate `FindOne()`, `First()`, `Count()` methods
- **`many`**: Generate `FindAll()`, `Where()` methods
- **`none`**: Generate `Create()`, `Update()`, `Delete()` methods (no return value)

## System Fields Feature

### Overview

The system fields feature automatically manages common fields required for database audit logs and version control (such as `created_at`, `updated_at`, `created_by`, `updated_by`, `lock_no`).

### Configuration

System fields are defined in the configuration file (`snapsql.yaml`):

```yaml
system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:
        default: "NOW()"
      on_update:
        parameter: error
    - name: updated_at
      type: timestamp
      on_insert:
        default: "NOW()"
        default: "NOW()"
      on_update:
        default: "NOW()"
    - name: created_by
      type: string
      on_insert:
        parameter: implicit
      on_update:
        parameter: error
    - name: updated_by
      type: string
      on_insert:
        parameter: implicit
      on_update:
        parameter: implicit
    - name: lock_no
      type: int
      on_insert:
        default: 1
      on_update:
        parameter: explicit
```

### Parameter Configuration

Each system field can be configured individually for INSERT and UPDATE statement behavior:

- **`explicit`**: Parameter must be explicitly provided
- **`implicit`**: Runtime automatically provides the value (e.g., user ID, session information)
- **`error`**: Error if parameter is provided
- **`default`**: Use default value

### Impact on Intermediate Format

#### Implicit Parameters

Based on system field configuration, implicit parameters are added to the intermediate format:

```json
{
  "format_version": "1",
  "name": "update_user",
  "function_name": "update_user",
  "parameters": [
    {"name": "name", "type": "string"},
    {"name": "email", "type": "string"},
    {"name": "lock_no", "type": "int"}
  ],
  "implicit_parameters": [
    {"name": "updated_at", "type": "timestamp", "default": "NOW()"},
    {"name": "updated_by", "type": "string"}
  ],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "UPDATE users SET name = ", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "name", "pos": "1:25"},
    {"op": "EMIT_STATIC", "value": ", email = ", "pos": "1:30"},
    {"op": "EMIT_EVAL", "param": "email", "pos": "1:40"},
    {"op": "EMIT_STATIC", "value": ", EMIT_SYSTEM_VALUE(updated_at), EMIT_SYSTEM_VALUE(updated_by) WHERE id = ", "pos": "1:46"},
    {"op": "EMIT_EVAL", "param": "user_id", "pos": "1:100"}
  ]
}
```

#### Automatic UPDATE Statement Modification

For UPDATE statements, system fields corresponding to implicit parameters are automatically added to the SET clause:

**Original SQL:**
```sql
UPDATE users SET name = 'John', email = 'john@example.com' WHERE id = 1
```

**After system field addition:**
```sql
UPDATE users SET 
  name = 'John', 
  email = 'john@example.com',
  EMIT_SYSTEM_VALUE(updated_at),
  EMIT_SYSTEM_VALUE(updated_by)
WHERE id = 1
```

### Validation

Based on system field configuration, the following validations are performed:

1. **Explicit Parameter Existence Check**: Verify that parameters corresponding to `explicit` configured fields are provided
2. **Error Parameter Detection**: Check that parameters corresponding to `error` configured fields are not provided
3. **Type Consistency Check**: Verify that parameter types match system field definitions

### Runtime Behavior

#### Implicit Parameter Resolution

Runtime libraries automatically resolve implicit parameters from the following sources:

- **User Context**: Authenticated user ID (`created_by`, `updated_by`)
- **System Time**: Current time (default values for `created_at`, `updated_at`)
- **Session Information**: Request metadata
- **Configuration Values**: Default values defined in configuration file

#### Optimistic Locking

Support for optimistic locking using `lock_no` field:

1. **On SELECT**: Retrieve current `lock_no`
2. **On UPDATE**: Provide `lock_no` as explicit parameter
3. **At Runtime**: Raise optimistic lock exception if `lock_no` doesn't match

### Benefits

1. **Consistency**: Unified system field management across all tables
2. **Automation**: No manual system field configuration required
3. **Security**: Tamper prevention (prohibit updates to `created_at`, `created_by`)
4. **Auditing**: Automatic recording of complete change history
5. **Concurrency Control**: Data integrity guarantee through optimistic locking
