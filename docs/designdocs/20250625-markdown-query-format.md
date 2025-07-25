# Markdown Query Definition Format

## Overview

SnapSQL supports literate programming through Markdown-based query definition files (`.snap.md`). This format integrates SQL templates with comprehensive documentation, test cases, and metadata into a single, readable document.

## File Structure

### Basic Structure

```markdown
---
function_name: "getUserData"
description: "Get user data"
---

# Query Name (title in any language)

## Description
Purpose and explanation of the query

## Parameters [Optional]
Input parameter definitions in YAML/JSON/text format

## SQL
SnapSQL format SQL template

## Test Cases [Optional]
Parameter sets and expected results

## Mock Data [Optional]
Mock data definitions for testing
```

### Extended Sections (Optional)

```markdown
## Performance [Optional]
Performance considerations and requirements

## Security [Optional]
Security considerations and access control

## Change Log [Optional]
Version history and changes
```

## Section Specifications

### Required Sections

| Section | Alternative Names | Purpose | Required |
|---------|-------------------|---------|----------|
| `Description` | `Overview` | Query description and purpose | Yes |
| `SQL` | - | SQL template | Yes |

### Optional Sections

| Section | Alternative Names | Purpose | Format |
|---------|-------------------|---------|--------|
| `Parameters` | `Params`, `Parameter` | Input parameter definitions | YAML/JSON/Text |
| `Test Cases` | `Tests`, `TestCases` | Test scenarios | YAML/JSON/CSV/XML/List |
| `Mock Data` | `Mocks`, `TestData`, `MockData` | Mock data for testing | YAML/JSON/CSV/XML/Markdown Table |

## Front Matter

### Basic Fields

```yaml
---
function_name: "getUserData"  # Generated function name
description: "Get user data"  # Query description
---
```

### Field Descriptions

- `function_name`: Function name for generated code (camelCase recommended)
- `description`: Concise description of the query
- Additional custom fields can be added

### Examples

```yaml
---
function_name: "getUserById"
description: "Retrieve user information by ID"
version: "1.0.0"
author: "Development Team"
---
```

## Parameters Section

### YAML Format

```yaml
user_id: int
include_email: bool
filters:
  active: bool
  departments: [str]
pagination:
  limit: int
  offset: int
```

### JSON Format

```json
{
  "user_id": "int",
  "include_email": "bool",
  "filters": {
    "active": "bool",
    "departments": ["string"]
  },
  "pagination": {
    "limit": "int",
    "offset": "int"
  }
}
```

### Text Format

```markdown
- user_id (int): The ID of the user to query
- include_email (bool): Whether to include email in results
- status (string): Filter by user status (active, inactive, pending)
- limit (int): Maximum number of results to return
- offset (int): Number of results to skip

Additional notes:
- All parameters are optional except user_id
- Default limit is 10 if not specified
```

### Mixed Format

```markdown
This query accepts the following parameters:

```yaml
user_id: int
include_email: bool
```

Additional parameter notes:
- user_id is required
- include_email defaults to false

```json
{
  "filters": {
    "status": "string",
    "department": "string"
  }
}
```
```

## Test Cases Section

### YAML Format

```yaml
parameters:
  user_id: 123
  include_email: true
expected:
  status: "success"
  count: 1
```

### CSV Format

```csv
user_id,active,expected
123,true,"user found"
456,false,"user inactive"
999,true,"user not found"
```

### List Format

```markdown
### Test Case 1: Basic Query
- Input: user_id = 123, include_email = true
- Expected: Returns user data with email

### Test Case 2: User Not Found
- Parameters: user_id = 999, active = true
- Expected: No results returned
```

### XML Format (DBUnit Compatible)

```xml
<dataset>
  <test_case>
    <parameters user_id="123" include_email="true"/>
    <expected status="success" count="1"/>
  </test_case>
</dataset>
```

## Mock Data Section

### YAML Format

```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    active: false
orders:
  - id: 101
    user_id: 1
    total: 99.99
    status: "completed"
```

### CSV Format

```csv
# users
id,name,email,active
1,"John Doe","john@example.com",true
2,"Jane Smith","jane@example.com",false
3,"Bob Wilson","bob@example.com",true
```

### XML Format (DBUnit Compatible)

```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" active="true"/>
  <users id="2" name="Jane Smith" email="jane@example.com" active="false"/>
  <orders id="101" user_id="1" total="99.99" status="completed"/>
  <orders id="102" user_id="2" total="149.50" status="pending"/>
</dataset>
```

### Markdown Table Format

```markdown
| id | name | email | active |
|----|------|-------|--------|
| 1  | John Doe | john@example.com | true |
| 2  | Jane Smith | jane@example.com | false |
| 3  | Bob Wilson | bob@example.com | true |
```

