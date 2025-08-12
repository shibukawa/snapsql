# Code Generation

SnapSQL generates type-safe code from SQL templates, creating functions, types, and interfaces that provide compile-time safety and runtime efficiency.

## Generated Code Structure

### Go Code Generation

#### Basic Function Generation

From this template:
```sql
/*#
function_name: get_user_by_id
description: Find a user by their ID
parameters:
  user_id: int
response_affinity: one
*/
SELECT id, name, email, created_at
FROM users 
WHERE id = /*= user_id */1;
```

Generates:
```go
// GetUserById finds a user by their ID
func GetUserById(ctx context.Context, db *sql.DB, userId int) (User, error) {
    query := `SELECT id, name, email, created_at FROM users WHERE id = $1`
    
    row := db.QueryRowContext(ctx, query, userId)
    
    var result User
    err := row.Scan(&result.ID, &result.Name, &result.Email, &result.CreatedAt)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return User{}, ErrNotFound
        }
        return User{}, err
    }
    
    return result, nil
}

type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### Iterator-Based Functions (Many Results)

```sql
/*#
function_name: get_active_users
response_affinity: many
*/
SELECT id, name, email FROM users WHERE active = true;
```

Generates:
```go
// GetActiveUsers returns an iterator over active users
func GetActiveUsers(ctx context.Context, db *sql.DB) iter.Seq2[User, error] {
    return func(yield func(User, error) bool) {
        query := `SELECT id, name, email FROM users WHERE active = true`
        
        rows, err := db.QueryContext(ctx, query)
        if err != nil {
            yield(User{}, err)
            return
        }
        defer rows.Close()
        
        for rows.Next() {
            var user User
            err := rows.Scan(&user.ID, &user.Name, &user.Email)
            if !yield(user, err) {
                return
            }
        }
        
        if err := rows.Err(); err != nil {
            yield(User{}, err)
        }
    }
}
```

#### No-Result Functions (DML Operations)

```sql
/*#
function_name: update_user_email
response_affinity: none
*/
UPDATE users 
SET email = /*= email */'new@example.com', updated_at = NOW()
WHERE id = /*= user_id */1;
```

Generates:
```go
// UpdateUserEmail updates a user's email address
func UpdateUserEmail(ctx context.Context, db *sql.DB, email string, userId int) (sql.Result, error) {
    query := `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`
    return db.ExecContext(ctx, query, email, userId)
}
```

### TypeScript Code Generation

#### Interface Generation

```typescript
// Generated TypeScript interfaces
interface User {
    id: number;
    name: string;
    email: string;
    created_at: Date;
}

interface GetUserByIdParams {
    user_id: number;
}

// Generated function signature
export async function getUserById(
    params: GetUserByIdParams,
    client?: DatabaseClient
): Promise<User | null>;

export async function getActiveUsers(
    client?: DatabaseClient
): Promise<User[]>;
```

#### Runtime Implementation

```typescript
// Generated implementation
export async function getUserById(
    params: GetUserByIdParams,
    client: DatabaseClient = defaultClient
): Promise<User | null> {
    const query = `SELECT id, name, email, created_at FROM users WHERE id = $1`;
    const result = await client.query(query, [params.user_id]);
    
    if (result.rows.length === 0) {
        return null;
    }
    
    return {
        id: result.rows[0].id,
        name: result.rows[0].name,
        email: result.rows[0].email,
        created_at: new Date(result.rows[0].created_at)
    };
}
```

## Advanced Type Generation

### Nested Structures

From structured field names:
```sql
SELECT 
    u.id,
    u.name,
    d.id AS department__id,
    d.name AS department__name,
    p.bio AS profile__bio,
    p.avatar_url AS profile__avatar_url
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN profiles p ON u.id = p.user_id;
```

Generates nested Go structs:
```go
type User struct {
    ID         int      `json:"id"`
    Name       string   `json:"name"`
    Department Department `json:"department"`
    Profile    *Profile `json:"profile,omitempty"`
}

type Department struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type Profile struct {
    Bio       string `json:"bio"`
    AvatarURL string `json:"avatar_url"`
}
```

### Array Handling

One-to-many relationships:
```sql
SELECT 
    u.id,
    u.name,
    t.id AS tags__id,
    t.name AS tags__name
FROM users u
    JOIN user_tags ut ON u.id = ut.user_id
    JOIN tags t ON ut.tag_id = t.id;
