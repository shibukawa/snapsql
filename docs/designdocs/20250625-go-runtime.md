# Go Low-Level Runtime Design Document

**Date**: 2025-06-25  
**Author**: Development Team  
**Status**: Draft (Revised)  

## Overview

This document outlines the design for the SnapSQL Go low-level runtime library, which provides specialized functionality for SQL generation and execution using Go 1.24 features.

## Goals

### Primary Goals
- Generate SQL from SnapSQL templates with runtime parameters
- Provide efficient PrepareContext with in-memory statement caching
- Execute queries using ExecContext/QueryContext on database/sql interfaces
- Return results using Go 1.24 iterator patterns for Row, Columns, ColumnTypes
- Minimal, focused API for SQL generation and execution

### Secondary Goals
- Statement caching for performance optimization
- Clean integration with database/sql.DB and database/sql.Tx
- Memory-efficient result iteration
- Proper context handling throughout

## Non-Goals
- High-level ORM functionality
- Database connection management
- Transaction management
- Mock/testing frameworks (separate concern)
- Configuration management
- Template parsing (uses pre-generated intermediate JSON)

## Architecture

### Core Components

#### 1. Template Loader (`runtime/snapsqlgo/loader.go`)
- Loads intermediate JSON files from any fs.FS implementation
- Supports go:embed, fs.Dir, fs.Root, and custom filesystem implementations
- Parses intermediate format into Template structures
- Provides first-level caching (filename → parsed template)

```go
type TemplateLoader struct {
    fs            fs.FS
    templateCache map[string]*Template
    mutex         sync.RWMutex
}

func NewTemplateLoader(fsys fs.FS) *TemplateLoader
func (l *TemplateLoader) LoadTemplate(templateName string) (*Template, error)
func (l *TemplateLoader) ClearTemplateCache()
```

#### 2. SQL Generator (`runtime/snapsqlgo/generator.go`)
- Generates SQL from templates with runtime parameters
- Provides second-level caching (structure-affecting params → generated SQL)
- Handles parameter substitution and conditional logic
- Centralized cache size configuration for all cache levels

```go
type SQLGeneratorConfig struct {
    TemplateCacheSize int // Default: unlimited (bounded by embedded templates)
    SQLCacheSize      int // Default: 1000
    StmtCacheSize     int // Default: 100 (can be overridden per Executor)
}

type SQLGenerator struct {
    loader    *TemplateLoader
    sqlCache  map[string]*CachedSQL
    config    SQLGeneratorConfig
    mutex     sync.RWMutex
}

type CachedSQL struct {
    SQL        string
    ParamNames []string
    GeneratedAt time.Time
}

func NewSQLGenerator(loader *TemplateLoader) *SQLGenerator
func NewSQLGeneratorWithConfig(loader *TemplateLoader, config SQLGeneratorConfig) *SQLGenerator
func (g *SQLGenerator) GenerateSQL(templateName string, params map[string]any) (string, []any, error)
func (g *SQLGenerator) GetDefaultStmtCacheSize() int
func (g *SQLGenerator) ClearSQLCache()
func (g *SQLGenerator) ClearAllCaches()
```

#### 3. Statement Cache (`runtime/snapsqlgo/cache.go`)
- Context-aware caching of prepared statements with hierarchical reusability
- Separate cache management for sql.DB, sql.Conn, and sql.Tx contexts
- Understands statement reusability rules across different contexts
- Thread-safe operations with proper cleanup

```go
type StatementCache struct {
    dbCache   map[string]*sql.Stmt                    // Global DB statements
    connCache map[uintptr]map[string]*sql.Stmt       // Per-connection statements
    txCache   map[uintptr]map[string]*sql.Stmt       // Per-transaction statements
    mutex     sync.RWMutex
    maxSize   int
}

func NewStatementCache(maxSize int) *StatementCache
func (c *StatementCache) Get(ctx ContextInfo, sql string) (*sql.Stmt, bool)
func (c *StatementCache) Set(ctx ContextInfo, sql string, stmt *sql.Stmt)
func (c *StatementCache) CleanupContext(ctx ContextInfo) // Clean up when context ends
func (c *StatementCache) Clear()
```

