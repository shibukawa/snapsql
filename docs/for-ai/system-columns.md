# System Columns Implementation

System columns are automatically managed database fields that are commonly used across applications, such as `created_at`, `updated_at`, `created_by`, and `version`. SnapSQL provides built-in support for automatic system column handling.

## Overview

System columns eliminate boilerplate code by automatically:
- Adding system columns to INSERT and UPDATE statements
- Providing default values from configuration
- Extracting values from application context
- Generating type-safe code with proper parameter handling

## Configuration

System columns are defined in the `snapsql.yaml` configuration file:

```yaml
dialect: postgres

system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
    - name: updated_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
      on_update:
        parameter: implicit
        default: "NOW()"
    - name: created_by
      type: int
      on_insert:
        parameter: implicit
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1
      on_update:
        parameter: implicit
```

### Configuration Fields

- **`name`**: Column name in the database
- **`type`**: Data type (`timestamp`, `int`, `string`, etc.)
- **`on_insert`**: Behavior during INSERT operations
- **`on_update`**: Behavior during UPDATE operations
- **`parameter`**: How the value is provided:
  - `implicit`: Automatically handled by the runtime
  - `explicit`: Must be provided as a function parameter
- **`default`**: Default value when not provided in context

## Template Usage

### Basic INSERT with System Columns

```sql
/*#
function_name: create_user
parameters:
  name: string
  email: string
*/
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com')
```

SnapSQL automatically expands this to:

```sql
INSERT INTO users (name, email, created_at, updated_at, created_by, version) 
VALUES ($1, $2, $3, $4, $5, $6)
```

### UPDATE with System Columns

```sql
/*#
function_name: update_user
parameters:
  id: int
  name: string
  email: string
*/
UPDATE users 
SET name = /*= name */'John', email = /*= email */'john@example.com'
WHERE id = /*= id */1
```

SnapSQL automatically adds system columns:

```sql
UPDATE users 
SET name = $1, email = $2, updated_at = $3, version = $4
WHERE id = $5
```

## Generated Code

### Go Runtime Integration

The generated Go code includes automatic system column handling:

```go
// Generated function signature
func CreateUser(ctx context.Context, executor snapsqlgo.DBExecutor, name string, email string, opts ...snapsqlgo.FuncOpt) (sql.Result, error) {
    // Extract implicit parameters from context
    implicitSpecs := []snapsqlgo.ImplicitParamSpec{
        {Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
        {Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
        {Name: "created_by", Type: "int", Required: true},
        {Name: "version", Type: "int", Required: false, DefaultValue: 1},
    }
    systemValues := snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)
    
    // Build SQL with system values
    query := "INSERT INTO users (name, email, created_at, updated_at, created_by, version) VALUES ($1, $2, $3, $4, $5, $6)"
    args := []any{
        name,
        email,
        systemValues["created_at"],
        systemValues["updated_at"],
        systemValues["created_by"],
        systemValues["version"],
    }
    
    // Execute query...
}
```

### Context-Based Values

System values can be provided through context:

```go
// Set system values in context
ctx = snapsqlgo.WithSystemValue(ctx, "created_by", 123)
ctx = snapsqlgo.WithSystemValue(ctx, "created_at", time.Now())

// Call generated function
result, err := CreateUser(ctx, db, "John Doe", "john@example.com")
```

## Default Value Processing

### Configuration-Based Defaults

Default values are processed at code generation time:

```yaml
system:
  fields:
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1  # Becomes: DefaultValue: 1
    - name: created_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"  # Becomes: DefaultValue: time.Now()
```

### Runtime Behavior

1. **Context Check**: First, check if value exists in context
2. **Default Fallback**: If not in context, use configured default value
3. **Required Validation**: Panic if required field is missing and has no default

```go
// Runtime logic
func ExtractImplicitParams(ctx context.Context, specs []ImplicitParamSpec) map[string]any {
    result := make(map[string]any)
    
    for _, spec := range specs {
        // Check context first
        if value := GetSystemValueFromContext(ctx, spec.Name); value != nil {
            result[spec.Name] = value
            continue
        }
        
        // Use default value if available
        if spec.DefaultValue != nil {
            result[spec.Name] = spec.DefaultValue
            continue
        }
        
        // Panic if required field is missing
        if spec.Required {
            panic(fmt.Sprintf("required system field '%s' not found in context", spec.Name))
        }
        
        // Optional field without default remains nil
        result[spec.Name] = nil
    }
    
    return result
}
```

