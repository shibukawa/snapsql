# Intermediate Instruction Set Design Document

**Date**: 2025-06-26  
**Author**: Development Team  
**Status**: Draft  

## Overview

This document outlines the design for a procedural instruction set that replaces the hierarchical AST in intermediate files. The system uses a **two-tier architecture**:

1. **Low-Level**: Instruction set interpreter for direct execution
2. **High-Level**: Generated programming language code that maps instructions to control structures and function calls

### Multi-Language Runtime Architecture

SnapSQL supports multiple programming languages with different implementation strategies based on language characteristics and use cases:

#### Go Runtime: Dual-Tier Architecture
**Low-Level Interpreter + High-Level Code Generation**

Go provides both execution modes to support different scenarios:

1. **Low-Level Interpreter** (`runtime/snapsqlgo`)
   - Direct instruction execution
   - Immediate SQL generation
   - **Primary Use Case**: REPL and interactive development
   - **Benefits**: No compilation step, instant feedback
   - **Performance**: Suitable for development and prototyping

2. **High-Level Code Generation** (planned)
   - Compile-time code generation
   - Type-safe function generation
   - **Primary Use Case**: Production applications
   - **Benefits**: Maximum performance, type safety, IDE support
   - **Performance**: Optimized for production workloads

#### Other Languages: High-Level Only
**Python, Node.js, Java, C#, etc.**

Other languages implement only high-level code generation:

1. **Code Generation Only**
   - Instructions compiled to native language constructs
   - Type-safe APIs where applicable
   - **Rationale**: Most languages don't need REPL support for SQL
   - **Benefits**: Simpler implementation, better performance
   - **Focus**: Production-ready, optimized code

### Architecture Comparison

| Language | Low-Level | High-Level | Primary Use Case | REPL Support |
|----------|-----------|------------|------------------|--------------|
| **Go** | ✅ | ✅ | Development + Production | ✅ |
| **Python** | ❌ | ✅ | Production | ❌ |
| **Node.js** | ❌ | ✅ | Production | ❌ |
| **Java** | ❌ | ✅ | Production | ❌ |
| **C#** | ❌ | ✅ | Production | ❌ |

### Go REPL Requirements

The Go runtime includes a low-level interpreter specifically to support REPL functionality:

#### REPL Use Cases
```go
// Interactive SQL development
> snapsql repl
SnapSQL> load template users.snap.sql
SnapSQL> set param user_id 123
SnapSQL> set param include_email true
SnapSQL> execute
SQL: SELECT id, name, email FROM users WHERE id = ?
Args: [123]

SnapSQL> set param include_email false
SnapSQL> execute
SQL: SELECT id, name FROM users WHERE id = ?
Args: [123]
```

#### Development Workflow
1. **Template Development**: Immediate feedback on SQL generation
2. **Parameter Testing**: Quick parameter value changes
3. **Debugging**: Step-by-step instruction execution
4. **Prototyping**: Rapid iteration without compilation

#### Implementation Benefits
- **Instant Feedback**: No compilation delay
- **Interactive Development**: Real-time SQL preview
- **Debugging Support**: Instruction-level debugging
- **Educational Tool**: Understanding SQL generation process

## Goals

### Primary Goals
- Convert hierarchical AST to linear instruction sequence
- Transform control structures (if/for) into conditional jumps and gotos
- Enable simple interpreter-based SQL generation at runtime
- Include fine-grained control for SQL formatting (commas, parentheses, etc.)
- Minimize runtime complexity and maximize performance

### Secondary Goals
- Maintain readability and debuggability of instruction sequences
- Support all existing SnapSQL template features
- Enable easy optimization of instruction sequences
- Provide clear mapping from original template to instructions

## Instruction Set Architecture

### Core Concepts

#### 1. Linear Execution Model
- Instructions are executed sequentially from index 0
- Control flow is managed through jump instructions
- No nested structures or recursive evaluation
- Simple program counter (PC) based execution

