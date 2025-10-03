# Development Guide

This guide covers how to contribute to SnapSQL development.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Git
- Make (optional, for build scripts)
- Docker (for testing with databases)

### Clone Repository

```bash
git clone https://github.com/shibukawa/snapsql.git
cd snapsql
```

### Build from Source

```bash
# Build the CLI tool
go build ./cmd/snapsql

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Run linter
golangci-lint run
```

### Project Structure

```
snapsql/
├── cmd/
│   └── snapsql/           # CLI tool source code
├── runtime/
│   ├── snapsqlgo/         # Go runtime library
│   ├── python/            # Python runtime (planned)
│   ├── node/              # Node.js runtime (planned)
│   └── java/              # Java runtime (planned)
├── examples/              # Example projects
├── testdata/              # Test data files
├── contrib/               # Community contributions
├── docs/                  # Documentation
├── intermediate/          # Intermediate format processing
├── query/                 # Query execution engine
├── parser/                # SQL parser
└── template/              # Template engine
```

## Coding Standards

### Go Guidelines

- Follow standard Go conventions
- Use Go 1.24 features
- Prefer `any` over `interface{}`
- Use `slices` and `maps` packages where appropriate
- Use generics when explicitly requested
- No backward compatibility copies of functionality

### Error Handling

- Use sentinel errors for parameterless errors:
  ```go
  var ErrTemplateNotFound = errors.New("template not found")
  ```
- Wrap errors with context:
  ```go
  return fmt.Errorf("failed to parse template: %w", err)
  ```

### Testing

- Use `github.com/alecthomas/assert/v2` for assertions
- Use `testing.T.Context()` for context in tests
- Use TestContainers for database tests
- Test file naming: `*_test.go`

### Dependencies

**Approved Libraries:**
- Web routing: `net/http` ServeMux
- CLI parsing: `github.com/alecthomas/kong`
- Colors: `github.com/fatih/color`
- YAML: `github.com/goccy/go-yaml`
- Expressions: `github.com/google/cel-go`
- Markdown: `github.com/yuin/goldmark`
- PostgreSQL: `github.com/jackc/pgx/v5`
- MySQL: `github.com/go-sql-driver/mysql`
- SQLite: `github.com/mattn/go-sqlite3`

## Development Workflow

### 1. Feature Development

All work must go through the TODO list management system:

1. **Check `docs/TODO.md`** before starting any work
2. **Add new tasks** to TODO.md with priority and phase
3. **Follow phases**:
   - Phase 1: Information gathering and design documents
   - Phase 2: Unit test creation (tests should fail initially)
   - Phase 3: Source code modification and testing
   - Phase 4: Refactoring proposal
   - Phase 5: Refactoring implementation

### 2. Design Documents

For features over 50 lines or major refactoring:
- Create `docs/designdocs/{YYYYMMDD}-{feature-name}.ja.md` in Japanese
- Include backward compatibility considerations
- Ask questions if requirements are unclear

### 3. Code Changes

After implementation, if new libraries or coding styles are introduced:
- Add comments to `docs/coding-standard-suggest.md`
- Include date and related task information

### 4. Git Workflow

```bash
# Create feature branch
git checkout -b feature/new-feature

# Make changes following the phases
# ... development work ...

# Commit with descriptive messages
git commit -m "feat: add dry-run support for query command"

# Push and create PR
git push origin feature/new-feature
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./cmd/snapsql

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### Integration Tests

```bash
# Start test databases with Docker
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./...

# Clean up
docker-compose -f docker-compose.test.yml down
```

### Test Database Setup

Using TestContainers for database tests:

```go
func TestWithPostgreSQL(t *testing.T) {
    ctx := t.Context()
    
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    assert.NoError(t, err)
    defer container.Terminate(ctx)
    
    connStr, err := container.ConnectionString(ctx)
    assert.NoError(t, err)
    
    // Use connStr for testing
}
```

## Architecture

### Parser Combinator System

SnapSQL uses a custom parser combinator library for SQL parsing:

```go
// Basic parser structure
type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)

// AST node interface
type ASTNode interface {
    Type() string
    Position() *pc.Pos
}
```

See `docs/.amazonq/rules/parsercombinator.md` for detailed usage.

### Template Engine

The template engine processes 2-way SQL templates:

1. **Lexical Analysis**: Tokenize SQL with template directives
2. **Parsing**: Build AST from tokens
3. **Intermediate Generation**: Create execution instructions
4. **Runtime Execution**: Generate final SQL with parameters

### Query Execution

The query execution engine:

1. **Template Processing**: Convert templates to executable SQL
2. **Parameter Binding**: Safely bind parameters to prevent injection
3. **Database Execution**: Execute queries with appropriate drivers
4. **Result Formatting**: Format results in various output formats

## Contributing

### Pull Request Process

1. **Fork the repository**
2. **Create feature branch** from `main`
3. **Follow development workflow** (TODO.md management)
4. **Write tests** for new functionality
5. **Update documentation** as needed
6. **Submit pull request** with clear description

### PR Requirements

- [ ] All tests pass
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] TODO.md updated if applicable
- [ ] No breaking changes without discussion

### Code Review

- Focus on correctness and maintainability
- Check for security issues (especially SQL injection prevention)
- Verify test coverage
- Ensure documentation accuracy

## Release Process

### Version Management

SnapSQL uses semantic versioning:
- `MAJOR.MINOR.PATCH`
- Major: Breaking changes
- Minor: New features (backward compatible)
- Patch: Bug fixes

### Release Steps

1. **Update version** in relevant files
2. **Update CHANGELOG.md**
3. **Create release tag**
4. **Build and publish binaries**
5. **Update documentation**

## Debugging

### Enable Debug Logging

```bash
# Verbose output
snapsql --verbose query template.snap.sql

# Debug parser
export SNAPSQL_DEBUG_PARSER=true
snapsql generate
```

Run generation for the Kanban sample from inside `examples/kanban`:

```bash
cd examples/kanban
snapsql generate
```

### Common Debug Scenarios

1. **Template parsing issues**:
   ```bash
   snapsql validate --strict template.snap.sql
   ```

2. **Parameter binding problems**:
   ```bash
   snapsql query template.snap.sql --dry-run --params-file params.json
   ```

3. **Database connection issues**:
   ```bash
   snapsql config test-db
   ```

## Performance Considerations

### Parser Performance

- Use efficient parser combinators
- Minimize backtracking
- Cache parsed results when possible

### Query Performance

- Validate generated SQL for efficiency
- Monitor query execution times
- Provide query analysis tools

### Memory Usage

- Stream large result sets
- Avoid loading entire datasets into memory
- Use connection pooling appropriately

## Security

### SQL Injection Prevention

- All parameter substitution must be safe
- Validate template modifications
- Use parameterized queries
- Sanitize user inputs

### Access Control

- Validate database permissions
- Implement query restrictions
- Audit dangerous operations

## Documentation

### Writing Documentation

- Use clear, concise language
- Provide practical examples
- Keep documentation up to date
- Include troubleshooting sections

### Documentation Structure

- `README.md`: Project overview and quick start
- `docs/`: Detailed documentation
- `examples/`: Working examples
- Code comments: Implementation details

## Getting Help

- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: General questions and ideas
- **Code Review**: Implementation feedback
- **Documentation**: Clarifications and improvements
