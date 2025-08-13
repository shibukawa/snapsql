# Template Formats

SnapSQL supports two template formats: **Markdown format** (.snap.md) and **SQL format** (.snap.sql). Both formats implement the 2-way SQL principle where templates work as valid SQL during development.

**Markdown format is the recommended approach** due to its superior documentation capabilities, embedded testing features, and better maintainability.

## Markdown Format (.snap.md) - **Recommended**

The Markdown format provides comprehensive documentation and testing capabilities, making it the preferred choice for most use cases.

### Basic Structure

````markdown
---
function_name: function_name
other_metadata: value
---

# Template Title

## Description
Human readable description of what this query does.

## Parameters
```yaml
param1: type
param2: type
```

## SQL
```sql
SELECT ...
```

## Test Cases (Optional)
### Test: Test case name
**Parameters:**
```yaml
param1: value
param2: value
```

**Expected Results:**
```yaml
- {field1: value1, field2: value2}
```
````

### Key Advantages

✅ **Rich Documentation**: Comprehensive descriptions and explanations  
✅ **Embedded Testing**: Test cases with mock data built into the template  
✅ **Better Maintainability**: Clear structure for complex queries  
✅ **Team Collaboration**: Self-documenting templates  
✅ **Version Control Friendly**: Readable diffs and change tracking  

### Frontmatter

The YAML frontmatter at the top contains metadata:

```yaml
---
function_name: get_user_profile
description: Get user profile with optional sections
response_affinity: one
---
```

### Parameters Section

Parameters are defined in YAML format:

```yaml
## Parameters
```yaml
user_id: int
include_preferences: bool
include_history: bool
date_range:
  start: timestamp
  end: timestamp
```

### SQL Section

The SQL section contains the actual query:

````markdown
## SQL
```sql
SELECT 
    u.id,
    u.name,
    u.email,
    /*# if include_preferences */
    p.theme,
    p.language,
    /*# end */
    /*# if include_history */
    h.last_login,
    h.login_count,
    /*# end */
FROM users u
    /*# if include_preferences */
    LEFT JOIN user_preferences p ON u.id = p.user_id
    /*# end */
    /*# if include_history */
    LEFT JOIN user_history h ON u.id = h.user_id
    /*# end */
WHERE u.id = /*= user_id */1
    /*# if date_range.start != null */
    AND u.created_at >= /*= date_range.start */'2024-01-01'
    /*# end */
    /*# if date_range.end != null */
    AND u.created_at <= /*= date_range.end */'2024-12-31'
    /*# end */;
```
````

### Test Cases Section - **Major Advantage**

Test cases provide executable examples with mock data:

````markdown
## Test Cases

### Test: Basic user profile

**Fixtures (Pre-test Data):**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z"}

user_preferences:
  - {user_id: 1, theme: "dark", language: "en"}
```

**Parameters:**
```yaml
user_id: 1
include_preferences: true
include_history: false
date_range:
  start: null
  end: null
```

**Expected Results:**
```yaml
- {id: 1, name: "John Doe", email: "john@example.com", theme: "dark", language: "en"}
```

### Test: User without preferences

**Parameters:**
```yaml
user_id: 2
include_preferences: true
include_history: false
```

**Expected Results:**
```yaml
- {id: 2, name: "Jane Smith", email: "jane@example.com", theme: null, language: null}
```
````

## SQL Format (.snap.sql) - **Migration Path**

The SQL format is provided primarily for **migrating existing SQL files** to SnapSQL with minimal changes.

### Basic Structure

```sql
/*#
function_name: function_name
description: Human readable description
parameters:
  param1: type
  param2: type
response_affinity: one|many|none
*/
SELECT ...
```

### When to Use SQL Format

Use SQL format (.snap.sql) **only** when:
- ✅ Migrating existing SQL files with minimal changes
- ✅ Very simple queries that don't need documentation
- ✅ Quick prototyping or temporary queries
- ✅ Legacy system integration requirements

### Limitations of SQL Format

❌ **No embedded testing** - Cannot include test cases or mock data  
❌ **Limited documentation** - Only basic metadata in comments  
❌ **Poor maintainability** - Complex queries become hard to understand  
❌ **No structured parameters** - Flat parameter definitions only  

