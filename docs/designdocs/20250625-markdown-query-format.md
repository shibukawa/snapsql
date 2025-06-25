# Markdown Query Definition Format

## Overview

SnapSQL supports literate programming through Markdown-based query definition files (`.snap.md`). This format combines SQL templates with comprehensive documentation, test cases, and metadata in a single, readable document.

## File Structure

### Basic Structure

```markdown
# Query Name (任意の言語でタイトル)

## Overview (概要)
Query purpose and description

## Parameters (パラメータ)
Input parameter definitions in YAML format

## SQL (SQL)
SQL template in SnapSQL format

## Test Cases (テストケース)
Parameter sets and expected results

## Mock Data (モックデータ) [Optional]
Mock data definitions for testing
```

### Extended Sections (Optional)

```markdown
## Performance (パフォーマンス) [Optional]
Performance considerations and requirements

## Security (セキュリティ) [Optional]
Security considerations and access control

## Change Log (変更履歴) [Optional]
Version history and changes
```

## Internationalization

### Heading Format

All section headings use English keywords. Additional text after the keyword is ignored during processing:

```markdown
## English Keyword (any additional text is ignored)
```

Examples:
- `## Overview` - Processed
- `## Overview (概要)` - Processed (Japanese text ignored)
- `## Parameters something else` - Processed (additional text ignored)
- `## SQL Template` - Processed (additional text ignored)

### Supported Headings

| English Keyword | Purpose | Required | Processing |
|----------------|---------|----------|------------|
| `Overview` | Query description and purpose | Yes | Content processed |
| `Parameters` | Input parameter definitions | Yes | Parsed as YAML |
| `SQL` | SQL template | Yes | Processed as SnapSQL |
| `Test Cases` | Test scenarios | Yes | Parsed as dbtestify format |
| `Mock Data` | Mock data for testing | No | Parsed as YAML |
| `Performance` | Performance information | No | Content processed |
| `Security` | Security considerations | No | Content processed |
| `Change Log` | Version history | No | Content processed |

### Other Sections

Any section not listed above is ignored during processing. You can add custom sections for documentation purposes:

```markdown
## Implementation Notes
This section will be ignored during code generation.

## Database Schema Requirements  
This section will also be ignored.

## Custom Documentation
Any custom section is allowed but ignored.
```

## Complete Example

```markdown
---
name: "user search"
dialect: "postgres"
---

# User Search Query

## Overview

Searches for active users based on various criteria with pagination support.
Supports department filtering and sorting functionality.

## Parameters

```yaml
user_id: int
filters:
  active: bool
  departments: [str]
  name_pattern: str
pagination:
  limit: int
  offset: int
sort_by: str
include_email: bool
table_suffix: str
```

## SQL

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    department,
    created_at
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
    /*# if filters.name_pattern */
    AND name ILIKE /*= filters.name_pattern */'%john%'
    /*# end */
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'engineering', 'design')
    /*# end */
/*# if sort_by */
ORDER BY /*= sort_by */created_at DESC
/*# end */
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

## Test Cases

### Case 1: Basic Search

**Fixture:**
```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    department: "marketing"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```

**Parameters:**
```yaml
user_id: 123
filters:
  active: true
  departments: ["engineering", "design"]
  name_pattern: null
pagination:
  limit: 20
  offset: 0
sort_by: "name"
include_email: false
table_suffix: "test"
```

**Expected Result:**
```yaml
- id: 1
  name: "John Doe"
  department: "engineering"
  created_at: "2024-01-15T10:30:00Z"
- id: 2
  name: "Jane Smith"
  department: "design"
  created_at: "2024-02-20T14:45:00Z"
```

### Case 2: Full Options with Email

**Fixture:**
```yaml
users:
  - id: 4
    name: "Alice Smith"
    email: "alice@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-10T08:00:00Z"
  - id: 5
    name: "Charlie Smith"
    email: "charlie@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-20T09:00:00Z"
```

**Parameters:**
```yaml
user_id: 456
filters:
  active: true
  departments: ["marketing"]
  name_pattern: "%smith%"
pagination:
  limit: 5
  offset: 0
sort_by: "created_at DESC"
include_email: true
table_suffix: "test"
```

**Expected Result:**
```yaml
- id: 5
  name: "Charlie Smith"
  email: "charlie@example.com"
  department: "marketing"
  created_at: "2024-01-20T09:00:00Z"
