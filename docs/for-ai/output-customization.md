# Output Customization

SnapSQL provides powerful output customization through AS aliases, CAST operations, and structured field naming to control the generated code's type system and data structure.

## Field Aliasing with AS

### Basic Aliasing

```sql
SELECT 
    id,
    name,
    email,
    created_at AS registration_date,
    updated_at AS last_modified
FROM users;
```

Generated Go struct:
```go
type User struct {
    ID               int       `json:"id"`
    Name             string    `json:"name"`
    Email            string    `json:"email"`
    RegistrationDate time.Time `json:"registration_date"`
    LastModified     time.Time `json:"last_modified"`
}
```

### Structured Field Names

Use double underscores (`__`) to create nested structures:

```sql
SELECT 
    u.id,
    u.name,
    u.email,
    d.id AS department__id,
    d.name AS department__name,
    d.budget AS department__budget,
    p.bio AS profile__bio,
    p.avatar_url AS profile__avatar_url
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN profiles p ON u.id = p.user_id;
```

Generated Go struct:
```go
type User struct {
    ID         int         `json:"id"`
    Name       string      `json:"name"`
    Email      string      `json:"email"`
    Department Department  `json:"department"`
    Profile    *Profile    `json:"profile,omitempty"`
}

type Department struct {
    ID     int     `json:"id"`
    Name   string  `json:"name"`
    Budget float64 `json:"budget"`
}

type Profile struct {
    Bio       string `json:"bio"`
    AvatarURL string `json:"avatar_url"`
}
```

### Multi-level Nesting

```sql
SELECT 
    u.id,
    u.name,
    c.id AS company__id,
    c.name AS company__name,
    d.id AS company__department__id,
    d.name AS company__department__name,
    l.city AS company__location__city,
    l.country AS company__location__country
FROM users u
    JOIN companies c ON u.company_id = c.id
    JOIN departments d ON u.department_id = d.id
    JOIN locations l ON c.location_id = l.id;
```

Generated structure:
```go
type User struct {
    ID      int     `json:"id"`
    Name    string  `json:"name"`
    Company Company `json:"company"`
}

type Company struct {
    ID         int        `json:"id"`
    Name       string     `json:"name"`
    Department Department `json:"department"`
    Location   Location   `json:"location"`
}
```

## Type Casting with CAST

### Basic Type Casting

```sql
SELECT 
    id,
    name,
    CAST(score AS DECIMAL(10,2)) AS score,
    CAST(is_active AS BOOLEAN) AS active,
    CAST(metadata AS JSON) AS metadata
FROM users;
```

### Database-Specific Casting

#### PostgreSQL
```sql
SELECT 
    id,
    name,
    score::DECIMAL(10,2) AS score,
    tags::TEXT[] AS tags,
    metadata::JSONB AS metadata,
    created_at::DATE AS registration_date
FROM users;
```

#### MySQL
```sql
SELECT 
    id,
    name,
    CAST(score AS DECIMAL(10,2)) AS score,
    CAST(tags AS JSON) AS tags,
    DATE(created_at) AS registration_date
FROM users;
```

#### SQLite
```sql
SELECT 
    id,
    name,
    CAST(score AS REAL) AS score,
    CAST(is_active AS INTEGER) AS active,
    DATE(created_at) AS registration_date
FROM users;
```

## Type Mapping

### Default Type Mapping

| SQL Type | Go Type | TypeScript Type | Java Type |
|----------|---------|-----------------|-----------|
| `INTEGER`, `INT` | `int` | `number` | `Integer` |
| `BIGINT` | `int64` | `number` | `Long` |
| `DECIMAL`, `NUMERIC` | `decimal.Decimal` | `number` | `BigDecimal` |
| `REAL`, `FLOAT` | `float64` | `number` | `Double` |
| `TEXT`, `VARCHAR` | `string` | `string` | `String` |
| `BOOLEAN` | `bool` | `boolean` | `Boolean` |
| `TIMESTAMP` | `time.Time` | `Date` | `LocalDateTime` |
| `DATE` | `time.Time` | `Date` | `LocalDate` |
| `JSON`, `JSONB` | `map[string]any` | `any` | `Object` |

### Custom Type Hints

Use comments to provide type hints:

```sql
SELECT 
    id,
    name,
    -- @type: decimal
    price,
    -- @type: uuid
    external_id,
    -- @type: enum<active,inactive,pending>
    status
FROM products;
```

## Array and Collection Handling

### Array Fields

```sql
-- PostgreSQL arrays
SELECT 
    id,
    name,
    tags, -- TEXT[]
    scores -- INTEGER[]
FROM users;
```

Generated Go:
```go
type User struct {
    ID     int      `json:"id"`
    Name   string   `json:"name"`
    Tags   []string `json:"tags"`
    Scores []int    `json:"scores"`
}
```

### JSON Arrays

```sql
SELECT 
    id,
    name,
    CAST(preferences AS JSON) AS preferences,
    CAST(history AS JSON) AS history
FROM users;
```

## Nullable Fields

### Handling NULL Values

```sql
SELECT 
    id,                    -- NOT NULL
    name,                  -- NOT NULL
    email,                 -- NULLABLE
    phone,                 -- NULLABLE
    last_login             -- NULLABLE
FROM users;
```

