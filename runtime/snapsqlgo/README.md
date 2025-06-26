# SnapSQL Go Runtime

This package provides the low-level instruction set interpreter for SnapSQL's Go runtime.

## Overview

The SnapSQL Go runtime implements a minimal instruction set that converts SQL templates into executable SQL queries with parameter binding. It uses a 10-instruction virtual machine that processes linear instruction sequences compiled from hierarchical ASTs.

## Instruction Set

| Instruction | Description | Parameters |
|-------------|-------------|------------|
| `EMIT_LITERAL` | Output literal string | `value`: string |
| `EMIT_PARAM` | Output single variable placeholder | `param`: variable name, `placeholder`: dummy value |
| `EMIT_EVAL` | Output CEL expression result placeholder | `exp`: CEL expression, `placeholder`: dummy value |
| `JUMP` | Unconditional jump | `target`: instruction index |
| `JUMP_IF_EXP` | Conditional jump | `exp`: CEL expression, `target`: instruction index |
| `LABEL` | Jump target marker | `name`: label name |
| `LOOP_START` | Initialize loop over collection | `variable`: loop var, `collection`: CEL expression, `end_label`: end label |
| `LOOP_NEXT` | Continue to next iteration | `start_label`: loop start label |
| `LOOP_END` | End loop and cleanup variable | `variable`: loop var, `label`: loop end label |
| `NOP` | No operation | none |

## Features

### CEL Expression Evaluation
- Support for complex logical expressions (`&&`, `||`, `!`)
- Dot notation for nested property access (`user.profile.name`)
- Automatic type conversion and truthiness evaluation

### Variable Scoping
- Loop variables shadow input parameters
- Automatic cleanup when loops end
- Support for nested loops with proper scoping

### Dynamic SQL Generation
- Conditional WHERE clauses
- Dynamic field selection
- Loop-based SQL construction
- Parameter binding with type safety

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/shibukawa/snapsql/runtime/snapsqlgo"
)

func main() {
    instructions := []snapsqlgo.Instruction{
        {Op: "EMIT_LITERAL", Value: "SELECT id, name FROM users WHERE id = "},
        {Op: "EMIT_PARAM", Param: "user_id", Placeholder: "123"},
    }

    params := map[string]any{
        "user_id": 42,
    }

    executor, err := snapsqlgo.NewInstructionExecutor(instructions, params)
    if err != nil {
        panic(err)
    }

    sql, args, err := executor.Execute()
    if err != nil {
        panic(err)
    }

    fmt.Printf("SQL: %s\n", sql)     // SELECT id, name FROM users WHERE id = ?
    fmt.Printf("Args: %v\n", args)   // [42]
}
```

### Conditional SQL Generation

```go
instructions := []snapsqlgo.Instruction{
    {Op: "EMIT_LITERAL", Value: "SELECT id, name"},
    {Op: "JUMP_IF_EXP", Exp: "!include_email", Target: 4},
    {Op: "EMIT_LITERAL", Value: ", email"},
    {Op: "LABEL", Name: "end_email"},
    {Op: "EMIT_LITERAL", Value: " FROM users"},
}

params := map[string]any{
    "include_email": true,
}

// Result: "SELECT id, name, email FROM users"
```

### Loop-based SQL Generation

```go
instructions := []snapsqlgo.Instruction{
    {Op: "EMIT_LITERAL", Value: "SELECT "},
    {Op: "LOOP_START", Variable: "field", Collection: "fields", EndLabel: "end_loop"},
    {Op: "EMIT_PARAM", Param: "field", Placeholder: "field_name"},
    {Op: "EMIT_LITERAL", Value: ", "},
    {Op: "LOOP_NEXT", StartLabel: "loop_start"},
    {Op: "LOOP_END", Variable: "field", Label: "end_loop"},
    {Op: "EMIT_LITERAL", Value: "1 FROM users"},
}

params := map[string]any{
    "fields": []any{"id", "name", "email"},
}

// Result: "SELECT ?, ?, ?, 1 FROM users" with args ["id", "name", "email"]
```

### Complex Conditional Logic

```go
instructions := []snapsqlgo.Instruction{
    {Op: "EMIT_LITERAL", Value: "SELECT * FROM users"},
    {Op: "JUMP_IF_EXP", Exp: "!(filters.active || filters.department)", Target: 9},
    {Op: "EMIT_LITERAL", Value: " WHERE 1=1"},
    {Op: "JUMP_IF_EXP", Exp: "!filters.active", Target: 6},
    {Op: "EMIT_LITERAL", Value: " AND active = "},
    {Op: "EMIT_EVAL", Exp: "filters.active", Placeholder: "true"},
    {Op: "JUMP_IF_EXP", Exp: "!filters.department", Target: 9},
    {Op: "EMIT_LITERAL", Value: " AND department = "},
    {Op: "EMIT_EVAL", Exp: "filters.department", Placeholder: "'sales'"},
    {Op: "LABEL", Name: "end_where"},
}

params := map[string]any{
    "filters": map[string]any{
        "active":     true,
        "department": "engineering",
    },
}

// Result: "SELECT * FROM users WHERE 1=1 AND active = ? AND department = ?"
// Args: [true, "engineering"]
```

## Architecture

### Instruction Executor
The `InstructionExecutor` processes instruction sequences linearly with support for:
- Program counter (PC) based execution
- Jump instructions for control flow
- Variable scoping with shadow support
- CEL expression evaluation
- Parameter collection for SQL binding

### CEL Integration
The runtime integrates with Google's Common Expression Language (CEL) for:
- Complex boolean expressions
- Arithmetic operations
- String manipulation
- Type conversion

### Memory Management
- Minimal memory footprint
- Automatic cleanup of loop variables
- Efficient parameter collection
- No garbage collection pressure

## Testing

The package includes comprehensive tests covering:
- Basic instruction execution
- Conditional jumps and control flow
- Loop processing (normal and empty collections)
- Variable scoping and shadowing
- CEL expression evaluation
- Complex nested conditions

Run tests with:
```bash
go test -v
```

## Performance

The instruction set is designed for optimal performance:
- **10 instructions total** - minimal instruction set
- **Linear execution** - predictable performance
- **Direct parameter access** - no stack overhead
- **Efficient jumps** - O(1) jump operations
- **Minimal allocations** - reuse of data structures

## Future Enhancements

This low-level runtime serves as the foundation for:
- High-level code generation (type-safe Go functions)
- Prepared statement caching
- Query optimization
- Performance profiling
- Integration with database drivers

## License

This runtime library is licensed under Apache-2.0, allowing flexible use in various projects.
