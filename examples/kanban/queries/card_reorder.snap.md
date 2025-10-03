# Card Reorder

## Description

Adjusts the position field of a card without changing its list, supporting intra-list reordering.

## Parameters

```yaml
card_id: int
position: float
```

## SQL

```sql
UPDATE cards
SET
    position = /*= position */0.0,
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

### Reorder card within list

**Fixtures:**
```yaml
boards:
  - id: 11
    name: "QA"
    status: "active"
    archived_at: null
    created_at: "2025-09-12T08:00:00Z"
    updated_at: "2025-09-12T08:00:00Z"
lists:
  - id: 300
    board_id: 11
    name: "Testing"
    stage_order: 2
    position: 1
    is_archived: 0
    created_at: "2025-09-12T09:00:00Z"
    updated_at: "2025-09-12T09:00:00Z"
cards:
  - id: 900
    list_id: 300
    title: "Regression"
    description: ""
    position: 1
    created_at: "2025-09-12T09:30:00Z"
    updated_at: "2025-09-12T09:30:00Z"
```

**Parameters:**
```yaml
card_id: 900
position: 2.5
```

**Expected Results:**
```yaml
- id: 900
  list_id: 300
  title: "Regression"
  description: ""
  position: 2.5
  created_at: "2025-09-12T09:30:00Z"
  updated_at: [currentdate]
```
