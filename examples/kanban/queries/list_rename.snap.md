# List Rename

## Description

Renames a list and updates its timestamp for optimistic concurrency.

## Parameters

```yaml
list_id: int
name: string
```

## SQL

```sql
UPDATE lists
SET
    name = /*= name */'Updated List',
    updated_at = CURRENT_TIMESTAMP
WHERE id = /*= list_id */1
RETURNING
    id,
    board_id,
    name,
    stage_order,
    position,
    is_archived,
    created_at,
    updated_at;
```

## Test Cases

### Rename list updates name only

**Fixtures:**
```yaml
boards:
  - id: 21
    name: "Growth"
    status: "active"
    archived_at: null
    created_at: "2025-09-07T08:00:00Z"
    updated_at: "2025-09-07T08:00:00Z"
lists:
  - id: 650
    board_id: 21
    name: "Backlog"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-07T09:00:00Z"
    updated_at: "2025-09-07T09:00:00Z"
```

**Parameters:**
```yaml
list_id: 650
name: "In Progress"
```

**Expected Results:**
```yaml
- id: 650
  board_id: 21
  name: "In Progress"
  stage_order: 1
  position: 1
  is_archived: 0
  created_at: "2025-09-07T09:00:00Z"
  updated_at: [currentdate]
```