```

Generates:
```go
type User struct {
    ID   int   `json:"id"`
    Name string `json:"name"`
    Tags []Tag `json:"tags"`
}

type Tag struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
```

### Optional Fields

Nullable database fields become pointer types:
```go
type User struct {
    ID          int        `json:"id"`
    Name        string     `json:"name"`
    Email       *string    `json:"email,omitempty"`       // Nullable
    Phone       *string    `json:"phone,omitempty"`       // Nullable
    LastLogin   *time.Time `json:"last_login,omitempty"`  // Nullable
}
```

## Dynamic Query Generation

### Conditional SQL

Template with conditions:
```sql
/*#
function_name: search_users
parameters:
  filters:
    name: string
    email: string
    active: bool
  sort_by: string
  limit: int
*/
SELECT id, name, email, active
FROM users
WHERE 1=1
    /*# if filters.name != "" */
    AND name ILIKE /*= "%" + filters.name + "%" */'%john%'
    /*# end */
    /*# if filters.email != "" */
    AND email ILIKE /*= "%" + filters.email + "%" */'%@example.com%'
    /*# end */
    /*# if filters.active != null */
    AND active = /*= filters.active */true
    /*# end */
/*# if sort_by == "name" */
ORDER BY name ASC
/*# else */
ORDER BY created_at DESC
/*# end */
LIMIT /*= limit */10;
```

Generates dynamic query builder:
```go
type SearchUsersFilters struct {
    Name   string `json:"name"`
    Email  string `json:"email"`
    Active *bool  `json:"active,omitempty"`
}

type SearchUsersParams struct {
    Filters SearchUsersFilters `json:"filters"`
    SortBy  string            `json:"sort_by"`
    Limit   int               `json:"limit"`
}

func SearchUsers(ctx context.Context, db *sql.DB, params SearchUsersParams) iter.Seq2[User, error] {
    return func(yield func(User, error) bool) {
        var queryBuilder strings.Builder
        var args []interface{}
        argIndex := 1
        
        queryBuilder.WriteString("SELECT id, name, email, active FROM users WHERE 1=1")
        
        if params.Filters.Name != "" {
            queryBuilder.WriteString(fmt.Sprintf(" AND name ILIKE $%d", argIndex))
            args = append(args, "%"+params.Filters.Name+"%")
            argIndex++
        }
        
        if params.Filters.Email != "" {
            queryBuilder.WriteString(fmt.Sprintf(" AND email ILIKE $%d", argIndex))
            args = append(args, "%"+params.Filters.Email+"%")
            argIndex++
        }
        
        if params.Filters.Active != nil {
            queryBuilder.WriteString(fmt.Sprintf(" AND active = $%d", argIndex))
            args = append(args, *params.Filters.Active)
            argIndex++
        }
        
        if params.SortBy == "name" {
            queryBuilder.WriteString(" ORDER BY name ASC")
        } else {
            queryBuilder.WriteString(" ORDER BY created_at DESC")
        }
        
        queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
        args = append(args, params.Limit)
        
        query := queryBuilder.String()
        rows, err := db.QueryContext(ctx, query, args...)
        if err != nil {
            yield(User{}, err)
            return
        }
        defer rows.Close()
        
        for rows.Next() {
            var user User
            err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Active)
            if !yield(user, err) {
                return
            }
        }
        
        if err := rows.Err(); err != nil {
            yield(User{}, err)
        }
    }
}
```

## Mock Integration

### Context-Based Mocking

Generated functions support mock data through context:

```go
// Production usage
users := make([]User, 0)
for user, err := range GetActiveUsers(ctx, db) {
    if err != nil {
        return err
    }
    users = append(users, user)
}

// Test usage with mock data
ctx := snapsqlgo.WithConfig(context.Background(), "get_active_users", 
    snapsqlgo.WithMockData("test_case_name"))

users := make([]User, 0)
for user, err := range GetActiveUsers(ctx, nil) { // db can be nil with mock
    if err != nil {
        return err
    }
    users = append(users, user)
}
```

### Mock Data Loading

```go
// Generated mock data loader
func loadMockData(testCaseName string) ([]User, error) {
    // Load from embedded test data or external files
    switch testCaseName {
    case "active_users_basic":
        return []User{
            {ID: 1, Name: "John Doe", Email: "john@example.com", Active: true},
            {ID: 2, Name: "Jane Smith", Email: "jane@example.com", Active: true},
        }, nil
    case "empty_result":
        return []User{}, nil
    default:
        return nil, fmt.Errorf("unknown test case: %s", testCaseName)
    }
}
```

## Configuration Options

### Generator Settings

```yaml
# snapsql.yaml
generation:
  generators:
    go:
      output: "./internal/queries"
      enabled: true
      preserve_hierarchy: true
      settings:
        package: "queries"
        generate_tests: true
        mock_path: "./testdata/mocks"
        
    typescript:
      output: "./src/generated"
      enabled: true
      settings:
        types: true
        runtime: "node-postgres"
        export_style: "named"
