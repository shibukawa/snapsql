# Board List

## Description

Fetches every board with basic metadata, ordered by most recently created first. Used for the dashboard overview.

## SQL

```sql
SELECT
    id,
    name,
    status,
    archived_at,
    created_at,
    updated_at
FROM boards
ORDER BY created_at DESC;
```

## Test Cases

### Boards are returned in descending creation order

**Fixtures:**
```yaml
boards:
  - id: 1
    name: "Project Alpha"
    status: "archived"
    archived_at: "2025-09-21T10:00:00Z"
    created_at: "2025-09-19T10:00:00Z"
    updated_at: "2025-09-19T10:00:00Z"
  - id: 2
    name: "Project Beta"
    status: "active"
    archived_at: null
    created_at: "2025-09-20T09:30:00Z"
    updated_at: "2025-09-20T09:30:00Z"
```

**Parameters:**
```yaml
{}
```

**Expected Results:**
```yaml
- id: 2
  name: "Project Beta"
  status: "active"
  archived_at: null
  created_at: "2025-09-20T09:30:00Z"
  updated_at: "2025-09-20T09:30:00Z"
- id: 1
  name: "Project Alpha"
  status: "archived"
  archived_at: "2025-09-21T10:00:00Z"
  created_at: "2025-09-19T10:00:00Z"
  updated_at: "2025-09-19T10:00:00Z"
```