#### 4. Executor (`runtime/snapsqlgo/executor.go`)
- Generic executor that works with any DBExecutor implementation
- Automatic context detection and appropriate caching strategy
- Returns Go 1.24 iterator-based results
- Automatic context cleanup for statement cache
- Support for both prepared statements (default) and direct execution (for DDL, etc.)
- Statement cache size can be overridden per executor instance

```go
type Executor[DB DBExecutor] struct {
    generator *SQLGenerator
    stmtCache *StatementCache
    db        DB
    context   ContextInfo
}

func NewExecutor[DB DBExecutor](generator *SQLGenerator, db DB) *Executor[DB]
func NewExecutorWithStmtCacheSize[DB DBExecutor](generator *SQLGenerator, db DB, stmtCacheSize int) *Executor[DB]

// Standard methods using prepared statements (recommended for most cases)
func (e *Executor[DB]) Query(ctx context.Context, templateName string, params map[string]any) (*ResultIterator, error)
func (e *Executor[DB]) Exec(ctx context.Context, templateName string, params map[string]any) (sql.Result, error)

// Direct execution methods (for DDL, one-time queries, etc.)
func (e *Executor[DB]) QueryDirect(ctx context.Context, templateName string, params map[string]any) (*ResultIterator, error)
func (e *Executor[DB]) ExecDirect(ctx context.Context, templateName string, params map[string]any) (sql.Result, error)

// Raw SQL execution (bypass template system entirely)
func (e *Executor[DB]) QueryRaw(ctx context.Context, sql string, args ...any) (*ResultIterator, error)
func (e *Executor[DB]) ExecRaw(ctx context.Context, sql string, args ...any) (sql.Result, error)

// Statement cache management (can override generator's default)
func (e *Executor[DB]) SetStmtCacheSize(size int)
func (e *Executor[DB]) GetStmtCacheSize() int
func (e *Executor[DB]) ClearStmtCache()

// Cache and lifecycle management
func (e *Executor[DB]) ClearAllCaches()
func (e *Executor[DB]) Close() error

// Context detection helpers
func detectContextInfo[DB DBExecutor](db DB) ContextInfo
func (e *Executor[DB]) cleanup() // Called automatically when executor is closed
```

#### 5. Result Iterator (`runtime/snapsqlgo/iterator.go`)
- Go 1.24 iterator implementation for query results (unchanged from previous design)
- Provides access to Rows, Columns, and ColumnTypes
- Memory-efficient streaming of results

### Data Flow

1. **Template Loading (First-level Cache)**
   - Templates are embedded using `go:embed` directive
   - `TemplateLoader.LoadTemplate(templateName)` checks template cache
   - If not cached, loads and parses intermediate JSON file
   - Parsed template is cached with filename as key

2. **SQL Generation (Second-level Cache)**
   - `SQLGenerator.GenerateSQL(templateName, params)` extracts structure-affecting parameters
   - Structure-affecting parameters include: table suffixes, conditional field selections, etc.
   - Cache key is generated from template name + structure-affecting parameter hash
   - If cached SQL exists, parameter values are substituted into cached SQL
   - If not cached, SQL is generated from template AST and cached

3. **Statement Preparation and Caching (Third-level Cache)**
   - `Executor.Query/Exec` receives generated SQL and parameters
   - Context type is automatically detected from the generic DB parameter
   - Statement cache lookup follows hierarchical reusability rules:
     - **DB Context**: Check global DB cache
     - **Conn Context**: Check conn-specific cache, fallback to DB cache
     - **Tx Context**: Check tx-specific cache, fallback to parent conn cache, then DB cache
   - If not cached, `PrepareContext` is called on the DB instance and result is cached
   - Prepared statement is used for execution
   - Context-specific statements are cleaned up when context ends

4. **Query Execution**
   - For queries: `Query` is called, returns `ResultIterator`
   - For commands: `Exec` is called, returns `sql.Result`
   - Results use Go 1.24 iterator patterns for efficient processing

### Statement Cache Strategy Details

#### Hierarchical Statement Reusability
Prepared statements have different reusability scopes based on their preparation context:

**sql.DB Context:**
- Statements can be reused across all connections and transactions
- Highest reusability scope
- Cached globally with SQL as key

