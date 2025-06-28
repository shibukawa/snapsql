# Type Inference System for SELECT Clause Fields

## Overview

This document describes the design of a type inference system that analyzes SELECT clause fields and determines their data types based on database schema information. The system will provide accurate type information for code generation and runtime type safety.

## Problem Statement

When generating code from SQL templates, we need to know the exact data types of each field in the SELECT clause to:

1. Generate strongly-typed data structures in target languages
2. Provide accurate type information for IDE support and IntelliSense
3. Enable compile-time type checking
4. Generate appropriate serialization/deserialization code
5. Validate parameter types against expected column types

## Requirements

### Functional Requirements

1. **Column Type Resolution**: Resolve types for direct column references (`table.column`, `column`)
2. **Expression Type Inference**: Infer types for expressions, functions, and calculations
3. **Alias Handling**: Track type information through column aliases
4. **Join Support**: Handle type resolution across multiple tables in JOINs
5. **Aggregate Function Support**: Determine return types of aggregate functions
6. **Built-in Function Support**: Support common SQL functions and their return types
7. **Subquery Support**: Handle type inference for subqueries and CTEs
8. **CASE Expression Support**: Infer types for conditional expressions
9. **Multi-Database Support**: Support PostgreSQL, MySQL, and SQLite type systems

### Non-Functional Requirements

1. **Performance**: Fast type resolution for large schemas
2. **Accuracy**: High precision in type inference with minimal false positives
3. **Extensibility**: Easy to add support for new functions and data types
4. **Error Handling**: Clear error messages for unresolvable types

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Schema Store  │    │ Type Resolver   │    │ Function Registry│
│                 │    │                 │    │                 │
│ - Table Info    │◄───┤ - Column Types  │◄───┤ - Built-in Funcs│
│ - Column Types  │    │ - Expression    │    │ - Aggregate     │
│ - Constraints   │    │   Analysis      │    │ - Cast Rules    │
└─────────────────┘    │ - Type Rules    │    └─────────────────┘
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │ Type Inference  │
                       │    Engine       │
                       │                 │
                       │ - AST Walker    │
                       │ - Context Stack │
                       │ - Error Handler │
                       └─────────────────┘
```

### Data Structures

#### Schema Information
```go
type SchemaStore struct {
    Tables    map[string]*TableInfo
    Views     map[string]*ViewInfo
    Functions map[string]*FunctionInfo
}

type TableInfo struct {
    Name        string
    Schema      string
    Columns     map[string]*ColumnInfo
    PrimaryKeys []string
    Indexes     map[string]*IndexInfo
}

type ColumnInfo struct {
    Name         string
    DataType     string
    IsNullable   bool
    DefaultValue *string
    MaxLength    *int
    Precision    *int
    Scale        *int
}
```

#### Type Information
```go
type TypeInfo struct {
    BaseType     string        // int, string, float, bool, time, etc.
    DatabaseType string        // varchar(255), int4, decimal(10,2), etc.
    IsNullable   bool
    IsArray      bool
    ArrayDims    int
    Precision    *int
    Scale        *int
    Constraints  []Constraint
}

type InferredField struct {
    Name         string
    Alias        string
    Type         *TypeInfo
    Source       FieldSource
    Dependencies []string      // Referenced columns/tables
}

