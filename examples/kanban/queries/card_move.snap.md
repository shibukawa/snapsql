# Card Move

## Description

Moves a card to a target list and position, using a transactional update for drag-and-drop operations.

## Parameters

```yaml
card_id: int
target_list_id: int
target_position: float
```

## SQL

```sql
UPDATE cards
SET
    list_id = /*= target_list_id */1,
    position = /*= target_position */0.0,
    updated_at = CURRENT_TIMESTAMP
WHERE id = /*= card_id */1
RETURNING
    id,
    list_id,
    title,
    description,
    position,
    created_at,
    updated_at;
```

## Test Cases

### Move card to different list

**Fixtures:**
```yaml
boards:
  - id: 10
    name: "Engineering"
    status: "active"
    archived_at: null
    created_at: "2025-09-15T08:00:00Z"
    updated_at: "2025-09-15T08:00:00Z"
lists:
  - id: 200
    board_id: 10
    name: "Todo"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-15T09:00:00Z"
    updated_at: "2025-09-15T09:00:00Z"
  - id: 201
    board_id: 10
    name: "In Review"
    stage_order: 3
    position: 2
    is_archived: 0
    created_at: "2025-09-15T10:00:00Z"
    updated_at: "2025-09-15T10:00:00Z"
cards:
  - id: 800
    list_id: 200
    title: "Ship new feature"
    description: ""
    position: 1
    created_at: "2025-09-15T11:00:00Z"
    updated_at: "2025-09-15T11:00:00Z"
```

**Parameters:**
```yaml
card_id: 800
target_list_id: 201
target_position: 5
```

**Expected Results:**
```yaml
- id: 800
  list_id: 201
  title: "Ship new feature"
  description: ""
  position: 5
  created_at: "2025-09-15T11:00:00Z"
  updated_at: [currentdate]
```