**sql.Conn Context:**
- Statements can be reused within the same connection and its transactions
- Medium reusability scope
- Cached per-connection, can fallback to DB cache for reuse

**sql.Tx Context:**
- Statements are bound to specific transaction
- Lowest reusability scope
- Cached per-transaction, can fallback to parent conn cache, then DB cache

#### Context Detection and Cache Strategy
```go
func detectContextInfo[DB DBExecutor](db DB) ContextInfo {
    switch v := any(db).(type) {
    case *sql.DB:
        return ContextInfo{
            Type: DBContext,
            ID:   uintptr(unsafe.Pointer(v)),
        }
    case *sql.Conn:
        return ContextInfo{
            Type:     ConnContext,
            ID:       uintptr(unsafe.Pointer(v)),
            ParentID: getParentDBID(v), // Implementation specific
        }
    case *sql.Tx:
        return ContextInfo{
            Type:     TxContext,
            ID:       uintptr(unsafe.Pointer(v)),
            ParentID: getParentConnID(v), // Implementation specific
        }
    default:
        panic("unsupported DB type")
    }
}

func (c *StatementCache) Get(ctx ContextInfo, sql string) (*sql.Stmt, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    switch ctx.Type {
    case DBContext:
        return c.dbCache[sql], true
    case ConnContext:
        // Check conn cache first, then fallback to DB cache
        if connStmts, exists := c.connCache[ctx.ID]; exists {
            if stmt, found := connStmts[sql]; found {
                return stmt, true
            }
        }
        return c.dbCache[sql], true
    case TxContext:
        // Check tx cache, then parent conn cache, then DB cache
        if txStmts, exists := c.txCache[ctx.ID]; exists {
            if stmt, found := txStmts[sql]; found {
                return stmt, true
            }
        }
        if connStmts, exists := c.connCache[ctx.ParentID]; exists {
            if stmt, found := connStmts[sql]; found {
                return stmt, true
            }
        }
        return c.dbCache[sql], true
    }
    return nil, false
}
```

### Cache Strategy Details

#### Structure-Affecting Parameters
Parameters that affect SQL structure and require separate cache entries:
- Table name suffixes (e.g., `table_suffix: "prod"` vs `table_suffix: "test"`)
- Conditional field inclusions (e.g., `include_email: true` vs `include_email: false`)
- Dynamic WHERE clause conditions that change SQL structure
- ORDER BY field selections
- Conditional JOIN clauses

#### Non-Structure-Affecting Parameters
Parameters that only affect values and can reuse cached SQL:
- Filter values (e.g., `user_id: 123` vs `user_id: 456`)
- Pagination parameters (LIMIT/OFFSET values)
- Search terms in WHERE clauses
- Date ranges and other filter criteria

#### Cache Key Generation
```go
type CacheKeyBuilder struct{}

func (b *CacheKeyBuilder) BuildSQLCacheKey(templateName string, params map[string]any) string {
    structuralParams := b.extractStructuralParams(params)
    hash := b.hashParams(structuralParams)
    return fmt.Sprintf("%s:%s", templateName, hash)
}

func (b *CacheKeyBuilder) extractStructuralParams(params map[string]any) map[string]any {
    // Extract only parameters that affect SQL structure
    // Implementation depends on template metadata
}
```

## API Design

### Core Interfaces

```go
package snapsqlgo

// DBExecutor represents the unified interface for database operations
type DBExecutor interface {
    PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Ensure sql.DB, sql.Conn, and sql.Tx implement DBExecutor
var (
    _ DBExecutor = (*sql.DB)(nil)
    _ DBExecutor = (*sql.Conn)(nil)
    _ DBExecutor = (*sql.Tx)(nil)
)

// ContextType represents the type of database context
type ContextType int

const (
    DBContext ContextType = iota
    ConnContext
    TxContext
)

// ContextInfo holds information about the database context
type ContextInfo struct {
    Type     ContextType
    ID       uintptr // Pointer to the actual DB/Conn/Tx instance
    ParentID uintptr // For Tx: parent Conn ID, for Conn: parent DB ID
}
```

// Generator handles SQL generation from templates
type Generator struct {
    // private fields
}