### Metadata Fields

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `function_name` | Yes | Generated function name | `get_user_by_id` |
| `description` | No | Human readable description | `"Find user by ID"` |
| `parameters` | No | Parameter definitions | See below |
| `response_affinity` | No | Return type hint | `one`, `many`, `none` |

### Parameter Types

```sql
/*#
parameters:
  user_id: int                    # Integer
  email: string                   # String
  active: bool                    # Boolean
  score: float                    # Float/Double
  tags: "int[]"                   # Array of integers
  filters: object                 # Object/Map
  created_after: timestamp        # Timestamp/DateTime
  amount: decimal                 # Decimal/BigDecimal
*/
```

### Complete SQL Format Example

```sql
/*#
function_name: find_active_users
description: Find active users with optional email filter
parameters:
  include_email: bool
  email_domain: string
  limit: int
response_affinity: many
*/
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    created_at
FROM users
WHERE active = true
    /*# if email_domain != "" */
    AND email LIKE /*= "%" + email_domain + "%" */'%example.com%'
    /*# end */
ORDER BY created_at DESC
LIMIT /*= limit */10;
```

## Format Comparison

| Feature | **Markdown (.snap.md)** | SQL (.snap.sql) |
|---------|-------------------------|-----------------|
| **Documentation** | ✅ Rich, comprehensive | ❌ Limited to comments |
| **Testing** | ✅ Embedded test cases | ❌ No built-in tests |
| **Mock Data** | ✅ Built-in fixtures | ❌ External only |
| **Maintainability** | ✅ Excellent for complex queries | ❌ Poor for complex logic |
| **Team Collaboration** | ✅ Self-documenting | ❌ Requires external docs |
| **Migration Ease** | ❌ Requires restructuring | ✅ Minimal changes needed |
| **IDE Support** | ✅ Markdown + SQL highlighting | ✅ SQL syntax highlighting |
| **Version Control** | ✅ Very readable diffs | ✅ Clean diffs |

## Migration Strategy

### From Existing SQL to SnapSQL

**Step 1: Quick Migration (SQL Format)**
```sql
-- Original SQL file: get_users.sql
SELECT id, name, email FROM users WHERE active = true;

-- Minimal SnapSQL migration: get_users.snap.sql
/*#
function_name: get_users
response_affinity: many
*/
SELECT id, name, email FROM users WHERE active = true;
```

**Step 2: Enhanced Migration (Markdown Format - Recommended)**
````markdown
---
function_name: get_users
---

# Get Active Users

## Description
Retrieve all active users from the system.

## SQL
```sql
SELECT id, name, email FROM users WHERE active = true;
```

## Test Cases
### Test: Basic active users
**Fixtures:**
```yaml
users:
  - {id: 1, name: "John", email: "john@example.com", active: true}
  - {id: 2, name: "Jane", email: "jane@example.com", active: false}
```

**Expected Results:**
```yaml
- {id: 1, name: "John", email: "john@example.com"}
```
````

## Recommendations

### For New Projects
**Always use Markdown format (.snap.md)** for:
- Better documentation and maintainability
- Built-in testing capabilities
- Team collaboration benefits
- Future-proof template design

### For Existing Projects
1. **Start with SQL format** for quick migration
2. **Gradually convert to Markdown format** for important/complex queries
3. **Prioritize conversion** for queries that need testing or documentation

### File Organization

Both formats support hierarchical organization with `preserve_hierarchy: true`:

```
queries/
├── users/
│   ├── find_user.snap.md          # Recommended
│   └── legacy_query.snap.sql      # Migration path
├── orders/
│   ├── create_order.snap.md       # Recommended
│   └── order_reports.snap.md      # Recommended
└── admin/
    └── system_stats.snap.md       # Recommended
```

Generated files maintain the same structure:

```
generated/
├── users/
│   ├── find_user.json
│   └── legacy_query.json
├── orders/
│   ├── create_order.json
│   └── order_reports.json
└── admin/
    └── system_stats.json
```