## Advanced Features

### Conditional System Columns

System columns can be conditionally applied based on statement type:

```yaml
system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:  # Only applied to INSERT statements
        parameter: implicit
        default: "NOW()"
    - name: updated_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
      on_update:  # Applied to both INSERT and UPDATE
        parameter: implicit
        default: "NOW()"
```

### Explicit System Columns

For cases where system columns need to be explicitly provided:

```yaml
system:
  fields:
    - name: created_by
      type: int
      on_insert:
        parameter: explicit  # Must be provided as function parameter
```

Generated function signature includes explicit parameters:

```go
func CreateUser(ctx context.Context, executor snapsqlgo.DBExecutor, name string, email string, createdBy int) (sql.Result, error)
```

### Mock Data Integration

System columns work seamlessly with mock data:

```go
// Test with mock data
ctx := snapsqlgo.WithConfig(context.Background(), "create_user", 
    snapsqlgo.WithMockData("test_case_1"))

// System values are automatically handled in mock mode
result, err := CreateUser(ctx, nil, "John Doe", "john@example.com")
```

## Best Practices

### 1. Consistent Configuration

Use consistent system column definitions across your application:

```yaml
# Standard system columns for all projects
system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
    - name: updated_at
      type: timestamp
      on_insert:
        parameter: implicit
        default: "NOW()"
      on_update:
        parameter: implicit
        default: "NOW()"
    - name: created_by
      type: int
      on_insert:
        parameter: implicit
    - name: version
      type: int
      on_insert:
        parameter: implicit
        default: 1
      on_update:
        parameter: implicit
```

### 2. Context Management

Set system values at the application boundary:

```go
// Middleware or request handler
func withUserContext(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserIDFromToken(r)
        ctx := snapsqlgo.WithSystemValue(r.Context(), "created_by", userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 3. Testing Strategy

Use mock data for system columns in tests:

```markdown
## Test Cases

### Test: Create user with system columns

**Parameters:**
```yaml
name: "John Doe"
email: "john@example.com"
```

**Expected Results:**
```yaml
- {id: 1, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z", version: 1}
```
```

### 4. Migration Considerations

When adding system columns to existing tables:

```sql
-- Add system columns with appropriate defaults
ALTER TABLE users ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN created_by INTEGER;
ALTER TABLE users ADD COLUMN version INTEGER DEFAULT 1;
```

## Troubleshooting

### Common Issues

1. **Missing Required System Value**
   ```
   Error: required system field 'created_by' not found in context
   ```
   Solution: Set the value in context or make it optional with a default

2. **Type Mismatch**
   ```
   Error: cannot convert string to int for system field 'created_by'
   ```
   Solution: Ensure context values match the configured type

3. **System Column Not Added**
   - Check that the statement type matches the configuration (`on_insert`, `on_update`)
   - Verify the system field configuration is correct
   - Ensure the template is being processed with the correct configuration file

### Debug Tips

1. **Check Generated Code**: Review the generated Go code to see how system columns are handled
2. **Verify Configuration**: Ensure `snapsql.yaml` is in the correct location and properly formatted
3. **Test Context Values**: Use logging to verify context values are set correctly
4. **Mock Data Testing**: Use mock data to isolate system column behavior from database interactions

## Integration Examples

### Web Application

```go
// User service with system columns
type UserService struct {
    db *sql.DB
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
    // System columns (created_at, created_by, version) are automatically handled
    result, err := CreateUser(ctx, s.db, name, email)
    if err != nil {
        return nil, err
    }
    
    // Get the created user ID and return full user object
    userID, _ := result.LastInsertId()
    return s.GetUserByID(ctx, int(userID))
}
```

### Background Jobs

```go
// Background job with system context
func processUserUpdates(ctx context.Context, db *sql.DB) {
    // Set system user for background processes
    ctx = snapsqlgo.WithSystemValue(ctx, "created_by", SYSTEM_USER_ID)
    
    // All database operations will use system user
    for _, update := range pendingUpdates {
        _, err := UpdateUser(ctx, db, update.ID, update.Name, update.Email)
        if err != nil {
            log.Printf("Failed to update user %d: %v", update.ID, err)
        }
    }
}
```

This system column implementation provides a clean, type-safe way to handle common database patterns while maintaining the 2-way SQL principle and supporting comprehensive testing strategies.
