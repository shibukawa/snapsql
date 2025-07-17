# SnapSQL New Parser (parser2) Design Document

## Overview

The SnapSQL parser has been redesigned with a focus on maintainability, extensibility, and testability. The parsing process is divided into seven steps, each implemented as independent functions. The parser is built using the `parsercombinator` package.

## Parsing Process Flow

### 1. Lexical Analysis (Lexer)
- Break SQL text into token sequences
- Identify comments, strings, numbers, operators, keywords
- Identify SnapSQL directives (/*# if */, /*= var */, etc.)

### 2. Basic Syntax Check (parserstep1)
- Check bracket matching ((), [], {})
- Verify SnapSQL directive matching (if/for/else/elseif/end)
- Validate nesting structure
- Detect basic syntax errors

### 3. SQL Grammar Check (parserstep2)
- Verify basic structure of SELECT, INSERT, UPDATE, DELETE statements
- Check clause order (WHERE, ORDER BY, GROUP BY, etc.)
- Detect trailing OR/AND, commas (for automatic processing)
- Group SQL structures

### 4. SnapSQL Directive Analysis (parserstep3)
- Parse directive syntax
- Parse conditions (CEL expressions)
- Parse variable references
- Validate directive nesting relationships

### 5. AST Construction (parserstep4)
- Convert SQL structure to AST
- Insert pseudo-IF nodes (for clause ON/OFF control)
- Automatic comma node insertion
- Insert sentinel nodes

### 6. AST Optimization (parserstep5)
- Remove unnecessary nodes
- Merge nodes
- Optimize conditions
- Add type information

### 7. Intermediate Format Generation (parserstep6)
- Convert AST to intermediate.Instruction sequence
- Add runtime required information
- Add debug information

## Directory Structure

```
parser2/
  ├── parsercommon/   # Common types, functions, utilities
  ├── parserstep1/    # Basic syntax check
  ├── parserstep2/    # SQL grammar check
  ├── parserstep3/    # SnapSQL directive analysis
  ├── parserstep4/    # AST construction
  ├── parserstep5/    # AST optimization
  ├── parserstep6/    # Intermediate format generation
  ├── parser.go       # Public API
  └── errors.go       # Error definitions
```

## Error Handling

### 1. Syntax Errors
- Bracket mismatches
- Directive mismatches
- SQL grammar errors
- Detailed position information (line, column) in error messages

### 2. Semantic Errors
- Undefined variable references
- Type mismatches
- Invalid directive usage
- Error messages with context information

### 3. Conversion Errors
- AST construction errors
- Intermediate format generation errors
- Error messages with debug information

## Testing Strategy

### 1. Unit Tests
- Independent testing of each step
- Coverage of edge cases
- Error case verification

### 2. Integration Tests
- Verification of end-to-end processing
- Tests using actual SQL templates
- Performance testing

### 3. Regression Tests
- Output comparison with existing parser
- Reuse of existing test cases
- Bug fix verification

## External Interface

```go
// Main parse function
func Parse(tokens []tokenizer.Token) (any, error)

// Step-by-step parse functions (for debugging)
func ParseStep1(tokens []tokenizer.Token) (*Step1Result, error)
func ParseStep2(result *Step1Result) (*Step2Result, error)
func ParseStep3(result *Step2Result) (*Step3Result, error)
func ParseStep4(result *Step3Result) (*Step4Result, error)
func ParseStep5(result *Step4Result) (*Step5Result, error)
func ParseStep6(result *Step5Result) (*intermediate.Instructions, error)
```

## Type System and Dependencies

### Package Structure

```
parser/
  ├── parsercommon/   # Parser internal common types, functions, utilities
  ├── parserstep1/    # Basic syntax check
  ├── parserstep2/    # SQL grammar check
  ├── parserstep3/    # SnapSQL directive analysis
  ├── parserstep4/    # AST construction
  ├── parserstep5/    # AST optimization
  ├── parserstep6/    # Intermediate format generation
  ├── parser.go       # Public API
  └── errors.go       # Public error definitions
```

### Type Definition Hierarchy

1. **Internal Common Types (parsercommon)**
   ```go
   package parsercommon

   // Basic node type for internal parser use
   type Node struct {
       Type     NodeType
       Children []Node
       // Internal processing fields
   }

   // Parse result type (for external use)
   type ParseResult struct {
       Nodes    []Node
       Metadata ResultMetadata
   }

   // Internal utility types
   type TokenStack struct {
       // Internal implementation details
   }
   ```

2. **Step-Specific Types (parserstepN)**
   ```go
   package parserstep4

   // Step-specific internal types
   type astNode struct {
       parsercommon.Node
       // Step-specific fields
   }

   // Step result type (internal)
   type step4Result struct {
       Nodes []astNode
       // Metadata
   }
   ```

3. **Public Interface (parser)**
   ```go
   package parser

   // Type aliases for external use (minimum necessary)
   type ParseResult = parsercommon.ParseResult
   type ResultMetadata = parsercommon.ResultMetadata

   // New types for external use
   type Options struct {
       // Parse configuration options
   }
   ```

### Dependency Control

1. **Package Reference Rules**
   - `parsercommon`: Provides common functionality (not public)
   - `parserstepN`: Can only reference `parsercommon` (internal packages)
   - `parser`: Provides only public interfaces (type aliases and new type definitions)

2. **Type Visibility Control**
   ```go
   // parsercommon/types.go - Internal common types
   type (
       // For internal processing (private)
       tokenProcessor struct { ... }
       nodeVisitor struct { ... }

       // For result storage (partially public)
       ParseResult struct { ... }
       ResultMetadata struct { ... }
   )

   // parser/types.go - Public interface
   type (
       // Re-export only necessary types
       ParseResult = parsercommon.ParseResult
       ResultMetadata = parsercommon.ResultMetadata
   )
   ```

3. **Error Type Management**
   ```go
   // parsercommon/errors.go - Internal errors
   var (
       errInvalidSyntax = errors.New("invalid syntax")
       errInternalError = errors.New("internal parser error")
   )

   // parser/errors.go - Public errors
   var (
       // Public error types (wrapping internal errors)
       ErrSyntax = fmt.Errorf("syntax error: %w", parsercommon.errInvalidSyntax)
       ErrParse = errors.New("parse error")
   )
   ```

### Package Structure Benefits

1. **Internal Implementation Hiding**
   - parsercommon contains implementation details, not public
   - Only necessary types re-exported in parser package
   - Internal changes don't affect external interface

2. **Maintainability and Extensibility**
   - Common internal processing consolidated in parsercommon
   - Each step can evolve independently
   - Easy addition of new steps

3. **Type Safety**
   - Clear separation of internal and public types
   - Safety through compile-time type checking
   - Interface consistency maintenance

4. **Testing Ease**
   - Easy unit testing of internal processing
   - Simple creation of mocks and stubs
   - External interface stability

## Performance Optimization

1. **Memory Efficiency**
   - Token reuse
   - Reduction of unnecessary copies
   - Memory pool usage

2. **Processing Speed**
   - Early error detection
   - Efficient data structures
   - Cache utilization

3. **Parallel Processing**
   - Parallel parsing of independent files
   - Parallel processing within steps
   - Resource usage optimization
