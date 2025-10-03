# Card Comment List

## Description

Retrieves comments associated with a card, ordered by creation time ascending for chronological display.

## Parameters

```yaml
card_id: int
```

## SQL

```sql
SELECT
    id,
    card_id,
    body,
    created_at
FROM card_comments
WHERE card_id = /*= card_id */1
ORDER BY created_at ASC, id ASC;
```

## Test Cases

### Fetch comments chronologically

**Fixtures:**
```yaml
boards:
  - id: 20
    name: "Team Launch"
    status: "active"
    archived_at: null
    created_at: "2025-09-14T09:40:00Z"
    updated_at: "2025-09-14T09:40:00Z"
lists:
  - id: 300
    board_id: 20
    name: "Todo"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-14T09:55:00Z"
    updated_at: "2025-09-14T09:55:00Z"
cards:
  - id: 401
    list_id: 300
    title: "Review"
    description: ""
    position: 3
    created_at: "2025-09-14T10:10:00Z"
    updated_at: "2025-09-14T10:10:00Z"
  - id: 999
    list_id: 300
    title: "Review"
    description: ""
    position: 4
    created_at: "2025-09-14T10:10:00Z"
    updated_at: "2025-09-14T10:10:00Z"
card_comments:
  - id: 600
    card_id: 401
    body: "Looks good"
    created_at: "2025-09-14T11:00:00Z"
    updated_at: "2025-09-14T11:10:00Z"
  - id: 601
    card_id: 401
    body: "Please update"
    created_at: "2025-09-14T11:05:00Z"
    updated_at: "2025-09-14T11:10:00Z"
  - id: 999
    card_id: 999
    body: "Other card"
    created_at: "2025-09-14T08:00:00Z"
    updated_at: "2025-09-14T11:10:00Z"
```

**Parameters:**
```yaml
card_id: 401
```

**Expected Results:**
```yaml
- id: 600
  card_id: 401
  body: "Looks good"
  created_at: "2025-09-14T11:00:00Z"
- id: 601
  card_id: 401
  body: "Please update"
  created_at: "2025-09-14T11:05:00Z"
```