#### 2. Direct Parameter Access with Loop Variable Scoping
- Parameters accessed directly by name from input map
- Loop variables created within loop scope and removed when loop exits
- Simple boolean evaluation for conditions
- Direct parameter substitution in SQL output
- Hierarchical variable resolution (loop variables shadow parameters)

#### 3. SQL Output Buffer
- Instructions write directly to SQL output buffer
- Automatic formatting control (spaces, commas, parentheses)
- Conditional output based on runtime parameter values

### Instruction Types

#### 1. Output Instructions
Generate SQL text and manage formatting.

```json
{
  "op": "EMIT_LITERAL",
  "value": "SELECT id, name"
}
```

```json
{
  "op": "EMIT_PARAM",
  "param": "user_id",
  "placeholder": "123"
}
```

```json
{
  "op": "EMIT_EVAL",
  "exp": "user.age + 1",
  "placeholder": "25"
}
```

```json
{
  "op": "EMIT_LITERAL", "value": ",",
  "condition": "not_last_field"
}
```

```json
{
  "op": "EMIT_LITERAL", "value": " "
}
```

```json
{
  "op": "EMIT_LITERAL", "value": "\n"
}
```

#### 2. Control Flow Instructions
Manage program execution flow.

```json
{
  "op": "JUMP",
  "target": 15
}
```

```json
{
  "op": "JUMP_IF_TRUE",
  "condition": "include_email",
  "target": 10
}
```

```json
{
  "op": "JUMP_IF_FALSE",
  "condition": "filters.active",
  "target": 20
}
```

```json
{
  "op": "LABEL",
  "name": "end_select_fields"
}
```

#### 3. Condition Evaluation Instructions
Handle CEL expression evaluation for conditional jumps.

```json
{
  "op": "JUMP_IF_EXP",
  "exp": "!include_email",
  "target": 10
}
```

```json
{
  "op": "JUMP_IF_EXP",
  "exp": "!include_email",
  "target": 10
}
```

#### 4. Loop Instructions
Handle iteration over collections with proper variable scoping.

```json
{
  "op": "LOOP_START",
  "variable": "field",
  "collection": "additional_fields",
  "end_label": "end_field_loop"
}
```

```json
{
  "op": "LOOP_NEXT",
  "start_label": "field_loop_start"
}
```

```json
{
  "op": "LOOP_END",
  "variable": "field",
  "label": "end_field_loop"
}
```

#### 5. State Management Instructions
**REMOVED**: State management through flags has been removed. All conditional logic is handled through CEL expressions in `JUMP_IF_EXP` instructions.

```json
{
  "op": "JUMP_IF_FLAG_TRUE",
  "flag": "has_where_clause",
  "target": 30
}
```

```json
{
  "op": "JUMP_IF_FLAG_FALSE",
  "flag": "has_where_clause",
  "target": 35
}
```

## Instruction Set Reference

### Output Instructions

| Instruction | Description | Parameters |
|-------------|-------------|------------|
| `EMIT_LITERAL` | Output literal SQL text | `value`: string |
| `EMIT_PARAM` | Output single variable placeholder | `param`: variable name, `placeholder`: dummy value |
| `EMIT_EVAL` | Output CEL expression result placeholder | `exp`: CEL expression, `placeholder`: dummy value |
| `EMIT_LPAREN` | Output left parenthesis | none |
| `EMIT_RPAREN` | Output right parenthesis | none |

### Control Flow Instructions

| Instruction | Description | Parameters |
|-------------|-------------|------------|
| `JUMP` | Unconditional jump | `target`: instruction index |
| `JUMP_IF_EXP` | Jump if CEL expression is true | `exp`: CEL expression, `target`: instruction index |
| `JUMP_IF_FLAG_TRUE` | Jump if flag is true | `flag`: flag name, `target`: instruction index |
| `JUMP_IF_FLAG_FALSE` | Jump if flag is false | `flag`: flag name, `target`: instruction index |
| `LABEL` | Define jump target | `name`: label name |
| `NOP` | No operation | none |