Generated Go with proper null handling:
```go
type User struct {
    ID        int        `json:"id"`
    Name      string     `json:"name"`
    Email     *string    `json:"email,omitempty"`
    Phone     *string    `json:"phone,omitempty"`
    LastLogin *time.Time `json:"last_login,omitempty"`
}
```

### Explicit NULL Handling

```sql
SELECT 
    id,
    name,
    COALESCE(email, '') AS email,           -- Never null
    COALESCE(phone, 'N/A') AS phone,        -- Default value
    NULLIF(description, '') AS description  -- Explicit null
FROM users;
```

## Aggregation and Computed Fields

### Aggregation Functions

```sql
SELECT 
    department_id,
    COUNT(*) AS user_count,
    AVG(salary) AS average_salary,
    MAX(created_at) AS latest_hire,
    MIN(created_at) AS earliest_hire,
    SUM(CASE WHEN active THEN 1 ELSE 0 END) AS active_count
FROM users
GROUP BY department_id;
```

### Window Functions

```sql
SELECT 
    id,
    name,
    salary,
    ROW_NUMBER() OVER (ORDER BY salary DESC) AS salary_rank,
    LAG(salary) OVER (ORDER BY created_at) AS previous_salary,
    AVG(salary) OVER (PARTITION BY department_id) AS dept_avg_salary
FROM users;
```

## Complex Data Structures

### One-to-Many Relationships

```sql
-- Query that returns multiple departments per user
SELECT 
    u.id,
    u.name,
    d.id AS departments__id,
    d.name AS departments__name,
    d.budget AS departments__budget
FROM users u
    JOIN user_departments ud ON u.id = ud.user_id
    JOIN departments d ON ud.department_id = d.id;
```

Generated structure handles arrays automatically:
```go
type User struct {
    ID          int          `json:"id"`
    Name        string       `json:"name"`
    Departments []Department `json:"departments"`
}
```

### Hierarchical Data

```sql
-- Self-referencing hierarchy
SELECT 
    c.id,
    c.name,
    c.parent_id,
    p.id AS parent__id,
    p.name AS parent__name,
    p.parent_id AS parent__parent_id
FROM categories c
    LEFT JOIN categories p ON c.parent_id = p.id;
```

## Custom Serialization

### JSON Field Customization

```sql
SELECT 
    id,
    name,
    -- Custom JSON serialization
    JSON_OBJECT(
        'id', id,
        'display_name', name,
        'is_premium', premium_user
    ) AS user_summary
FROM users;
```

### Computed Fields

```sql
SELECT 
    id,
    first_name,
    last_name,
    CONCAT(first_name, ' ', last_name) AS full_name,
    CASE 
        WHEN age < 18 THEN 'minor'
        WHEN age < 65 THEN 'adult'
        ELSE 'senior'
    END AS age_category,
    EXTRACT(YEAR FROM created_at) AS registration_year
FROM users;
```

## Response Shaping

### Conditional Field Inclusion

```sql
SELECT 
    id,
    name,
    /*# if include_sensitive_data */
    ssn,
    salary,
    /*# end */
    /*# if include_contact_info */
    email,
    phone,
    /*# end */
    created_at
FROM users;
```

### Dynamic Field Selection

```sql
SELECT 
    id,
    name,
    /*# for field : selected_fields */
    /*= field */'email',
    /*# end */
FROM users;
```

## Performance Considerations

### Efficient Joins

```sql
-- Efficient nested structure query
SELECT 
    u.id,
    u.name,
    u.email,
    d.id AS department__id,
    d.name AS department__name,
    m.id AS department__manager__id,
    m.name AS department__manager__name
FROM users u
    JOIN departments d ON u.department_id = d.id
    LEFT JOIN users m ON d.manager_id = m.id
WHERE u.active = true;
```

### Avoiding N+1 Queries

```sql
-- Single query with all needed data
SELECT 
    o.id,
    o.total,
    o.created_at,
    oi.id AS items__id,
    oi.quantity AS items__quantity,
    oi.price AS items__price,
    p.name AS items__product__name,
    p.sku AS items__product__sku
FROM orders o
    JOIN order_items oi ON o.id = oi.order_id
    JOIN products p ON oi.product_id = p.id
WHERE o.user_id = /*= user_id */1;
```

## Best Practices

### 1. Use Meaningful Aliases
```sql
-- ✅ Good: Clear, descriptive aliases
SELECT 
    created_at AS registration_date,
    updated_at AS last_activity

-- ❌ Bad: Cryptic aliases
SELECT 
    created_at AS cr_dt,
    updated_at AS up_dt
```

### 2. Consistent Naming Conventions
```sql
-- ✅ Good: Consistent snake_case
SELECT 
    user_id,
    first_name,
    last_login_date

-- ❌ Bad: Mixed conventions
SELECT 
    userId,
    first_name,
    lastLoginDate
```

### 3. Proper NULL Handling
```sql
-- ✅ Good: Explicit null handling
SELECT 
    id,
    COALESCE(email, '') AS email,
    phone  -- Nullable field

-- ❌ Bad: Unclear null semantics
SELECT 
    id,
    email,
    phone
```

### 4. Type-Safe Casting
```sql
-- ✅ Good: Explicit type casting
SELECT 
    id,
    CAST(score AS DECIMAL(10,2)) AS score,
    CAST(metadata AS JSON) AS metadata

-- ❌ Bad: Implicit type conversion
SELECT 
    id,
    score,
    metadata
```
