# Type Inference System - Implementation Examples

## Overview

This document provides concrete implementation examples and test cases for the type inference system described in the main design document.

## Example Scenarios

### Scenario 1: Simple Column Selection

**SQL Query:**
```sql
SELECT id, name, email, created_at 
FROM users;
```

**Schema:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Expected Type Inference:**
```go
[]*InferredField{
    {
        Name: "id",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "id"},
    },
    {
        Name: "name", 
        Type: &TypeInfo{BaseType: "string", IsNullable: false, MaxLength: 255},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "name"},
    },
    {
        Name: "email",
        Type: &TypeInfo{BaseType: "string", IsNullable: true, MaxLength: 255},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "email"},
    },
    {
        Name: "created_at",
        Type: &TypeInfo{BaseType: "time", IsNullable: true},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "created_at"},
    },
}
```

### Scenario 2: JOIN with Aliases

**SQL Query:**
```sql
SELECT u.id, u.name, d.name as department_name, u.salary * 1.1 as adjusted_salary
FROM users u
JOIN departments d ON u.department_id = d.id;
```

**Schema:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id INTEGER REFERENCES departments(id),
    salary DECIMAL(10,2)
);

CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
```

**Expected Type Inference:**
```go
[]*InferredField{
    {
        Name: "id",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "id"},
    },
    {
        Name: "name",
        Type: &TypeInfo{BaseType: "string", IsNullable: false, MaxLength: 255},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "name"},
    },
    {
        Name: "department_name",
        Alias: "department_name",
        Type: &TypeInfo{BaseType: "string", IsNullable: false, MaxLength: 100},
        Source: FieldSource{Type: ColumnSource, Table: "departments", Column: "name"},
    },
    {
        Name: "adjusted_salary",
        Alias: "adjusted_salary",
        Type: &TypeInfo{BaseType: "decimal", IsNullable: true, Precision: 10, Scale: 2},
        Source: FieldSource{Type: ExpressionSource, Expression: "u.salary * 1.1"},
        Dependencies: []string{"users.salary"},
    },
}
```

### Scenario 3: Aggregate Functions

**SQL Query:**
```sql
SELECT 
    department_id,
    COUNT(*) as employee_count,
    AVG(salary) as avg_salary,
    MAX(created_at) as latest_hire,
    SUM(CASE WHEN active THEN 1 ELSE 0 END) as active_count
FROM users 
GROUP BY department_id;
```

**Expected Type Inference:**
```go
[]*InferredField{
    {
        Name: "department_id",
        Type: &TypeInfo{BaseType: "int", IsNullable: true},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "department_id"},
    },
    {
        Name: "employee_count",
        Alias: "employee_count",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: FunctionSource, Expression: "COUNT(*)"},
    },
    {
        Name: "avg_salary",
        Alias: "avg_salary",
        Type: &TypeInfo{BaseType: "decimal", IsNullable: true},
        Source: FieldSource{Type: FunctionSource, Expression: "AVG(salary)"},
        Dependencies: []string{"users.salary"},
    },
    {
        Name: "latest_hire",
        Alias: "latest_hire",
        Type: &TypeInfo{BaseType: "time", IsNullable: true},
        Source: FieldSource{Type: FunctionSource, Expression: "MAX(created_at)"},
        Dependencies: []string{"users.created_at"},
    },
    {
        Name: "active_count",
        Alias: "active_count",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: FunctionSource, Expression: "SUM(CASE WHEN active THEN 1 ELSE 0 END)"},
        Dependencies: []string{"users.active"},
    },
}
```

### Scenario 4: Complex Expressions and Functions

**SQL Query:**
```sql
SELECT 
    CONCAT(first_name, ' ', last_name) as full_name,
    EXTRACT(YEAR FROM created_at) as hire_year,
    CASE 
        WHEN salary > 100000 THEN 'Senior'
        WHEN salary > 50000 THEN 'Mid'
        ELSE 'Junior'
    END as level,
    COALESCE(phone, email, 'No contact') as contact_info