### CEL Expression Instructions

| Instruction | Description | Parameters |
|-------------|-------------|------------|
| `JUMP_IF_EXP` | Jump if CEL expression is truthy | `exp`: CEL expression, `target`: instruction index |

### Loop Instructions

| Instruction | Description | Parameters |
|-------------|-------------|------------|
| `LOOP_START` | Initialize loop over collection | `variable`: loop var, `collection`: CEL expression for collection, `end_label`: end label |
| `LOOP_NEXT` | Continue to next iteration | `start_label`: loop start label |
| `LOOP_END` | End loop and cleanup variable | `variable`: loop var to remove, `label`: loop end label |

### EMIT Instruction Usage Examples

#### Simple Variable Output
```sql
-- Template: /*= user_id */123
{"op": "EMIT_PARAM", "param": "user_id", "placeholder": "123"}

-- Template: /*= field */field_name  
{"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"}
```

#### Complex Expression Output
```sql
-- Template: /*= user.age + 1 */25
{"op": "EMIT_EVAL", "exp": "user.age + 1", "placeholder": "25"}

-- Template: /*= table.name */table_name
{"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"}

-- Template: /*= len(items) */5
{"op": "EMIT_EVAL", "exp": "len(items)", "placeholder": "5"}
```

#### Literal Output (Spaces, Commas, Newlines)
```sql
-- All formatting is handled by EMIT_LITERAL
{"op": "EMIT_LITERAL", "value": ", "}      // Comma with space
{"op": "EMIT_LITERAL", "value": "\n"}      // Newline
{"op": "EMIT_LITERAL", "value": "  "}      // Indentation
{"op": "EMIT_LITERAL", "value": " AND "}   // SQL keywords
```

#### Performance Considerations
- `EMIT_LITERAL`: Fastest, direct string output
- `EMIT_PARAM`: Fast direct variable lookup, no CEL engine overhead
- `EMIT_EVAL`: Full CEL expression evaluation, more flexible but slower

## Example Transformations

### Simple Conditional Field

**Original Template:**
```sql
SELECT id, name
/*# if include_email */
, email
/*# end */
FROM users
```

**Generated Instructions:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT id, name"},
  {"op": "JUMP_IF_EXP", "exp": "!include_email", "target": 5},
  {"op": "EMIT_LITERAL", "value": ", email"},
  {"op": "LABEL", "name": "end_email_field"},
  {"op": "EMIT_LITERAL", "value": " FROM users"}
]
```

### Loop with Comma Control

**Original Template:**
```sql
SELECT 
/*# for field : additional_fields */
    /*= field */,
/*# end */
    created_at
FROM users
```

**Generated Instructions:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_START", "variable": "field", "collection": "additional_fields", "end_label": "end_field_loop"},
  {"op": "LABEL", "name": "field_loop_start"},
  {"op": "EMIT_LITERAL", "value": "    "},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": ","},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_NEXT", "start_label": "field_loop_start"},
  {"op": "LOOP_END", "variable": "field", "label": "end_field_loop"},
  {"op": "EMIT_LITERAL", "value": "    created_at"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "EMIT_LITERAL", "value": "FROM users"}
]
```

### Nested Loops with Variable Scoping

**Original Template:**
```sql
SELECT 
/*# for table : tables */
  /*# for field : table.fields */
    /*= table.name */./*= field */ AS /*= table.name */_/*= field */,
  /*# end */
/*# end */
  1 as dummy
FROM dual
```

