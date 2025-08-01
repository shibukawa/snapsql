# Markdown Query Definition Format

## Overview

SnapSQL supports literate programming through Markdown-based query definition files (`.snap.md`). This format integrates SQL templates with comprehensive documentation, test cases, and metadata into a single, readable document.

## Section Structure

| Section/Item | Required | Supported Formats | Alternative | Count |
|-------------|----------|-------------------|-------------|-------|
| Front Matter | × | YAML | Function Name + Description Section | 0-1 |
| Function Name | ○ | H1 Title (auto snake_case) | Front Matter `function_name` | 1 |
| Description | ○ | Text, Markdown | Overview | 1 |
| Parameters | × | YAML, JSON, List | - | 0-1 |
| SQL | ○ | SQL (SnapSQL format) | - | 1 |
| Test Cases | × | YAML, JSON, List | - | 0-n |
| - Fixtures | × | YAML, JSON, CSV, DBUnit XML, List | - | 0-1 per test |
| - Parameters | ○ | YAML, JSON, List | - | 1 per test |
| - Expected Results | ○ | YAML, JSON, CSV, DBUnit XML, List | - | 1 per test |

## Section Details

### Front Matter (Optional)

YAML format metadata. Can be replaced by Function Name and Description sections.

```yaml
---
function_name: "get_user_data"  # Explicit function name (optional)
description: "Get user data"    # Description (optional)
version: "1.0.0"               # Additional metadata
---
```

### Function Name (Required)

Automatically generates snake_case function name from H1 title.

```markdown
# Get User Data Query
```
↓ Auto-converts to
```
function_name: "get_user_data_query"
```

### Description (Required)

Purpose and explanation of the query. Can also use `Overview` as heading.

```markdown
## Description

This query retrieves user data based on the specified user ID.
Email retrieval can be controlled via an option.
```

### Parameters (Optional)

Input parameter definitions. Supports multiple formats.

**YAML format (recommended):**
```yaml
user_id: int
include_email: bool
filters:
  active: bool
  departments: [string]
pagination:
  limit: int
  offset: int
```

**JSON format:**
```json
{
  "user_id": "int",
  "include_email": "bool",
  "filters": {
    "active": "bool",
    "departments": ["string"]
  }
}
```

**List format:**
```markdown
- user_id (int): User ID
- include_email (bool): Whether to include email
- filters.active (bool): Active users only
- filters.departments ([string]): Department filter
```

### SQL (Required)

SQL template in SnapSQL format.

```sql
SELECT 
    u.id,
    u.name,
    /*# if include_email */
    u.email,
    /*# end */
    d.id as departments__id,
    d.name as departments__name
FROM users u
    JOIN departments d ON u.department_id = d.id
WHERE u.id = /*= user_id */1
```

### Test Cases (Optional)

Test case definitions. Each test case can include:
- Fixtures: Database state before test execution
- Parameters: Test input values
- Expected Results: Expected output

#### Fixtures Format Examples

**YAML format:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**CSV format:**
```csv
# users
id,name,email,department_id
1,"John Doe","john@example.com",1
2,"Jane Smith","jane@example.com",2

# departments
id,name
1,"Engineering"
2,"Design"
```

**DBUnit XML format:**
```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" department_id="1"/>
  <users id="2" name="Jane Smith" email="jane@example.com" department_id="2"/>
  <departments id="1" name="Engineering"/>
  <departments id="2" name="Design"/>
</dataset>
```

#### Parameters Format Examples

**YAML format (recommended):**
```yaml
user_id: 1
include_email: true
```

**JSON format:**
```json
{
  "user_id": 1,
  "include_email": true
}
```

**List format:**
```markdown
- user_id: 1
- include_email: true
```

#### Expected Results Format Examples

Supports the same formats as Fixtures.

**YAML format:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

**CSV format:**
```csv
id,name,email,departments__id,departments__name
1,"John Doe","john@example.com",1,"Engineering"
```

## Complete Example

````markdown
---
version: "1.0.0"
author: "Development Team"
---

# Get User Data Query

## Description

Retrieves user data based on user ID.
Email retrieval can be controlled via option.

## Parameters

```yaml
user_id: int
include_email: bool
```

## SQL

```sql
SELECT 
    u.id,
    u.name,
    /*# if include_email */
    u.email,
    /*# end */
    d.id as departments__id,
    d.name as departments__name
FROM users u
    JOIN departments d ON u.department_id = d.id
WHERE u.id = /*= user_id */1
```

## Test Cases

### Test: Basic user data

**Fixtures:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**Parameters:**
```yaml
user_id: 1
include_email: true
```

**Expected Results:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

### Test: Without email

**Parameters:**
```yaml
user_id: 2
include_email: false
```

**Expected Results:**
```yaml
- id: 2
  name: "Jane Smith"
  departments__id: 2
  departments__name: "Design"
```
````