FROM employees;
```

**Expected Type Inference:**
```go
[]*InferredField{
    {
        Name: "full_name",
        Alias: "full_name",
        Type: &TypeInfo{BaseType: "string", IsNullable: true},
        Source: FieldSource{Type: FunctionSource, Expression: "CONCAT(first_name, ' ', last_name)"},
        Dependencies: []string{"employees.first_name", "employees.last_name"},
    },
    {
        Name: "hire_year",
        Alias: "hire_year",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: FunctionSource, Expression: "EXTRACT(YEAR FROM created_at)"},
        Dependencies: []string{"employees.created_at"},
    },
    {
        Name: "level",
        Alias: "level",
        Type: &TypeInfo{BaseType: "string", IsNullable: false},
        Source: FieldSource{Type: ExpressionSource, Expression: "CASE WHEN salary > 100000 THEN 'Senior' WHEN salary > 50000 THEN 'Mid' ELSE 'Junior' END"},
        Dependencies: []string{"employees.salary"},
    },
    {
        Name: "contact_info",
        Alias: "contact_info",
        Type: &TypeInfo{BaseType: "string", IsNullable: false},
        Source: FieldSource{Type: FunctionSource, Expression: "COALESCE(phone, email, 'No contact')"},
        Dependencies: []string{"employees.phone", "employees.email"},
    },
}
```

### Scenario 5: Subqueries

**SQL Query:**
```sql
SELECT 
    u.name,
    u.salary,
    (SELECT AVG(salary) FROM users WHERE department_id = u.department_id) as dept_avg_salary,
    (SELECT COUNT(*) FROM projects p WHERE p.owner_id = u.id) as project_count
FROM users u;
```

**Expected Type Inference:**
```go
[]*InferredField{
    {
        Name: "name",
        Type: &TypeInfo{BaseType: "string", IsNullable: false},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "name"},
    },
    {
        Name: "salary",
        Type: &TypeInfo{BaseType: "decimal", IsNullable: true},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "salary"},
    },
    {
        Name: "dept_avg_salary",
        Alias: "dept_avg_salary",
        Type: &TypeInfo{BaseType: "decimal", IsNullable: true},
        Source: FieldSource{Type: SubquerySource, Expression: "(SELECT AVG(salary) FROM users WHERE department_id = u.department_id)"},
        Dependencies: []string{"users.salary", "users.department_id"},
    },
    {
        Name: "project_count",
        Alias: "project_count",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: SubquerySource, Expression: "(SELECT COUNT(*) FROM projects p WHERE p.owner_id = u.id)"},
        Dependencies: []string{"users.id", "projects.owner_id"},
    },
}
```

## Implementation Code Examples

### Core Type Inference Engine

```go
package typeinference

import (
    "fmt"
    "strings"
    "github.com/shibukawa/snapsql/parser"
)

type TypeInferenceEngine struct {
    schema    *SchemaStore
    functions *FunctionRegistry
    cache     *TypeCache
}

func NewTypeInferenceEngine(schema *SchemaStore) *TypeInferenceEngine {
    return &TypeInferenceEngine{
        schema:    schema,
        functions: NewFunctionRegistry(),
        cache:     NewTypeCache(),
    }
}

func (tie *TypeInferenceEngine) InferSelectTypes(selectClause *parser.SelectClause, context *InferenceContext) ([]*InferredField, error) {
    var fields []*InferredField
    
    for i, field := range selectClause.Fields {
        inferredField, err := tie.inferFieldType(field, context)
        if err != nil {
            return nil, fmt.Errorf("failed to infer type for field %d: %w", i, err)
        }
        fields = append(fields, inferredField)
    }
    
    return fields, nil
}