## Complete Example

```markdown
---
function_name: "searchUsers"
description: "Search for active users with filtering and pagination"
version: "1.2.0"
---

# User Search Query

## Description

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

### Test Case 1: Basic Search

```yaml
parameters:
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
expected:
  count: 2
  status: "success"
```

### Test Case 2: CSV Format Test

```csv
user_id,active,include_email,expected
123,true,false,"user found"
456,false,true,"user inactive"
999,true,false,"user not found"
```

### Test Case 3: List Format Test

- Input: user_id = 789, include_email = true, active = true
- Expected: Returns user data with email included

## Mock Data

### YAML Format

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

### CSV Format

```csv
# users
id,name,email,department,active,created_at
1,"John Doe","john@example.com","engineering",true,"2024-01-15T10:30:00Z"
2,"Jane Smith","jane@example.com","design",true,"2024-02-20T14:45:00Z"
3,"Bob Wilson","bob@example.com","marketing",false,"2024-03-10T09:15:00Z"
```

### Markdown Table Format

| id | name | email | department | active | created_at |
|----|------|-------|------------|--------|------------|
| 1  | John Doe | john@example.com | engineering | true | 2024-01-15T10:30:00Z |
| 2  | Jane Smith | jane@example.com | design | true | 2024-02-20T14:45:00Z |
| 3  | Bob Wilson | bob@example.com | marketing | false | 2024-03-10T09:15:00Z |

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

## Implementation Details

### AST-Based Parsing

The parser leverages goldmark's AST directly to achieve:

1. **Robust Parsing**: Unaffected by Markdown syntax within code blocks
2. **Accurate Line Numbers**: Precise line number information for SQL code blocks
3. **Structured Data**: Direct data extraction from AST nodes

### Multi-Format Support

Each section supports multiple formats:

- **YAML**: Optimal for structured data
- **JSON**: Compatible with API specifications
- **CSV**: Optimal for tabular data
- **XML**: DBUnit compatible format
- **Markdown Table**: Visually clear
- **Text**: Free-form descriptions

### Error Handling

- Validation of required sections (Description/Overview, SQL)
- Detection of invalid front matter
- Detailed syntax error reporting

## File Organization

### Directory Structure

```
queries/
├── users/
│   ├── search.snap.md          # User search
│   ├── create.snap.md          # User creation
│   └── update.snap.md          # User update
├── posts/
│   ├── list.snap.md            # Post listing
│   ├── detail.snap.md          # Post details
│   └── search.snap.md          # Post search
└── analytics/
    ├── user-stats.snap.md      # User statistics
    └── daily-report.snap.md    # Daily report
```

### Naming Conventions

- Use kebab-case for file names
- Use `.snap.md` extension
- Group related queries in subdirectories
- Use descriptive names that reflect query purpose

## Processing Rules

### Section Recognition

1. Parser looks for English keywords in headings
2. Case-insensitive matching
3. Support for multiple alternative names (e.g., `Parameters`, `Params`, `Parameter`)

### Content Processing

- **Parameters**: Parsed as YAML/JSON/text format and stored in `ParameterBlock` field
- **SQL**: Processed as SnapSQL template with line number information
- **Test Cases**: Support multiple formats (YAML/JSON/CSV/XML/List)
- **Mock Data**: Support multiple formats (YAML/JSON/CSV/XML/Markdown Table)

### Validation Rules

1. Required sections (Description/Overview, SQL) must exist
2. Front matter must be valid YAML
3. SQL section must follow valid SnapSQL syntax
4. Each format's data must follow appropriate syntax

## SnapSQL CLI Integration

### File Discovery

```bash
# Process all .snap.md files
snapsql generate -i ./queries

# Process specific file
snapsql generate queries/users/search.snap.md
```

### Output Generation

Each `.snap.md` file generates:
- Intermediate JSON with parsed content
- Language-specific code (when requested)
- Test files with test cases and mock data

### Validation

```bash
# Validate markdown query file
snapsql validate queries/users/search.snap.md

# Validate all markdown files
snapsql validate -i ./queries --format json
```

## Benefits

1. **Literate Programming**: Integration of code and documentation
2. **Multi-Format Support**: Support for YAML, JSON, CSV, XML, Markdown tables, and more
3. **AST-Based**: Robust parsing using goldmark's AST directly
4. **Testability**: Comprehensive support for test cases and mock data
5. **Maintainability**: Version history and change tracking
6. **IDE Support**: Markdown syntax highlighting and preview
7. **Collaboration**: Easy review and commenting
8. **Type Safety**: Structured parameter definitions
9. **Internationalization**: Support for documentation in multiple languages