type FieldSource struct {
    Type       SourceType    // Column, Expression, Function, Subquery
    Table      string
    Column     string
    Expression string
}
```

## Type Inference Algorithm

### Phase 1: Schema Analysis

1. **Load Database Schema**
   ```go
   func LoadSchema(db Database) (*SchemaStore, error) {
       // Query information_schema or equivalent
       // Build table and column mappings
       // Cache type information
   }
   ```

2. **Build Type Mappings**
   ```go
   func BuildTypeMappings(dialect string) map[string]string {
       // Map database-specific types to canonical types
       // Handle type variations across databases
   }
   ```

### Phase 2: AST Analysis

1. **Parse SELECT Clause**
   ```go
   func AnalyzeSelectClause(selectClause *SelectClause, context *InferenceContext) ([]*InferredField, error) {
       var fields []*InferredField
       
       for _, field := range selectClause.Fields {
           inferredField, err := InferFieldType(field, context)
           if err != nil {
               return nil, err
           }
           fields = append(fields, inferredField)
       }
       
       return fields, nil
   }
   ```

2. **Field Type Inference**
   ```go
   func InferFieldType(field *SelectField, context *InferenceContext) (*InferredField, error) {
       switch field.Type {
       case ColumnReference:
           return InferColumnType(field, context)
       case FunctionCall:
           return InferFunctionType(field, context)
       case Expression:
           return InferExpressionType(field, context)
       case Subquery:
           return InferSubqueryType(field, context)
       case CaseExpression:
           return InferCaseType(field, context)
       }
   }
   ```

### Phase 3: Type Resolution Rules

#### Column Reference Resolution
```go
func InferColumnType(field *SelectField, context *InferenceContext) (*InferredField, error) {
    // 1. Resolve table alias if present
    tableName := ResolveTableAlias(field.Table, context)
    
    // 2. Find column in schema
    column := context.Schema.FindColumn(tableName, field.Column)
    if column == nil {
        return nil, ErrColumnNotFound
    }
    
    // 3. Convert database type to canonical type
    typeInfo := ConvertDatabaseType(column.DataType, context.Dialect)
    
    return &InferredField{
        Name:   field.Column,
        Alias:  field.Alias,
        Type:   typeInfo,
        Source: FieldSource{Type: ColumnSource, Table: tableName, Column: field.Column},
    }, nil
}
```

#### Function Call Resolution
```go
func InferFunctionType(field *SelectField, context *InferenceContext) (*InferredField, error) {
    funcName := strings.ToUpper(field.Function.Name)
    
    // 1. Check built-in functions
    if funcDef, exists := context.Functions[funcName]; exists {
        return ResolveFunctionType(funcDef, field.Function.Args, context)
    }
    
    // 2. Check user-defined functions
    if userFunc, exists := context.Schema.Functions[funcName]; exists {
        return ResolveUserFunctionType(userFunc, field.Function.Args, context)
    }
    
    return nil, ErrUnknownFunction
}
```

#### Expression Type Resolution
```go
func InferExpressionType(field *SelectField, context *InferenceContext) (*InferredField, error) {
    switch field.Expression.Type {
    case BinaryOperation:
        return InferBinaryOpType(field.Expression, context)
    case UnaryOperation:
        return InferUnaryOpType(field.Expression, context)
    case Literal:
        return InferLiteralType(field.Expression, context)
    case Cast:
        return InferCastType(field.Expression, context)
    }
}
```

## Built-in Function Registry

### Aggregate Functions
```go
var AggregateFunctions = map[string]FunctionDef{
    "COUNT": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "int", IsNullable: false}
        },
    },
    "SUM": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            if len(args) == 0 {
                return TypeInfo{BaseType: "unknown"}
            }
            return PromoteNumericType(args[0])
        },
    },
    "AVG": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "decimal", IsNullable: true}
        },
    },
    "MAX": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            if len(args) == 0 {
                return TypeInfo{BaseType: "unknown"}
            }
            return args[0] // Same type as input
        },
    },
}
```

### String Functions
```go
var StringFunctions = map[string]FunctionDef{
    "CONCAT": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "string", IsNullable: AnyNullable(args)}
        },
    },
    "SUBSTRING": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "string", IsNullable: true}
        },
    },
    "LENGTH": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "int", IsNullable: false}
        },
    },
}
```

### Date/Time Functions
```go
var DateTimeFunctions = map[string]FunctionDef{
    "NOW": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "timestamp", IsNullable: false}
        },
    },
    "DATE_TRUNC": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "timestamp", IsNullable: false}
        },
    },
    "EXTRACT": {
        ReturnType: func(args []TypeInfo) TypeInfo {
            return TypeInfo{BaseType: "int", IsNullable: false}
        },
    },
}
```

## Type Promotion Rules

### Numeric Type Promotion
```go
func PromoteNumericTypes(left, right TypeInfo) TypeInfo {
    // Promotion hierarchy: int -> decimal -> float
    if IsFloatType(left) || IsFloatType(right) {
        return TypeInfo{BaseType: "float"}
    }
    if IsDecimalType(left) || IsDecimalType(right) {
        return TypeInfo{BaseType: "decimal"}
    }
    return TypeInfo{BaseType: "int"}
}
```

### Nullability Rules
```go
func InferNullability(operation string, operands []TypeInfo) bool {
    switch operation {
    case "AND", "OR":
        return AnyNullable(operands)
    case "+", "-", "*", "/":
        return AnyNullable(operands)
    case "COALESCE":
        return AllNullable(operands)
    default:
        return AnyNullable(operands)
    }
}
```

## Database-Specific Type Mappings

### PostgreSQL
```go
var PostgreSQLTypeMap = map[string]string{
    "int4":         "int",
    "int8":         "bigint",
    "varchar":      "string",
    "text":         "string",
    "bool":         "bool",
    "timestamp":    "time",
    "timestamptz":  "time",
    "decimal":      "decimal",
    "numeric":      "decimal",
    "float8":       "float",
    "json":         "json",
    "jsonb":        "json",
    "uuid":         "uuid",
    "bytea":        "bytes",
}
```

### MySQL
```go
var MySQLTypeMap = map[string]string{
    "int":          "int",
    "bigint":       "bigint",
    "varchar":      "string",
    "text":         "string",
    "tinyint":      "bool",
    "datetime":     "time",
    "timestamp":    "time",
    "decimal":      "decimal",
    "double":       "float",
    "json":         "json",
    "binary":       "bytes",
    "varbinary":    "bytes",
}
```

### SQLite
```go
var SQLiteTypeMap = map[string]string{
    "INTEGER":      "int",
    "TEXT":         "string",
    "REAL":         "float",
    "BLOB":         "bytes",
    "NUMERIC":      "decimal",
}
```

## Complex Scenarios

### JOIN Type Resolution
```go
func ResolveJoinTypes(selectClause *SelectClause, fromClause *FromClause) error {
    // 1. Build table alias mapping
    aliasMap := BuildAliasMap(fromClause)
    
    // 2. For each field, resolve table context
    for _, field := range selectClause.Fields {
        if field.Table != "" {
            // Explicit table reference
            tableName := ResolveAlias(field.Table, aliasMap)
            field.ResolvedTable = tableName
        } else {
            // Implicit reference - search all available tables
            candidates := FindColumnCandidates(field.Column, aliasMap)
            if len(candidates) == 1 {
                field.ResolvedTable = candidates[0]
            } else if len(candidates) > 1 {
                return ErrAmbiguousColumn
            } else {
                return ErrColumnNotFound
            }
        }
    }
}
```

### Subquery Type Resolution
```go
func InferSubqueryType(subquery *Subquery, context *InferenceContext) ([]*InferredField, error) {
    // 1. Create new context for subquery
    subContext := context.NewSubContext()
    
    // 2. Resolve subquery types recursively
    fields, err := AnalyzeSelectClause(subquery.SelectClause, subContext)
    if err != nil {
        return nil, err
    }
    
    // 3. Handle scalar vs. table subqueries
    if subquery.IsScalar {
        if len(fields) != 1 {
            return nil, ErrScalarSubqueryMultipleColumns
        }
        return fields, nil
    }
    
    return fields, nil
}
```

### CASE Expression Type Resolution
```go
func InferCaseType(caseExpr *CaseExpression, context *InferenceContext) (*InferredField, error) {
    var resultTypes []TypeInfo
    
    // 1. Infer types for all WHEN clauses
    for _, when := range caseExpr.WhenClauses {
        whenType, err := InferExpressionType(when.Result, context)
        if err != nil {
            return nil, err
        }
        resultTypes = append(resultTypes, whenType.Type)
    }
    
    // 2. Infer type for ELSE clause
    if caseExpr.ElseClause != nil {
        elseType, err := InferExpressionType(caseExpr.ElseClause, context)
        if err != nil {
            return nil, err
        }
        resultTypes = append(resultTypes, elseType.Type)
    }
    
    // 3. Find common type
    commonType := FindCommonType(resultTypes)
    
    return &InferredField{
        Type: commonType,
        Source: FieldSource{Type: ExpressionSource, Expression: "CASE"},
    }, nil
}
```

## Error Handling

### Error Types
```go
var (
    ErrColumnNotFound     = errors.New("column not found in schema")
    ErrAmbiguousColumn    = errors.New("ambiguous column reference")
    ErrUnknownFunction    = errors.New("unknown function")
    ErrIncompatibleTypes  = errors.New("incompatible types in expression")
    ErrUnsupportedType    = errors.New("unsupported data type")
    ErrCircularReference  = errors.New("circular reference in type inference")
)
```

### Error Context
```go
type TypeInferenceError struct {
    Message    string
    Field      string
    Expression string
    Line       int
    Column     int
    Cause      error
}

