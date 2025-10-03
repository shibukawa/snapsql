# Card Comment Create

## Description

Adds a comment to a card and returns the stored row, enabling optimistic UI updates.

## Parameters

```yaml
card_id: int
body: string
```

## SQL

```sql
INSERT INTO card_comments (
    card_id,
    body
)
VALUES (
    /*= card_id */1,
    /*= body */''
)
RETURNING
    id,
    card_id,
    body,
    created_at;
```

## Test Cases

### Insert comment returns stored row

**Fixtures:**
```yaml
boards:
  - id: 5
    name: "Release Plan"
    status: "active"
    archived_at: null
    created_at: "2025-09-18T09:00:00Z"
    updated_at: "2025-09-18T09:00:00Z"
lists:
  - id: 50
    board_id: 5
    name: "Todo"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-18T09:10:00Z"
    updated_at: "2025-09-18T09:10:00Z"
cards:
  - id: 500
    list_id: 50
    title: "Set up project"
    description: ""
    position: 1
    created_at: "2025-09-18T09:30:00Z"
    updated_at: "2025-09-18T09:30:00Z"
```

**Parameters:**
```yaml
card_id: 500
body: "Looks good!"
```

**Expected Results:**
```yaml
- id: 1
  card_id: 500
  body: "Looks good!"
  created_at: [currentdate]
```