**Generated Instructions:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_START", "variable": "table", "collection": "tables", "end_label": "end_table_loop"},
  {"op": "LABEL", "name": "table_loop_start"},
  {"op": "LOOP_START", "variable": "field", "collection": "table.fields", "end_label": "end_field_loop"},
  {"op": "LABEL", "name": "field_loop_start"},
  {"op": "EMIT_LITERAL", "value": "  "},
  {"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"},
  {"op": "EMIT_LITERAL", "value": "."},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": " AS "},
  {"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"},
  {"op": "EMIT_LITERAL", "value": "_"},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": ","},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_NEXT", "start_label": "field_loop_start"},
  {"op": "LOOP_END", "variable": "field", "label": "end_field_loop"},
  {"op": "LOOP_NEXT", "start_label": "table_loop_start"},
  {"op": "LOOP_END", "variable": "table", "label": "end_table_loop"},
  {"op": "EMIT_LITERAL", "value": "  1 as dummy"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "EMIT_LITERAL", "value": "FROM dual"}
]
```

**Variable Resolution During Execution:**
1. **Outside loops**: Only input parameters are visible
2. **Inside table loop**: `table` variable shadows any input parameter named `table`
3. **Inside nested field loop**: Both `table` and `field` variables are visible
4. **After field loop ends**: `field` variable is removed, only `table` remains visible
5. **After table loop ends**: `table` variable is removed, back to input parameters only

### Complex Conditional with WHERE Clause

**Original Template:**
```sql
SELECT * FROM users
/*# if filters.active || filters.department */
WHERE 1=1
  /*# if filters.active */AND active = /*= filters.active */true/*# end */
  /*# if filters.department */AND department = /*= filters.department */'sales'/*# end */
/*# end */
```

**Generated Instructions:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT * FROM users"},
  
  {"op": "JUMP_IF_EXP", "exp": "!(filters.active || filters.department)", "target": 8},
  {"op": "EMIT_LITERAL", "value": " WHERE 1=1"},
  
  {"op": "JUMP_IF_EXP", "exp": "!filters.active", "target": 6},
  {"op": "EMIT_LITERAL", "value": " AND active = "},
  {"op": "EMIT_EVAL", "exp": "filters.active", "placeholder": "true"},
  
  {"op": "JUMP_IF_EXP", "exp": "!filters.department", "target": 8},
  {"op": "EMIT_LITERAL", "value": " AND department = "},
  {"op": "EMIT_EVAL", "exp": "filters.department", "placeholder": "'sales'"},
  
  {"op": "LABEL", "name": "end_where_clause"}
]
```

## Runtime Execution Model

### Execution Engine

```go
type InstructionExecutor struct {
    instructions []Instruction
    pc          int
    params      map[string]any
    output      strings.Builder
    loops       []LoopState
    variables   map[string]any  // Loop variables with scoping
}

type LoopState struct {
    Variable   string
    Collection []any
    Index      int
    StartPC    int
}

func (e *InstructionExecutor) Execute() (string, []any, error) {
    e.variables = make(map[string]any)
    for e.pc < len(e.instructions) {
        inst := e.instructions[e.pc]
        if err := e.executeInstruction(inst); err != nil {
            return "", nil, err
        }
        e.pc++
    }
    return e.output.String(), e.extractParameters(), nil
}
```

### Variable Resolution and Loop Scoping