```

### Package Structure

Generated Go code structure:
```
internal/queries/
├── users/
│   ├── get_user_by_id.go
│   ├── search_users.go
│   └── types.go
├── orders/
│   ├── create_order.go
│   ├── get_order_history.go
│   └── types.go
└── common/
    ├── errors.go
    └── interfaces.go
```

## Error Handling

### Generated Error Types

```go
// Common error types
var (
    ErrNotFound     = errors.New("record not found")
    ErrInvalidInput = errors.New("invalid input parameters")
    ErrDatabase     = errors.New("database error")
)

// Query-specific error handling
func GetUserById(ctx context.Context, db *sql.DB, userId int) (User, error) {
    if userId <= 0 {
        return User{}, fmt.Errorf("%w: user_id must be positive", ErrInvalidInput)
    }
    
    // ... query execution
    
    if errors.Is(err, sql.ErrNoRows) {
        return User{}, ErrNotFound
    }
    
    return result, nil
}
```

### Validation

```go
// Generated parameter validation
type SearchUsersParams struct {
    Filters SearchUsersFilters `json:"filters"`
    SortBy  string            `json:"sort_by" validate:"oneof=name email created_at"`
    Limit   int               `json:"limit" validate:"min=1,max=1000"`
}

func (p SearchUsersParams) Validate() error {
    if p.Limit <= 0 || p.Limit > 1000 {
        return fmt.Errorf("%w: limit must be between 1 and 1000", ErrInvalidInput)
    }
    
    validSortFields := []string{"name", "email", "created_at"}
    if p.SortBy != "" && !contains(validSortFields, p.SortBy) {
        return fmt.Errorf("%w: invalid sort_by field", ErrInvalidInput)
    }
    
    return nil
}
```

## Performance Optimizations

### Connection Pooling

```go
// Generated code supports connection interfaces
type DatabaseExecutor interface {
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Works with *sql.DB, *sql.Tx, or custom implementations
func GetUserById(ctx context.Context, exec DatabaseExecutor, userId int) (User, error) {
    // Implementation uses exec instead of db directly
}
```

### Prepared Statements

```go
// Generated prepared statement support
type UserQueries struct {
    getUserById *sql.Stmt
    searchUsers *sql.Stmt
}

func NewUserQueries(db *sql.DB) (*UserQueries, error) {
    getUserById, err := db.Prepare("SELECT id, name, email FROM users WHERE id = $1")
    if err != nil {
        return nil, err
    }
    
    return &UserQueries{
        getUserById: getUserById,
    }, nil
}

func (q *UserQueries) GetUserById(ctx context.Context, userId int) (User, error) {
    row := q.getUserById.QueryRowContext(ctx, userId)
    // ... scan logic
}
```

## Best Practices

### 1. Consistent Function Naming
```go
// ✅ Good: Consistent naming convention
func GetUserById(ctx context.Context, db *sql.DB, userId int) (User, error)
func GetActiveUsers(ctx context.Context, db *sql.DB) iter.Seq2[User, error]
func CreateUser(ctx context.Context, db *sql.DB, user CreateUserParams) (sql.Result, error)

// ❌ Bad: Inconsistent naming
func getUserById(ctx context.Context, db *sql.DB, userId int) (User, error)
func ListActiveUsers(ctx context.Context, db *sql.DB) iter.Seq2[User, error]
func InsertUser(ctx context.Context, db *sql.DB, user CreateUserParams) (sql.Result, error)
```

### 2. Proper Error Handling
```go
// ✅ Good: Specific error types
if errors.Is(err, sql.ErrNoRows) {
    return User{}, ErrNotFound
}

// ❌ Bad: Generic error handling
if err != nil {
    return User{}, err
}
```

### 3. Context Usage
```go
// ✅ Good: Always use context
func GetUserById(ctx context.Context, db *sql.DB, userId int) (User, error)

// ❌ Bad: Missing context
func GetUserById(db *sql.DB, userId int) (User, error)
```