func NewGenerator(templatesDir string) (*Generator, error)
func (g *Generator) GenerateSQL(templateName string, params map[string]any) (string, []any, error)

// Executor handles SQL execution with statement caching
type Executor struct {
    // private fields
}

func NewExecutor(cacheSize int) *Executor
func (e *Executor) QueryContext(ctx context.Context, db QueryExecutor, sql string, args []any) (*ResultIterator, error)
func (e *Executor) ExecContext(ctx context.Context, db ExecExecutor, sql string, args []any) (sql.Result, error)
func (e *Executor) Close() error

// ResultIterator provides Go 1.24 iterator access to query results
type ResultIterator struct {
    // private fields
}

func (r *ResultIterator) All() iter.Seq2[[]any, error]
func (r *ResultIterator) Columns() []string
func (r *ResultIterator) ColumnTypes() []*sql.ColumnType
func (r *ResultIterator) Close() error
```

### Usage Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/shibukawa/snapsql/runtime/snapsqlgo"
)

func main() {
    // Initialize components with centralized cache configuration
    
    // Option 1: Using go:embed (most common for production)
    //go:embed templates/*.json
    var templatesFS embed.FS
    loader := snapsqlgo.NewTemplateLoader(templatesFS)
    
    // Option 2: Using filesystem directory (useful for development)
    // loader := snapsqlgo.NewTemplateLoader(os.DirFS("./generated"))
    
    // Option 3: Using fs.Root for subdirectory
    // subFS, _ := fs.Sub(templatesFS, "templates")
    // loader := snapsqlgo.NewTemplateLoader(subFS)
    
    // Configure cache sizes centrally
    config := snapsqlgo.SQLGeneratorConfig{
        TemplateCacheSize: 0,    // Unlimited (bounded by embedded templates)
        SQLCacheSize:      1000, // 1000 unique SQL combinations
        StmtCacheSize:     100,  // Default statement cache size
    }
    generator := snapsqlgo.NewSQLGeneratorWithConfig(loader, config)
    
    ctx := context.Background()
    db, _ := sql.Open("postgres", "connection_string")
    
    // Create executor with default statement cache size from generator
    dbExecutor := snapsqlgo.NewExecutor(generator, db)
    defer dbExecutor.Close()
    
    // Standard execution with prepared statements (recommended)
    results, err := dbExecutor.Query(ctx, "users", map[string]any{
        "include_email": true,        // structure-affecting
        "table_suffix": "prod",       // structure-affecting
        "active": true,               // value parameter
        "department": "engineering",  // value parameter
    })
    if err != nil {
        panic(err)
    }
    defer results.Close()
    
    // Iterate results using Go 1.24 iterator
    for row, err := range results.All() {
        if err != nil {
            panic(err)
        }
        fmt.Printf("Row: %v\n", row)
    }
    
    // Direct execution for DDL or one-time operations
    _, err = dbExecutor.ExecDirect(ctx, "create_index", map[string]any{
        "table_name": "users",
        "index_name": "idx_users_email",
        "column": "email",
    })
    if err != nil {
        panic(err)
    }
    
    // Raw SQL execution (bypass template system)
    rawResults, err := dbExecutor.QueryRaw(ctx, "SELECT version()")
    if err != nil {
        panic(err)
    }
    defer rawResults.Close()
    
    for row, err := range rawResults.All() {
        if err != nil {
            panic(err)
        }
        fmt.Printf("Database version: %v\n", row)
    }
    
    // Execute with connection context
    executeWithConnection(ctx, db, generator)
    
    // Execute within transaction with custom statement cache size
    executeInTransactionWithCustomCache(ctx, db, generator)
    
    // Clear caches when needed
    dbExecutor.ClearAllCaches()
}

func executeWithConnection(ctx context.Context, db *sql.DB, generator *snapsqlgo.SQLGenerator) {
    // Get dedicated connection
    conn, err := db.Conn(ctx)
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    
    // Create executor with default statement cache size
    connExecutor := snapsqlgo.NewExecutor(generator, conn)
    defer connExecutor.Close()
    
    // Execute queries - statements cached per-connection, can reuse DB statements
    results, err := connExecutor.Query(ctx, "orders", map[string]any{
        "user_id": 123,
        "status": "pending",
    })
    if err != nil {
        panic(err)
    }
    defer results.Close()
    
    for row, err := range results.All() {
        if err != nil {
            panic(err)
        }
        fmt.Printf("Order: %v\n", row)
    }
}

func executeInTransactionWithCustomCache(ctx context.Context, db *sql.DB, generator *snapsqlgo.SQLGenerator) {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        panic(err)
    }
    
    // Create executor with custom statement cache size (override generator's default)
    txExecutor := snapsqlgo.NewExecutorWithStmtCacheSize(generator, tx, 25)
    
    defer func() {
        txExecutor.Close() // Automatically cleans up transaction-specific statements
        if err != nil {
            tx.Rollback()
        } else {
            tx.Commit()
        }
    }()
    
    // Execute queries within transaction - can reuse conn and DB statements
    results, err := txExecutor.Query(ctx, "user_orders", map[string]any{
        "user_id": 456,
        "include_details": true,
    })
    if err != nil {
        return
    }
    defer results.Close()
    
    for row, err := range results.All() {
        if err != nil {
            return
        }
        fmt.Printf("User Order: %v\n", row)
    }
    
    // Execute update within same transaction
    _, err = txExecutor.Exec(ctx, "update_order_status", map[string]any{
        "order_id": 789,
        "status": "processed",
    })
}

// Example of cache size management
func demonstrateCacheManagement() {
    //go:embed templates/*.json
    var templatesFS embed.FS
    loader := snapsqlgo.NewTemplateLoader(templatesFS)
    
    // Default configuration
    generator1 := snapsqlgo.NewSQLGenerator(loader)
    
    // Custom configuration
    config := snapsqlgo.SQLGeneratorConfig{
        SQLCacheSize:  2000, // Larger SQL cache
        StmtCacheSize: 200,  // Larger default statement cache
    }
    generator2 := snapsqlgo.NewSQLGeneratorWithConfig(loader, config)
    
    db, _ := sql.Open("postgres", "connection_string")
    
    // Executor with generator's default statement cache size
    executor1 := snapsqlgo.NewExecutor(generator2, db)
    fmt.Printf("Executor1 stmt cache size: %d\n", executor1.GetStmtCacheSize()) // 200
    
    // Executor with custom statement cache size (overrides generator's default)
    executor2 := snapsqlgo.NewExecutorWithStmtCacheSize(generator2, db, 50)
    fmt.Printf("Executor2 stmt cache size: %d\n", executor2.GetStmtCacheSize()) // 50
    
    // Runtime cache size adjustment
    executor1.SetStmtCacheSize(300)
    fmt.Printf("Executor1 new stmt cache size: %d\n", executor1.GetStmtCacheSize()) // 300
    
    defer executor1.Close()
    defer executor2.Close()
}

// Example of different filesystem implementations
func demonstrateFilesystemOptions() {
    // 1. go:embed (production deployment)
    //go:embed templates/*.json
    var embeddedFS embed.FS
    loader1 := snapsqlgo.NewTemplateLoader(embeddedFS)
    
    // 2. Directory filesystem (development)
    dirFS := os.DirFS("./generated")
    loader2 := snapsqlgo.NewTemplateLoader(dirFS)
    
    // 3. Subdirectory of embedded filesystem
    //go:embed assets/templates/*.json
    var assetsFS embed.FS
    subFS, err := fs.Sub(assetsFS, "assets/templates")
    if err != nil {
        panic(err)
    }
    loader3 := snapsqlgo.NewTemplateLoader(subFS)
    
    // 4. Custom filesystem implementation (e.g., from database, network, etc.)
    // customFS := &MyCustomFS{...}
    // loader4 := snapsqlgo.NewTemplateLoader(customFS)
    
    // All loaders work the same way regardless of filesystem implementation
    generator1 := snapsqlgo.NewSQLGenerator(loader1)
    generator2 := snapsqlgo.NewSQLGenerator(loader2)
    generator3 := snapsqlgo.NewSQLGenerator(loader3)
    
    // Use generators normally
    _ = generator1
    _ = generator2
    _ = generator3
}
```