```go
func (e *InstructionExecutor) addParameter(paramName string) {
    // Add simple variable value to parameter list
    value := e.getVariableValue(paramName)
    e.parameters = append(e.parameters, value)
}

func (e *InstructionExecutor) addExpression(celExpression string) {
    // Evaluate CEL expression and add result to parameter list
    value := e.evaluateCELExpression(celExpression)
    e.parameters = append(e.parameters, value)
}

func (e *InstructionExecutor) getVariableValue(name string) any {
    // Check loop variables first (they shadow parameters)
    if value, exists := e.variables[name]; exists {
        return value
    }
    // Fall back to input parameters
    return e.getParamValue(name)
}

func (e *InstructionExecutor) executeInstruction(inst Instruction) error {
    switch inst.Op {
    case "EMIT_LITERAL":
        e.output.WriteString(inst.Value)
    case "EMIT_PARAM":
        e.output.WriteString("?")
        e.addParameter(inst.Param)
    case "EMIT_EVAL":
        e.output.WriteString("?")
        e.addExpression(inst.Exp)
    case "JUMP":
        e.pc = inst.Target - 1
    case "JUMP_IF_EXP":
        if e.isExpressionTruthy(inst.Exp) {
            e.pc = inst.Target - 1
        }
    case "LOOP_START":
        collection := e.evaluateCELExpression(inst.Collection)
        if collectionSlice, ok := collection.([]any); ok && len(collectionSlice) > 0 {
            // Set loop variable to first element
            e.variables[inst.Variable] = collectionSlice[0]
            // Push loop state
            e.loops = append(e.loops, LoopState{
                Variable:   inst.Variable,
                Collection: collectionSlice,
                Index:      0,
                StartPC:    e.pc + 1,
            })
        } else {
            // Empty collection, jump to end
            e.pc = e.findLabel(inst.EndLabel) - 1
        }
    case "LOOP_NEXT":
        if len(e.loops) > 0 {
            loop := &e.loops[len(e.loops)-1]
            loop.Index++
            if loop.Index < len(loop.Collection) {
                // Update loop variable and jump back
                e.variables[loop.Variable] = loop.Collection[loop.Index]
                e.pc = loop.StartPC - 1
            }
            // Otherwise continue to LOOP_END
        }
    case "LOOP_END":
        if len(e.loops) > 0 {
            // Remove loop variable from scope
            delete(e.variables, inst.Variable)
            // Pop loop state
            e.loops = e.loops[:len(e.loops)-1]
        }
    case "LABEL":
        // No operation, just a marker
    case "NOP":
        // No operation
    }
    return nil
}

func (e *InstructionExecutor) isExpressionTruthy(celExpression string) bool {
    // Evaluate CEL expression with current variable context
    value := e.evaluateCELExpression(celExpression)
    // Handle Go truthiness: false, 0, "", nil, empty slices/maps are falsy
    switch v := value.(type) {
    case bool:
        return v
    case int, int8, int16, int32, int64:
        return v != 0
    case uint, uint8, uint16, uint32, uint64:
        return v != 0
    case float32, float64:
        return v != 0
    case string:
        return v != ""
    case []any:
        return len(v) > 0
    case map[string]any:
        return len(v) > 0
    case nil:
        return false
    default:
        return true
    }
}

func (e *InstructionExecutor) evaluateCELExpression(expression string) any {
    // Create CEL environment with current variables and input parameters
    env := e.createCELEnvironment()
    
    // Parse and evaluate CEL expression
    ast, issues := env.Parse(expression)
    if issues != nil && issues.Err() != nil {
        return nil
    }
    
    prg, err := env.Program(ast)
    if err != nil {
        return nil
    }
    
    // Create evaluation context with variables and parameters
    vars := make(map[string]any)
    for k, v := range e.variables {
        vars[k] = v
    }
    for k, v := range e.params {
        vars[k] = v
    }
    
    out, _, err := prg.Eval(vars)
    if err != nil {
        return nil
    }
    
    return out.Value()
}
```

## Intermediate File Format

