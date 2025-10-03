# Board Archive

## Description

Archives the currently active board by switching its status to `archived` and stamping the archive timestamp. Because only one board can be active at a time, no parameters are required.

## SQL

```sql
UPDATE boards
SET
    status = 'archived',
    archived_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE status = 'active'
RETURNING
    id,
    name,
    status,
    archived_at,
    created_at,
    updated_at;
```

## Test Cases

### Active board is archived

**Fixtures:**
```yaml
boards:
  - id: 1
    name: "Sprint Board"
    status: "active"
    archived_at: [null]
    created_at: "2025-09-20T10:00:00Z"
    updated_at: "2025-09-20T10:00:00Z"
```

**Expected Results: boards[pk-match]**
```yaml
- id: 1
  name: "Sprint Board"
  status: "archived"
  archived_at: [currentdate]
  created_at: "2025-09-20T10:00:00Z"
  updated_at: [currentdate]
```

### No active boards returns empty result

**Fixtures:**
```yaml
boards:
  - id: 1
    name: "Archived Board"
    status: "archived"
    archived_at: "2025-09-10T12:00:00Z"
    created_at: "2025-09-01T09:00:00Z"
    updated_at: "2025-09-10T12:00:00Z"
```

**Expected Results: boards[pk-match]**
```yaml
- id: 1
  name: "Archived Board"
  status: "archived"
  archived_at: "2025-09-10T12:00:00Z"
  created_at: "2025-09-01T09:00:00Z"
  updated_at: "2025-09-10T12:00:00Z"
```