## Error Handling

### Sentinel Errors

```go
var (
    ErrTemplateNotFound     = errors.New("template not found")
    ErrInvalidParameters    = errors.New("invalid parameters")
    ErrSQLGeneration       = errors.New("SQL generation error")
    ErrStatementCache      = errors.New("statement cache error")
    ErrResultIteration     = errors.New("result iteration error")
)
```

### Error Wrapping

All errors with parameters will wrap sentinel errors:

```go
func (g *Generator) GenerateSQL(templateName string, params map[string]any) (string, []any, error) {
    template, exists := g.templateCache[templateName]
    if !exists {
        return "", nil, fmt.Errorf("%w: template '%s'", ErrTemplateNotFound, templateName)
    }
    // ... rest of implementation
}
```

## Performance Considerations

### Hierarchical Caching Strategy with Generics
1. **Template Cache**: Filename → Parsed Template (in-memory, persistent)
2. **SQL Cache**: Structure-affecting params → Generated SQL (in-memory, persistent)
3. **Statement Cache**: Hierarchical context-aware SQL → Prepared Statement
   - **DB Context**: Global cache, highest reusability, shared across all connections
   - **Conn Context**: Per-connection cache, can reuse DB statements, medium reusability
   - **Tx Context**: Per-transaction cache, can reuse Conn and DB statements, lowest reusability