### Updated Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-instruction-format.json",
  "title": "SnapSQL Intermediate Instruction Format",
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "properties": {
        "file": {"type": "string"},
        "content": {"type": "string"}
      },
      "required": ["file", "content"]
    },
    "interface_schema": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "function_name": {"type": "string"},
        "parameters": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "type": {"type": "string"},
              "required": {"type": "boolean"},
              "default": {}
            },
            "required": ["name", "type"]
          }
        }
      }
    },
    "instructions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "op": {"type": "string"},
          "value": {},
          "param": {"type": "string"},
          "exp": {"type": "string"},
          "placeholder": {"type": "string"},
          "condition": {"type": "string"},
          "target": {"type": "integer"},
          "label": {"type": "string"},
          "variable": {"type": "string"},
          "collection": {"type": "string"},
          "end_label": {"type": "string"},
          "start_label": {"type": "string"},
          "flag": {"type": "string"},
          "state": {"type": "string"}
        },
        "required": ["op"]
      }
    },
    "labels": {
      "type": "object",
      "additionalProperties": {"type": "integer"}
    },
    "metadata": {
      "type": "object",
      "properties": {
        "version": {"type": "string"},
        "generated_at": {"type": "string"},
        "template_hash": {"type": "string"}
      }
    }
  },
  "required": ["source", "instructions"]
}
```

### Example Intermediate File

```json
{
  "source": {
    "file": "/path/to/users.snap.sql",
    "content": "SELECT id, name /*# if include_email */, email /*# end */ FROM users"
  },
  "interface_schema": {
    "name": "user_query",
    "function_name": "getUsers",
    "parameters": [
      {
        "name": "include_email",
        "type": "bool",
        "required": false,
        "default": false
      }
    ]
  },
  "instructions": [
    {"op": "EMIT_LITERAL", "value": "SELECT id, name"},
    {"op": "JUMP_IF_EXP", "exp": "!include_email", "target": 5},
    {"op": "EMIT_LITERAL", "value": ", email"},
    {"op": "LABEL", "name": "end_email_field"},
    {"op": "EMIT_LITERAL", "value": " FROM users"}
  ],
  "labels": {
    "end_email_field": 5
  },
  "metadata": {
    "version": "1.0.0",
    "generated_at": "2025-06-26T01:00:00Z",
    "template_hash": "sha256:abc123..."
  }
}
```

## Final Instruction Set

| Category | Instruction | Description | Parameters |
|----------|-------------|-------------|------------|
| **Output** | `EMIT_LITERAL` | Output literal string | `value`: string |
| **Output** | `EMIT_PARAM` | Output single variable placeholder | `param`: variable name, `placeholder`: dummy value |
| **Output** | `EMIT_EVAL` | Output CEL expression result placeholder | `exp`: CEL expression, `placeholder`: dummy value |
| **Control** | `JUMP` | Unconditional jump | `target`: instruction index |
| **Control** | `JUMP_IF_EXP` | Conditional jump | `exp`: CEL expression, `target`: instruction index |
| **Control** | `LABEL` | Jump target marker | `name`: label name |
| **Loop** | `LOOP_START` | Initialize loop over collection | `variable`: loop var, `collection`: CEL expression, `end_label`: end label |
| **Loop** | `LOOP_NEXT` | Continue to next iteration | `start_label`: loop start label |
| **Loop** | `LOOP_END` | End loop and cleanup variable | `variable`: loop var, `label`: loop end label |
| **Misc** | `NOP` | No operation | none |

**Total: 10 instructions** - A minimal, efficient instruction set for SQL template processing.

## Benefits of Instruction Set Approach

### Runtime Performance
- Linear execution with simple program counter
- No stack operations or complex value management
- Direct parameter access and evaluation
- Minimal branching and decision making
- Direct SQL output generation

### Simplicity
- Clear, debuggable instruction sequences
- Easy to implement interpreter
- No stack management complexity
- Straightforward parameter evaluation
- Reduced runtime complexity

### Flexibility
- Easy to add new instruction types
- Support for complex control flow
- Fine-grained formatting control
- Extensible for future features

### Debugging and Analysis
- Clear execution trace
- Easy to visualize program flow
- Simple breakpoint and stepping support
- Performance profiling capabilities

## Multi-Language High-Level Interface Design

### Go High-Level Interface (Planned)

#### Generated Go Code Example
```go
// Generated from SQL template metadata
type SearchUserResult struct {
    ID          string `db:"id"`
    Name        string `db:"name"`
    Org         string `db:"org"`
    PhoneNumber string `db:"phone_number"`
}