func (e *TypeInferenceError) Error() string {
    return fmt.Sprintf("type inference error at %d:%d in field '%s': %s", 
        e.Line, e.Column, e.Field, e.Message)
}
```

## Implementation Plan

### Phase 1: Core Infrastructure
1. Schema store implementation
2. Basic type system
3. Column reference resolution
4. Simple expression inference

### Phase 2: Function Support
1. Built-in function registry
2. Aggregate function support
3. String and math functions
4. Date/time functions

### Phase 3: Advanced Features
1. JOIN type resolution
2. Subquery support
3. CASE expression support
4. CTE (Common Table Expression) support

### Phase 4: Database-Specific Features
1. PostgreSQL-specific types and functions
2. MySQL-specific types and functions
3. SQLite-specific handling
4. Custom type extensions

## Testing Strategy

### Unit Tests
- Individual type inference functions
- Type promotion rules
- Function registry
- Error handling

### Integration Tests
- Complex queries with JOINs
- Nested subqueries
- Mixed expression types
- Real-world schema scenarios

### Database Tests
- Test against actual database schemas
- Verify type mappings
- Performance benchmarks
- Cross-database compatibility

## Performance Considerations

### Optimization Strategies
1. **Schema Caching**: Cache schema information to avoid repeated database queries
2. **Type Memoization**: Cache inferred types for repeated expressions
3. **Lazy Loading**: Load schema information on-demand
4. **Parallel Processing**: Process independent fields in parallel

### Memory Management
```go
type TypeCache struct {
    mu     sync.RWMutex
    cache  map[string]*TypeInfo
    maxSize int
}

func (tc *TypeCache) Get(key string) (*TypeInfo, bool) {
    tc.mu.RLock()
    defer tc.mu.RUnlock()
    
    typeInfo, exists := tc.cache[key]
    return typeInfo, exists
}
```

## Future Enhancements

### Advanced Type Features
1. **Generic Types**: Support for parameterized types
2. **Custom Types**: User-defined types and domains
3. **Array Types**: Multi-dimensional array support
4. **Composite Types**: Record and object types

### AI-Assisted Inference
1. **Machine Learning**: Learn from query patterns
2. **Heuristic Rules**: Smart guessing for ambiguous cases
3. **Context Awareness**: Use surrounding code context

### IDE Integration
1. **Real-time Inference**: Live type checking in editors
2. **Auto-completion**: Type-aware suggestions
3. **Error Highlighting**: Immediate feedback on type errors

## Conclusion

The type inference system provides a robust foundation for generating strongly-typed code from SQL templates. By leveraging database schema information and implementing comprehensive type resolution rules, we can achieve high accuracy in type inference while supporting complex SQL constructs across multiple database systems.

The modular design allows for incremental implementation and easy extension to support new databases, functions, and type systems as requirements evolve.