### Generic Type Safety and Statement Reusability
- **Type Safety**: Generic `Executor[DB DBExecutor]` ensures compile-time type checking
- **Automatic Context Detection**: Runtime context detection from generic type parameter
- **Hierarchical Fallback**: Lower-level contexts can reuse higher-level cached statements
- **Optimal Reuse**: Maximizes statement reuse while respecting database/sql constraints

### Cache Efficiency and Statement Lifecycle
- Template parsing is expensive, cached permanently after first load
- SQL generation is moderately expensive, cached by structural parameters
- **DB Statement preparation**: Expensive, globally cached, maximum reusability
- **Conn Statement preparation**: Expensive, per-connection cached, can fallback to DB cache
- **Tx Statement preparation**: Expensive, per-transaction cached, can fallback to Conn and DB caches
- Value-only parameter changes reuse cached statements with different bind values

### Memory Management with Context Awareness
- Template cache: Bounded by number of embedded templates
- SQL cache: Bounded by unique combinations of structural parameters
- DB statement cache: LRU with configurable maximum size, application lifetime
- Conn statement cache: Automatically cleaned up when connection closes
- Tx statement cache: Automatically cleaned up when transaction ends
- Iterator-based result processing to minimize memory usage

### Generic Benefits
- **Single API**: One `NewExecutor` function works with all database context types
- **Type Inference**: Go compiler automatically infers correct generic type
- **Compile-Time Safety**: Prevents mixing incompatible database context types
- **Runtime Efficiency**: No interface boxing/unboxing overhead for database operations

### go:embed Benefits
- Zero file I/O at runtime for template loading when using embedded FS
- Templates bundled with binary, no external dependencies
- Compile-time validation of template file existence
- Flexible filesystem support for development and testing scenarios

## Go 1.24 Iterator Integration

### Iterator Pattern Implementation

```go
// ResultIterator implements Go 1.24 iterator patterns
type ResultIterator struct {
    rows        *sql.Rows
    columns     []string
    columnTypes []*sql.ColumnType
    closed      bool
}

// All returns an iterator over all rows
func (r *ResultIterator) All() iter.Seq2[[]any, error] {
    return func(yield func([]any, error) bool) {
        defer r.Close()
        
        for r.rows.Next() {
            values := make([]any, len(r.columns))
            valuePtrs := make([]any, len(r.columns))
            for i := range values {
                valuePtrs[i] = &values[i]
            }
            
            if err := r.rows.Scan(valuePtrs...); err != nil {
                yield(nil, fmt.Errorf("%w: %v", ErrResultIteration, err))
                return
            }
            
            if !yield(values, nil) {
                return
            }
        }
        
        if err := r.rows.Err(); err != nil {
            yield(nil, fmt.Errorf("%w: %v", ErrResultIteration, err))
        }
    }
}
```

## Testing Strategy

### Unit Tests
- Template loading from embedded files with various scenarios
- SQL generation with different parameter combinations and caching behavior
- Cache key generation for structural vs. value parameters
- Statement cache operations (get, set, eviction)
- Iterator functionality and resource cleanup
- Cache clearing operations
- Error handling scenarios