func SearchUsers[D DB](ctx context.Context, db D, id string, name string) iter.Seq2[SearchUserResult, error] {
    // Compiled instruction set to optimized Go code
    var sql strings.Builder
    var args []any
    
    sql.WriteString("SELECT id, name, org, phone_number FROM users WHERE id = ")
    args = append(args, strings.ToUpper(id)) // CEL: id.upper()
    sql.WriteString(" AND name = ")
    args = append(args, name)
    
    return executeQuery[SearchUserResult](ctx, db, sql.String(), args)
}
```

### Python High-Level Interface (Planned)

#### Generated Python Code Example
```python
from typing import Iterator, NamedTuple
from dataclasses import dataclass

@dataclass
class SearchUserResult:
    id: str
    name: str
    org: str
    phone_number: str

def search_users(db: Connection, id: str, name: str) -> Iterator[SearchUserResult]:
    """Generated from searchuser.snap.sql"""
    sql = "SELECT id, name, org, phone_number FROM users WHERE id = %s AND name = %s"
    args = [id.upper(), name]  # CEL expressions compiled to Python
    
    cursor = db.execute(sql, args)
    for row in cursor:
        yield SearchUserResult(*row)
```

### Node.js High-Level Interface (Planned)

#### Generated TypeScript Code Example
```typescript
interface SearchUserResult {
    id: string;
    name: string;
    org: string;
    phoneNumber: string;
}

export async function* searchUsers(
    db: Database, 
    id: string, 
    name: string
): AsyncIterableIterator<SearchUserResult> {
    const sql = "SELECT id, name, org, phone_number FROM users WHERE id = ? AND name = ?";
    const args = [id.toUpperCase(), name]; // CEL expressions compiled to JS
    
    const rows = await db.query(sql, args);
    for (const row of rows) {
        yield {
            id: row.id,
            name: row.name,
            org: row.org,
            phoneNumber: row.phone_number
        };
    }
}
```

### Java High-Level Interface (Planned)

#### Generated Java Code Example
```java
public record SearchUserResult(String id, String name, String org, String phoneNumber) {}

