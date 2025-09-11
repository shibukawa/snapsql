# Markdown Query Definition Format

## Overview

SnapSQL supports literate programming through Markdown-based query definition files (`.snap.md`). This format integrates SQL templates with comprehensive documentation, test cases, and metadata into a single, readable document.

## Section Structure (Implementation-aligned)

| Section/Item | Required | Supported Formats | Alternative | Count |
|-------------|----------|-------------------|-------------|-------|
| Front Matter | × | YAML | Function Name + Description Section | 0-1 |
| Function Name | Auto | Auto (explicit allowed) | `function_name` (explicit) / file name fallback | 1 |
| Description | Required | Text/Markdown (H2) | Overview | 1 |
| Parameters | Optional | Fenced code: YAML/JSON (lists allowed for docs) | - | 0-1 |
| SQL | Required | Fenced code: language `sql` | - | 1 |
| Test Cases | Optional | H3+ heading + emphasis marker sub-sections | - | 0-n |
| - Fixtures | Optional | YAML/JSON (multi-table), CSV (single-table), DBUnit XML | - | 0-many per test |
| - Parameters | Required | YAML/JSON (fenced) | - | 1 per test |
| - Expected Results | Required | YAML/JSON (array) | - | 1 per test |
| - Verify Query | Optional | Fenced code: language `sql` | - | 0-1 per test |

## Section Details

### Front Matter (Optional)

YAML format metadata. Can be replaced by Function Name and Description sections.

```yaml
---
function_name: "get_user_data"  # Explicit function name (optional)
description: "Get user data"    # Description (optional)
dialect: postgres                # Dialect hint (optional)
---
```

### Function Name (Automatic)

Resolved by priority (auto-generated if omitted):

1. `function_name` in front matter
2. File name without extension (e.g., `get_users.snap.md` → `get_users`)

Note: H1 is used as the document title and not for name generation.

#### Normalization / conversion rules
- Extension handling: if the file ends with `.snap.md`, strip that pair; otherwise strip the last extension only (e.g., `.md`).
- Character conversion: do not change letter case, spaces, or symbols; the file name is used as-is.
- Recommendation: using `snake_case` is encouraged but not enforced.
- Collisions: behavior is undefined; avoid duplicate function names in the same package.

### Description (Required)

Purpose and explanation of the query. Can also use `Overview` as heading.

```markdown
## Description

This query retrieves user data based on the specified user ID.
Email retrieval can be controlled via an option.
```

### Parameters (Optional)

Define input parameters. Parsing switches by fenced language. The parser also stores raw text (`ParametersText`/`ParametersType`).

Fully optional: if omitted, the query has no parameters (empty fenced blocks are discouraged).

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

Lists may be used for human-readable docs only; they are not treated as typed definitions.

### SQL (Required)

SQL template in SnapSQL format. Must be a fenced block with language `sql` (the parser records the starting line number).

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

Use H3 (`###`) or deeper as the test case name. Then place an emphasized label in the following paragraph to switch sub-sections.

Labels (case-insensitive, colon accepted):
- Parameters: `Parameters:` / `Params:` / `Input Parameters:`
- Expected Results: `Expected:` / `Expected Results:` / `Expected Result:` / `Results:`
- Verify Query: `Verify Query:` / `Verification Query:` (optional)
- Fixtures: `Fixtures:` (for CSV, prefer `Fixtures: <table>[strategy]`)

#### Strict rules
- Each test case must contain exactly one `Parameters` and one `Expected` section (duplicates are errors).
- `Fixtures` is optional and may appear multiple times (YAML/JSON/CSV/XML can be mixed).
- `Verify Query` is optional and may appear at most once.
- Labels must be written as emphasis (italic/bold) in a paragraph (not as headings).
- The label must precede the fenced code block of that sub-section.
- Typical errors include:
  - Duplicate `Parameters` or `Expected` sections
  - Missing required sections (`Description`, `SQL`)
  - Fenced language mismatch (e.g., SQL code block not tagged as `sql`)

#### Fixtures Examples

**YAML/JSON (multi-table):**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**CSV (single-table with strategy):**

Emphasis label example:

```
**Fixtures: users[insert]**
```

Then the fenced code:

```csv
id,name,email,department_id
1,John Doe,john@example.com,1
2,Jane Smith,jane@example.com,2
```

Strategies: `clear-insert` (default) / `insert` / `upsert` / `delete`.

**DBUnit XML format:**
```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" department_id="1"/>
  <users id="2" name="Jane Smith" email="jane@example.com" department_id="2"/>
  <departments id="1" name="Engineering"/>
  <departments id="2" name="Design"/>
</dataset>
```

## Fixture Strategies and Formats

Supported insert strategies (`[strategy]`):
- `clear-insert` (default): truncate then insert
- `insert`: insert only (keep existing rows)
- `upsert`: insert or update on conflict
- `delete`: delete rows matching the provided keys

Format-specific handling:
- YAML/JSON (multi-table):
  - Use a map keyed by table name; each value is an array of rows.
  - If the label is written as `Fixtures: <table>[strategy]`, apply the strategy to that table.
  - If no strategy is specified, `clear-insert` is applied.
- CSV (single-table):
  - The label must specify `Fixtures: <table>[strategy]`.
  - The fenced CSV contains a header row followed by data rows.
- DBUnit XML:
  - Children under `<dataset>` are treated as table rows with attributes as columns.

Notes:
- When multiple `Fixtures` sections are combined, rows for the same table are concatenated in order of appearance.
- Strategy is interpreted per table; when strategies differ, the one specified with the section label takes precedence for that section.

#### Parameters Examples (inside test cases)

**YAML/JSON (recommended):**
```yaml
user_id: 1
include_email: true
```

Lists are not supported here.

#### Expected Results Examples

Supports YAML/JSON arrays.

**YAML:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

CSV/DBUnit XML are not supported for expected results.

## Complete Example

````markdown
---
function_name: "get_user_data"
description: "Get user data"
dialect: postgres
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
