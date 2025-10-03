# List Archive Toggle

## Description

Sets the archive flag on a list, keeping other attributes intact.

## Parameters

```yaml
list_id: int
is_archived: bool
```

## SQL

```sql
UPDATE lists
SET
    is_archived = /*= is_archived */false,
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

### Archive list marks flag and updates timestamp

**Fixtures:**
```yaml
boards:
  - id: 20
    name: "Marketing"
    status: "active"
    archived_at: null
    created_at: "2025-09-08T08:00:00Z"
    updated_at: "2025-09-08T08:00:00Z"
lists:
  - id: 600
    board_id: 20
    name: "Ideas"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-08T09:00:00Z"
    updated_at: "2025-09-08T09:00:00Z"
```

**Parameters:**
```yaml
list_id: 600
is_archived: true
```

**Expected Results:**
```yaml
- id: 600
  board_id: 20
  name: "Ideas"
  stage_order: 1
  position: 1
  is_archived: 1
  created_at: "2025-09-08T09:00:00Z"
  updated_at: [currentdate]
```