public class SearchUserQuery {
    public static Stream<SearchUserResult> searchUsers(
            Connection db, String id, String name) throws SQLException {
        
        String sql = "SELECT id, name, org, phone_number FROM users WHERE id = ? AND name = ?";
        Object[] args = {id.toUpperCase(), name}; // CEL expressions compiled to Java
        
        PreparedStatement stmt = db.prepareStatement(sql);
        for (int i = 0; i < args.length; i++) {
            stmt.setObject(i + 1, args[i]);
        }
        
        ResultSet rs = stmt.executeQuery();
        return StreamSupport.stream(new ResultSetSpliterator<>(rs, row -> 
            new SearchUserResult(
                row.getString("id"),
                row.getString("name"), 
                row.getString("org"),
                row.getString("phone_number")
            )), false);
    }
}
```

### Language-Specific Features

#### Go Features
- **Generics**: Type-safe database interfaces
- **Iterators**: Go 1.23+ range-over-func support
- **Context**: Built-in cancellation and timeout
- **REPL Support**: Interactive development with low-level interpreter

#### Python Features
- **Dataclasses**: Automatic result type generation
- **Type Hints**: Full typing support
- **Async/Await**: Asynchronous database operations
- **Context Managers**: Resource management

#### Node.js/TypeScript Features
- **Async Iterators**: Streaming result processing
- **Type Safety**: Full TypeScript support
- **Promise-based**: Modern async patterns
- **ESM/CommonJS**: Module system compatibility

#### Java Features
- **Records**: Immutable result types
- **Streams**: Functional result processing
- **JDBC Integration**: Standard database connectivity
- **Annotation Processing**: Compile-time code generation

### Instruction-to-Code Mapping by Language

| Instruction | Go | Python | Node.js | Java |
|-------------|----|---------|---------|----- |
| `EMIT_LITERAL` | `sql.WriteString("text")` | `sql += "text"` | `sql += "text"` | `sql.append("text")` |
| `EMIT_PARAM` | `args = append(args, var)` | `args.append(var)` | `args.push(var)` | `stmt.setObject(i, var)` |
| `EMIT_EVAL` | `args = append(args, expr())` | `args.append(eval_expr())` | `args.push(evalExpr())` | `stmt.setObject(i, evalExpr())` |
| `JUMP_IF_EXP` | `if condition { ... }` | `if condition:` | `if (condition) {` | `if (condition) {` |
| `LOOP_START` | `for item := range coll {` | `for item in coll:` | `for (const item of coll) {` | `for (var item : coll) {` |

### Benefits by Implementation Strategy

#### Go (Dual Implementation)
- **Development**: Fast iteration with interpreter
- **Production**: Maximum performance with generated code
- **Flexibility**: Choose execution mode based on use case
- **REPL**: Interactive SQL development and debugging

#### Other Languages (High-Level Only)
- **Simplicity**: Single implementation path
- **Performance**: Optimized generated code
- **Maintainability**: Fewer moving parts
- **Focus**: Production-ready applications

## Implementation Plan

### Phase 1: Instruction Set Definition ✅
1. Define instruction format and JSON schema ✅
2. Create instruction set documentation ✅
3. Design multi-language architecture ✅

### Phase 2: Go Low-Level Implementation ✅
1. Implement instruction interpreter ✅
2. Create AST-to-instruction compiler ✅
3. Add CEL expression evaluation ✅
4. Implement loop variable scoping ✅
5. Package organization (`runtime/snapsqlgo`) ✅

### Phase 3: Multi-Language High-Level Implementation (Planned)
1. **Go High-Level Generator**
   - Design type-safe function generation
   - Implement prepared statement caching
   - Add iterator-based result handling
   - Integrate with low-level interpreter for REPL

2. **Python Code Generator**
   - Generate dataclass-based result types
   - Implement async/await patterns
   - Add type hint support
   - Create pip-installable package

3. **Node.js/TypeScript Generator**
   - Generate TypeScript interfaces
   - Implement async iterator patterns
   - Add ESM/CommonJS support
   - Create npm package

4. **Java Code Generator**
   - Generate record-based result types
   - Implement Stream-based processing
   - Add annotation processing
   - Create Maven/Gradle artifacts

### Phase 4: REPL and Tooling (Go-specific)
1. **Interactive REPL**
   - Command-line interface
   - Template loading and management
   - Parameter manipulation
   - Real-time SQL preview

2. **Development Tools**
   - Template validation
   - Performance profiling
   - Debugging support
   - IDE integration

### Phase 5: Integration and Testing
1. Integrate with existing SnapSQL pipeline
2. Add comprehensive test coverage for all languages
3. Performance benchmarking across languages
4. Documentation and examples for each language

## Architecture Benefits

### Go Dual Implementation
- **Development Flexibility**: Choose between interpreter (fast iteration) and generated code (performance)
- **REPL Support**: Interactive SQL development and debugging
- **Production Ready**: Optimized generated code for production workloads
- **Educational Value**: Understanding SQL generation through instruction-level debugging

### Multi-Language Consistency
- **Unified Instruction Set**: Same intermediate format across all languages
- **Language-Specific Optimization**: Each language uses its strengths
- **Maintainable Codebase**: Clear separation between instruction logic and language-specific generation
- **Extensible Design**: Easy to add new target languages

## Open Questions

1. **Type Inference**: How should type information be propagated from SQL to generated code?
2. **Error Handling**: What error handling patterns should be used in each language?
3. **Performance**: What are the performance characteristics of interpreter vs generated code?
4. **Debugging**: How should debugging information be embedded across languages?
5. **Packaging**: What are the distribution strategies for each language runtime?

## References

- [SnapSQL README](../README.md)
- [Go Runtime Design](./20250625-go-runtime.md)
- [Coding Standards](../coding-standard.md)
