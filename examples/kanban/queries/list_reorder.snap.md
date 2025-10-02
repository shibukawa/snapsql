# List Reorder

## Description

Adjusts the position value of a list to support drag and drop operations.

## Parameters

```yaml
list_id: int
position: float
```

## SQL

```sql
UPDATE lists
SET
    position = /*= position */0.0,
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

### Reorder list adjusts position only

**Fixtures:**
```yaml
boards:
  - id: 22
    name: "Sales"
    status: "active"
    archived_at: null
    created_at: "2025-09-06T08:00:00Z"
    updated_at: "2025-09-06T08:00:00Z"
lists:
  - id: 700
    board_id: 22
    name: "Pipeline"
    stage_order: 2
    position: 1
    is_archived: 0
    created_at: "2025-09-06T09:00:00Z"
    updated_at: "2025-09-06T09:00:00Z"
```

**Parameters:**
```yaml
list_id: 700
position: 3.75
```

**Expected Results:**
```yaml
- id: 700
  board_id: 22
  name: "Pipeline"
  stage_order: 2
  position: 3.75
  is_archived: 0
  created_at: "2025-09-06T09:00:00Z"
  updated_at: [currentdate]
```
