# List Create

## Description

Creates lists for the specified board by duplicating all active entries from `list_templates`.

## Parameters

```yaml
board_id: int
```

## SQL

```sql
INSERT INTO lists (
  board_id,
  name,
  stage_order,
  position
)
SELECT
  /*= board_id */1 AS board_id,
  lt.name,
  lt.stage_order AS stage_order,
  CAST(lt.stage_order AS REAL) AS position
FROM list_templates AS lt
WHERE lt.is_active = 1
ORDER BY lt.stage_order
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

### Insert list returns created row

**Fixtures:**
```yaml
boards:
  - id: 7
    name: "Sprint"
    status: "active"
    archived_at: [null]
    created_at: [currentdate, -1d]
    updated_at: [currentdate, -1d]
list_templates:
  - id: 1
    name: "Backlog"
    stage_order: 1
    is_active: 1
  - id: 2
    name: "In Progress"
    stage_order: 2
    is_active: 1
```

**Parameters:**
```yaml
board_id: 7
```

**Expected Results: lists[pk-match]**
```yaml
- id: 1
  board_id: 7
  name: "Backlog"
  stage_order: 1
  position: 1.0
  is_archived: 0
  created_at: [currentdate]
  updated_at: [currentdate]
- id: 2
  board_id: 7
  name: "In Progress"
  stage_order: 2
  position: 2.0
  is_archived: 0
  created_at: [currentdate]
  updated_at: [currentdate]
```