func (tie *TypeInferenceEngine) inferFieldType(field *parser.SelectField, context *InferenceContext) (*InferredField, error) {
    // Check cache first
    cacheKey := tie.buildCacheKey(field, context)
    if cached, exists := tie.cache.Get(cacheKey); exists {
        return cached, nil
    }
    
    var result *InferredField
    var err error
    
    switch field.Type {
    case parser.ColumnReference:
        result, err = tie.inferColumnType(field, context)
    case parser.FunctionCall:
        result, err = tie.inferFunctionType(field, context)
    case parser.Expression:
        result, err = tie.inferExpressionType(field, context)
    case parser.Subquery:
        result, err = tie.inferSubqueryType(field, context)
    case parser.CaseExpression:
        result, err = tie.inferCaseType(field, context)
    default:
        err = fmt.Errorf("unsupported field type: %v", field.Type)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    tie.cache.Set(cacheKey, result)
    
    return result, nil
}
```

### Column Type Resolution

```go
func (tie *TypeInferenceEngine) inferColumnType(field *parser.SelectField, context *InferenceContext) (*InferredField, error) {
    tableName := field.Table
    columnName := field.Column
    
    // Resolve table alias if present
    if tableName != "" {
        if resolved, exists := context.TableAliases[tableName]; exists {
            tableName = resolved
        }
    } else {
        // Find table containing this column
        candidates := tie.findColumnCandidates(columnName, context)
        switch len(candidates) {
        case 0:
            return nil, fmt.Errorf("column '%s' not found in any available table", columnName)
        case 1:
            tableName = candidates[0]
        default:
            return nil, fmt.Errorf("ambiguous column reference '%s', found in tables: %v", columnName, candidates)
        }
    }
    
    // Get column info from schema
    table, exists := tie.schema.Tables[tableName]
    if !exists {
        return nil, fmt.Errorf("table '%s' not found in schema", tableName)
    }
    
    column, exists := table.Columns[columnName]
    if !exists {
        return nil, fmt.Errorf("column '%s' not found in table '%s'", columnName, tableName)
    }
    
    // Convert database type to canonical type
    typeInfo := tie.convertDatabaseType(column, context.Dialect)
    
    return &InferredField{
        Name:  columnName,
        Alias: field.Alias,
        Type:  typeInfo,
        Source: FieldSource{
            Type:   ColumnSource,
            Table:  tableName,
            Column: columnName,
        },
    }, nil
}

func (tie *TypeInferenceEngine) convertDatabaseType(column *ColumnInfo, dialect string) *TypeInfo {
    var typeMap map[string]string
    
    switch dialect {
    case "postgres":
        typeMap = PostgreSQLTypeMap
    case "mysql":
        typeMap = MySQLTypeMap
    case "sqlite":
        typeMap = SQLiteTypeMap
    default:
        typeMap = PostgreSQLTypeMap // Default to PostgreSQL
    }
    
    baseType, exists := typeMap[strings.ToLower(column.DataType)]
    if !exists {
        baseType = "unknown"
    }
    
    return &TypeInfo{
        BaseType:     baseType,
        DatabaseType: column.DataType,
        IsNullable:   column.IsNullable,
        MaxLength:    column.MaxLength,
        Precision:    column.Precision,
        Scale:        column.Scale,
    }
}
```

### Function Type Resolution

```go
func (tie *TypeInferenceEngine) inferFunctionType(field *parser.SelectField, context *InferenceContext) (*InferredField, error) {
    funcCall := field.Function
    funcName := strings.ToUpper(funcCall.Name)
    
    // Get function definition
    funcDef, exists := tie.functions.Get(funcName)
    if !exists {
        return nil, fmt.Errorf("unknown function: %s", funcName)
    }
    
    // Infer argument types
    var argTypes []TypeInfo
    var dependencies []string
    
    for _, arg := range funcCall.Args {
        argField := &parser.SelectField{
            Type:       arg.Type,
            Column:     arg.Column,
            Table:      arg.Table,
            Expression: arg.Expression,
            Function:   arg.Function,
        }
        
        inferredArg, err := tie.inferFieldType(argField, context)
        if err != nil {
            return nil, fmt.Errorf("failed to infer argument type: %w", err)
        }
        
        argTypes = append(argTypes, *inferredArg.Type)
        dependencies = append(dependencies, inferredArg.Dependencies...)
    }
    
    // Calculate return type
    returnType := funcDef.ReturnType(argTypes)
    
    return &InferredField{
        Name:  funcCall.Name,
        Alias: field.Alias,
        Type:  &returnType,
        Source: FieldSource{
            Type:       FunctionSource,
            Expression: tie.buildFunctionExpression(funcCall),
        },
        Dependencies: dependencies,
    }, nil
}
```

### Expression Type Resolution

```go
func (tie *TypeInferenceEngine) inferExpressionType(field *parser.SelectField, context *InferenceContext) (*InferredField, error) {
    expr := field.Expression
    
    switch expr.Type {
    case parser.BinaryOperation:
        return tie.inferBinaryOpType(expr, context)
    case parser.UnaryOperation:
        return tie.inferUnaryOpType(expr, context)
    case parser.Literal:
        return tie.inferLiteralType(expr, context)
    case parser.Cast:
        return tie.inferCastType(expr, context)
    default:
        return nil, fmt.Errorf("unsupported expression type: %v", expr.Type)
    }
}

func (tie *TypeInferenceEngine) inferBinaryOpType(expr *parser.Expression, context *InferenceContext) (*InferredField, error) {
    // Infer left operand type
    leftField, err := tie.inferFieldType(expr.Left, context)
    if err != nil {
        return nil, fmt.Errorf("failed to infer left operand type: %w", err)
    }
    
    // Infer right operand type
    rightField, err := tie.inferFieldType(expr.Right, context)
    if err != nil {
        return nil, fmt.Errorf("failed to infer right operand type: %w", err)
    }
    
    // Apply type promotion rules
    resultType := tie.promoteTypes(expr.Operator, *leftField.Type, *rightField.Type)
    
    dependencies := append(leftField.Dependencies, rightField.Dependencies...)
    
    return &InferredField{
        Type: &resultType,
        Source: FieldSource{
            Type:       ExpressionSource,
            Expression: tie.buildBinaryExpression(expr),
        },
        Dependencies: dependencies,
    }, nil
}
```

## Test Cases

### Unit Test Examples

```go
package typeinference

import (
    "testing"
    "github.com/alecthomas/assert/v2"
)

func TestInferColumnType(t *testing.T) {
    schema := &SchemaStore{
        Tables: map[string]*TableInfo{
            "users": {
                Name: "users",
                Columns: map[string]*ColumnInfo{
                    "id": {
                        Name:       "id",
                        DataType:   "int4",
                        IsNullable: false,
                    },
                    "name": {
                        Name:       "name",
                        DataType:   "varchar",
                        IsNullable: false,
                        MaxLength:  intPtr(255),
                    },
                },
            },
        },
    }
    
    engine := NewTypeInferenceEngine(schema)
    context := &InferenceContext{
        Dialect: "postgres",
        TableAliases: map[string]string{},
    }
    
    field := &parser.SelectField{
        Type:   parser.ColumnReference,
        Table:  "users",
        Column: "name",
    }
    
    result, err := engine.inferColumnType(field, context)
    
    assert.NoError(t, err)
    assert.Equal(t, "name", result.Name)
    assert.Equal(t, "string", result.Type.BaseType)
    assert.Equal(t, false, result.Type.IsNullable)
    assert.Equal(t, 255, *result.Type.MaxLength)
}

func TestInferFunctionType(t *testing.T) {
    schema := &SchemaStore{
        Tables: map[string]*TableInfo{
            "users": {
                Name: "users",
                Columns: map[string]*ColumnInfo{
                    "salary": {
                        Name:       "salary",
                        DataType:   "decimal",
                        IsNullable: true,
                        Precision:  intPtr(10),
                        Scale:      intPtr(2),
                    },
                },
            },
        },
    }
    
    engine := NewTypeInferenceEngine(schema)
    context := &InferenceContext{
        Dialect: "postgres",
        TableAliases: map[string]string{},
    }
    
    field := &parser.SelectField{
        Type: parser.FunctionCall,
        Function: &parser.FunctionCall{
            Name: "AVG",
            Args: []*parser.SelectField{
                {
                    Type:   parser.ColumnReference,
                    Table:  "users",
                    Column: "salary",
                },
            },
        },
    }
    
    result, err := engine.inferFunctionType(field, context)
    
    assert.NoError(t, err)
    assert.Equal(t, "AVG", result.Name)
    assert.Equal(t, "decimal", result.Type.BaseType)
    assert.Equal(t, true, result.Type.IsNullable)
    assert.Contains(t, result.Dependencies, "users.salary")
}
```

### Integration Test Examples

```go
func TestComplexQueryTypeInference(t *testing.T) {
    schema := buildTestSchema()
    engine := NewTypeInferenceEngine(schema)
    
    sqlQuery := `
        SELECT 
            u.id,
            u.name,
            d.name as department_name,
            COUNT(*) as employee_count,
            AVG(u.salary) as avg_salary,
            CASE 
                WHEN u.salary > 100000 THEN 'Senior'
                ELSE 'Junior'
            END as level
        FROM users u
        JOIN departments d ON u.department_id = d.id
        GROUP BY u.id, u.name, d.name, u.salary
    `
    
    // Parse SQL (assuming we have a parser)
    ast, err := parser.Parse(sqlQuery)
    assert.NoError(t, err)
    
    context := &InferenceContext{
        Dialect: "postgres",
        TableAliases: map[string]string{
            "u": "users",
            "d": "departments",
        },
    }
    
    fields, err := engine.InferSelectTypes(ast.SelectClause, context)
    assert.NoError(t, err)
    assert.Equal(t, 6, len(fields))
    
    // Verify field types
    assert.Equal(t, "int", fields[0].Type.BaseType)      // u.id
    assert.Equal(t, "string", fields[1].Type.BaseType)   // u.name
    assert.Equal(t, "string", fields[2].Type.BaseType)   // department_name
    assert.Equal(t, "int", fields[3].Type.BaseType)      // employee_count
    assert.Equal(t, "decimal", fields[4].Type.BaseType)  // avg_salary
    assert.Equal(t, "string", fields[5].Type.BaseType)   // level
    
    // Verify nullability
    assert.Equal(t, false, fields[0].Type.IsNullable)    // u.id (primary key)
    assert.Equal(t, false, fields[3].Type.IsNullable)    // COUNT(*) never null
    assert.Equal(t, true, fields[4].Type.IsNullable)     // AVG can be null
    assert.Equal(t, false, fields[5].Type.IsNullable)    // CASE with ELSE never null
}
```

## Performance Benchmarks

```go
func BenchmarkTypeInference(b *testing.B) {
    schema := buildLargeTestSchema() // 100 tables, 1000 columns
    engine := NewTypeInferenceEngine(schema)
    
    complexQuery := buildComplexQuery() // Query with 50 fields, multiple JOINs
    ast, _ := parser.Parse(complexQuery)
    
    context := &InferenceContext{
        Dialect: "postgres",
        TableAliases: buildAliasMap(),
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := engine.InferSelectTypes(ast.SelectClause, context)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

This implementation provides a solid foundation for type inference with comprehensive error handling, caching, and support for complex SQL constructs.