- id: 4
  name: "Alice Smith"
  email: "alice@example.com"
  department: "marketing"
  created_at: "2024-01-10T08:00:00Z"
```

### Case 3: Empty Result

**Fixture:**
```yaml
users:
  - id: 6
    name: "David Wilson"
    email: "david@example.com"
    department: "hr"
    active: false
    created_at: "2024-01-05T10:00:00Z"
```

**Parameters:**
```yaml
user_id: 789
filters:
  active: true
  departments: ["engineering"]
  name_pattern: null
pagination:
  limit: 10
  offset: 0
sort_by: null
include_email: false
table_suffix: "test"
```

**Expected Result:**
```yaml
[]
```

## Mock Data

```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    department: "marketing"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```

## Performance

### Index Requirements
- `users.active` - Required
- `users.department` - Recommended
- `users.name` - For LIKE searches

### Estimated Execution Time
- Small scale (< 10K rows): < 10ms
- Medium scale (< 100K rows): < 50ms  
- Large scale (> 100K rows): < 200ms

## Security

### Access Control
- Administrators: Can search all departments
- Regular users: Can only search their own department

### Data Masking
- `email` field is masked based on permissions

## Change Log

### v1.2.0 (2024-03-01)
- Added `name_pattern` parameter
- Support for ILIKE search

### v1.1.0 (2024-02-15)
- Added `include_email` option
- Performance optimization

### v1.0.0 (2024-01-01)
- Initial version
```

## File Organization

### Directory Structure

```
queries/
├── users/
│   ├── search.md          # User search
│   ├── create.md          # User creation
│   └── update.md          # User update
├── posts/
│   ├── list.md            # Post listing
│   ├── detail.md          # Post details
│   └── search.md          # Post search
└── analytics/
    ├── user-stats.md      # User statistics
    └── daily-report.md    # Daily reports
```

### Naming Conventions

- Use kebab-case for file names
- Group related queries in subdirectories
- Use descriptive names that reflect the query purpose

## Front Matter

### Required Fields

```yaml
---
name: "user search"
dialect: "postgres"
---
```

### Field Descriptions

- `name`: Function name for generated code (space-separated words, converted to appropriate naming convention for each language)
- `dialect`: SQL dialect (`postgres`, `mysql`, `sqlite`)

### Examples

```yaml
---
name: "get user by id"
dialect: "postgres"
---
```

```yaml
---
name: "list active posts"
dialect: "mysql"
---
```

```yaml
---
name: "analytics daily report"
dialect: "sqlite"
---
```

## Processing Rules

### Heading Recognition

1. Parser looks for English keywords at the beginning of headings
2. Any text after the keyword is ignored during processing
3. Content under each recognized heading is processed according to its type
4. Unrecognized headings and their content are ignored

### Content Processing

- **Parameters**: Parsed as YAML
- **SQL**: Processed as SnapSQL template
- **Test Cases**: Parsed as dbtestify format with YAML fixtures, parameters, and expected results
- **Mock Data**: Parsed as YAML for development and testing

### Validation Rules

1. Required sections must be present
2. Parameters section must be valid YAML
3. SQL section must be valid SnapSQL syntax
4. Test cases must follow dbtestify format with valid YAML fixtures, parameters, and expected results
5. Test case parameters must match the Parameters section structure

## Integration with SnapSQL CLI

### File Discovery

```bash
# Process all .snap.md files
snapsql generate -i ./queries

# Process specific file
snapsql generate queries/users/search.md
```

### Output Generation

Each `.snap.md` file generates:
- Intermediate JSON with parsed content
- Language-specific code (if requested)
- dbtestify-compatible test files with fixtures and expected results

### Validation

```bash
# Validate markdown query files
snapsql validate queries/users/search.md

# Validate all markdown files
snapsql validate -i ./queries --format json
```

## Benefits

1. **Literate Programming**: Combines code and documentation
2. **Internationalization**: Supports multiple languages
3. **Testability**: Integrated test cases with dbtestify format for database testing
4. **Maintainability**: Version history and change tracking
5. **IDE Support**: Markdown syntax highlighting and preview
6. **Collaboration**: Easy to review and comment on
7. **Database Testing**: Full database integration testing with fixtures and expected results
