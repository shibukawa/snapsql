# CTE (Common Table Expression) Support Design

## Overview

This document outlines the design for implementing CTE (WITH clause) support in SnapSQL. CTEs allow defining temporary named result sets that can be referenced within a SELECT, INSERT, UPDATE, or DELETE statement.

## Requirements

### Functional Requirements

1. **Basic CTE Support**
   - Parse `WITH table_name AS (subquery)` syntax
   - Support multiple CTEs in a single statement
   - Support CTE references in main query and other CTEs

2. **Recursive CTE Support**
   - Parse `WITH RECURSIVE table_name AS (...)` syntax
   - Handle recursive CTE structure validation

3. **SnapSQL Integration**
   - Support SnapSQL directives within CTE definitions
   - Support variable substitution in CTE queries
   - Support conditional CTE inclusion

4. **AST Integration**
   - Extend AST to represent CTE structures
   - Maintain compatibility with existing parser

### Non-Functional Requirements

1. **Performance**: CTE parsing should not significantly impact parser performance
2. **Maintainability**: Clean separation of CTE logic from existing parser
3. **Extensibility**: Design should allow future CTE enhancements

## Design

### AST Extensions

#### New Node Types

```go
// Add to NodeType enum
WITH_CLAUSE     // WITH clause containing CTEs
CTE_DEFINITION  // Individual CTE definition

// New AST structures
type WithClause struct {
    BaseAstNode
    CTEs []CTEDefinition
}

type CTEDefinition struct {
    BaseAstNode
    Name      string
    Recursive bool
    Query     *SelectStatement
    Columns   []string // Optional column list
}
```

#### SelectStatement Extension

```go
type SelectStatement struct {
    BaseAstNode
    WithClause    *WithClause      // NEW: CTE support
    SelectClause  *SelectClause
    FromClause    *FromClause
    WhereClause   *WhereClause
    OrderByClause *OrderByClause
    GroupByClause *GroupByClause
    HavingClause  *HavingClause
    LimitClause   *LimitClause
    OffsetClause  *OffsetClause
}
```

### Parser Extensions

#### CTE Parsing Logic

```go
// New parsing methods
func (p *SqlParser) parseWithClause() (*WithClause, error)
func (p *SqlParser) parseCTEDefinition() (*CTEDefinition, error)
func (p *SqlParser) parseCTEColumnList() ([]string, error)
```

#### Integration Points

1. **parseSelectStatement()**: Check for WITH keyword at the beginning
2. **Clause Validation**: Extend clause validator for CTE constraints
3. **Variable Resolution**: Handle CTE table references

### Tokenizer Integration

The tokenizer already supports the `WITH` keyword, so no changes are needed.

### SnapSQL Integration

#### Supported Patterns

```sql
-- Conditional CTE inclusion
/*# if include_stats */
WITH user_stats AS (
    SELECT user_id, COUNT(*) as post_count
    FROM posts 
    WHERE created_at > /*= date_filter */
    GROUP BY user_id
)
/*# end */
SELECT u.name, /*# if include_stats */s.post_count/*# end */
FROM users u
/*# if include_stats */
LEFT JOIN user_stats s ON u.id = s.user_id
/*# end */

-- Variable substitution in CTE
WITH filtered_data AS (
    SELECT * FROM /*= table_name */
    WHERE status = /*= status_filter */
)
SELECT * FROM filtered_data
```

### Implementation Plan

#### Phase 1: AST Extensions
1. Add CTE-related node types to ast.go
2. Extend SelectStatement structure
3. Implement String() methods for new nodes

#### Phase 2: Parser Extensions
1. Implement parseWithClause() method
2. Implement parseCTEDefinition() method
3. Integrate WITH clause parsing into parseSelectStatement()
4. Add CTE-specific error handling

#### Phase 3: Validation Extensions
1. Extend clause validator for CTE constraints
2. Add CTE reference validation
3. Handle recursive CTE validation

#### Phase 4: Testing
1. Unit tests for CTE parsing
2. Integration tests with SnapSQL directives
3. Error handling tests
4. Performance tests

## Examples

### Basic CTE

```sql
WITH active_users AS (
    SELECT id, name FROM users WHERE active = true
)
SELECT * FROM active_users WHERE name LIKE 'A%'
```

### Multiple CTEs

```sql
WITH 
    active_users AS (
        SELECT id, name FROM users WHERE active = true
    ),
    user_posts AS (
        SELECT user_id, COUNT(*) as post_count
        FROM posts
        GROUP BY user_id
    )
SELECT u.name, COALESCE(p.post_count, 0) as posts
FROM active_users u
LEFT JOIN user_posts p ON u.id = p.user_id
```

### Recursive CTE

```sql
WITH RECURSIVE employee_hierarchy AS (
    -- Anchor member
    SELECT id, name, manager_id, 0 as level
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    -- Recursive member
    SELECT e.id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.id
)
SELECT * FROM employee_hierarchy ORDER BY level, name
```

### SnapSQL Integration

```sql
/*# if include_hierarchy */
WITH RECURSIVE employee_hierarchy AS (
    SELECT id, name, manager_id, 0 as level
    FROM employees
    WHERE department = /*= department_filter */
    
    UNION ALL
    
    SELECT e.id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.id
    WHERE eh.level < /*= max_depth */
)
/*# end */
SELECT 
    /*# if include_hierarchy */eh.level,/*# end */
    e.name,
    e.department
FROM employees e
/*# if include_hierarchy */
INNER JOIN employee_hierarchy eh ON e.id = eh.id
/*# end */
WHERE e.active = /*= active_filter */
```

## Testing Strategy

### Unit Tests

1. **Basic CTE Parsing**
   - Single CTE
   - Multiple CTEs
   - CTE with column lists
   - Recursive CTEs

2. **Error Handling**
   - Invalid CTE syntax
   - Missing AS keyword
   - Invalid subquery
   - Circular references

3. **SnapSQL Integration**
   - Conditional CTEs
   - Variable substitution in CTEs
   - Complex nested scenarios

### Integration Tests

1. **End-to-End Parsing**
   - Complete SQL statements with CTEs
   - Mixed CTE and regular clauses
   - Performance with large CTEs

2. **AST Validation**
   - Correct AST structure generation
   - Proper node relationships
   - String representation accuracy

## Risk Assessment

### Technical Risks

1. **Parser Complexity**: Adding CTE support increases parser complexity
   - **Mitigation**: Clean separation of CTE logic, comprehensive testing

2. **Performance Impact**: CTE parsing might slow down the parser
   - **Mitigation**: Efficient parsing algorithms, performance testing

3. **SnapSQL Integration**: Complex interactions with existing SnapSQL features
   - **Mitigation**: Careful design, extensive integration testing

### Implementation Risks

1. **Recursive CTE Complexity**: Recursive CTEs have complex validation rules
   - **Mitigation**: Phase implementation, focus on basic CTEs first

2. **Backward Compatibility**: Changes might break existing functionality
   - **Mitigation**: Comprehensive regression testing

## Success Criteria

1. **Functional**: All CTE syntax variants are correctly parsed
2. **Integration**: SnapSQL directives work seamlessly with CTEs
3. **Performance**: No significant performance degradation
4. **Quality**: 100% test coverage for CTE functionality
5. **Compatibility**: All existing tests continue to pass

## Future Enhancements

1. **CTE Optimization**: Analyze CTE usage patterns for optimization hints
2. **Advanced Validation**: Cross-CTE reference validation
3. **Documentation**: Generate documentation for CTE usage patterns
4. **IDE Support**: Enhanced syntax highlighting for CTEs with SnapSQL
