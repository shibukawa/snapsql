# Board Get

## Description

Retrieves one board record for detail views. Returns an empty result if the board does not exist.

## Parameters

```yaml
board_id: int
```

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
WHERE id = /*= board_id */1;
```

## Test Cases

### Fetch single board

**Fixtures:**
```yaml
boards:
  - id: 10
    name: "Backlog"
    status: "archived"
    archived_at: "2025-09-05T08:00:00Z"
    created_at: "2025-08-01T08:00:00Z"
    updated_at: "2025-08-03T12:00:00Z"
```

**Parameters:**
```yaml
board_id: 10
```

**Expected Results:**
```yaml
- id: 10
  name: "Backlog"
  status: "archived"
  archived_at: "2025-09-05T08:00:00Z"
  created_at: "2025-08-01T08:00:00Z"
  updated_at: "2025-08-03T12:00:00Z"
```