### Integration Tests
- End-to-end SQL generation and execution with TestContainers
- Multi-level caching with real database connections
- Iterator performance with large result sets
- Concurrent access patterns across all cache levels
- Cache invalidation scenarios

### Test Structure

```go
func TestSQLGenerator_GenerateSQL_Caching(t *testing.T) {
    ctx := t.Context()
    
    tests := []struct {
        name                    string
        templateName           string
        params1                map[string]any
        params2                map[string]any
        expectSameCachedSQL    bool
        expectedSQL            string
        expectedArgs1          []any
        expectedArgs2          []any
    }{
        {
            name:         "same_structural_params_different_values",
            templateName: "users",
            params1: map[string]any{
                "include_email": true,
                "active":       true,
                "user_id":      123,
            },
            params2: map[string]any{
                "include_email": true,
                "active":       false,
                "user_id":      456,
            },
            expectSameCachedSQL: true,
            expectedSQL:        "SELECT id, name, email FROM users WHERE active = ? AND user_id = ?",
            expectedArgs1:      []any{true, 123},
            expectedArgs2:      []any{false, 456},
        },
        {
            name:         "different_structural_params",
            templateName: "users",
            params1: map[string]any{
                "include_email": true,
                "active":       true,
            },
            params2: map[string]any{
                "include_email": false,
                "active":       true,
            },
            expectSameCachedSQL: false,
        },
        // more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            loader := NewTemplateLoader()
            generator := NewSQLGenerator(loader)
            
            // First call
            sql1, args1, err := generator.GenerateSQL(tt.templateName, tt.params1)
            assert.NoError(t, err)
            
            // Second call
            sql2, args2, err := generator.GenerateSQL(tt.templateName, tt.params2)
            assert.NoError(t, err)
            
            if tt.expectSameCachedSQL {
                assert.Equal(t, sql1, sql2, "SQL should be same when structural params are identical")
                assert.Equal(t, tt.expectedSQL, sql1)
                assert.Equal(t, tt.expectedArgs1, args1)
                assert.Equal(t, tt.expectedArgs2, args2)
            } else {
                assert.NotEqual(t, sql1, sql2, "SQL should be different when structural params differ")
            }
        })
    }
}
```

## Implementation Plan

### Phase 1: Template Loading and First-level Cache
1. Implement go:embed integration for intermediate JSON files
2. Template parsing from embedded files
3. First-level cache (filename → template)
4. Unit tests for template loading and caching

### Phase 2: SQL Generation and Second-level Cache
1. SQL generation from template AST
2. Structure-affecting parameter identification
3. Cache key generation and second-level cache
4. Parameter substitution for cached SQL
5. Unit tests for SQL generation and caching behavior

### Phase 3: Statement Caching and Execution
1. Third-level LRU cache for prepared statements
2. Executor implementation with integrated caching
3. Go 1.24 iterator-based result handling
4. Integration tests with real databases

### Phase 4: Cache Management and Optimization
1. Cache clearing mechanisms
2. Performance optimizations
3. Memory usage improvements
4. Comprehensive benchmarking

## Directory Structure

```
runtime/snapsqlgo/
├── templates/            # Embedded intermediate JSON files
│   ├── users.json
│   ├── orders.json
│   └── ...
├── loader.go            # Template loading with go:embed
├── generator.go         # SQL generation with 2nd-level cache
├── cache.go             # Statement cache (3rd-level)
├── executor.go          # Integrated execution with all caches
├── iterator.go          # Go 1.24 iterator implementation
├── template.go          # Template representation
├── cache_key.go         # Cache key generation logic
├── errors.go            # Sentinel errors
├── loader_test.go       # Template loader tests
├── generator_test.go    # SQL generator tests
├── cache_test.go        # Cache tests
├── executor_test.go     # Executor tests
├── iterator_test.go     # Iterator tests
└── integration_test.go  # End-to-end integration tests
```

## References

- [SnapSQL README](../README.md)
- [Intermediate Format Schema](../intermediate-format-schema.json)
- [Coding Standards](../coding-standard.md)
