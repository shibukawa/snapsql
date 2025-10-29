# Card Create

## Description

Creates a card within a list and returns the inserted row for client-side refresh.

## Parameters

```yaml
title: string
description: string
position: float
```

## SQL

```sql
INSERT INTO cards (
    list_id,
    title,
    description,
    position
)
VALUES (
    (SELECT l.id 
     FROM lists AS l
     JOIN boards b ON l.board_id = b.id
     WHERE b.status = 'active'
     ORDER BY l.stage_order ASC
     LIMIT 1),
    /*= title */'Dummy Card Title',
    /*= description */'Dummy Description',
    /*= position */99.0
)
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

### Insert card into list

**Fixtures:**
```yaml
boards:
  - id: 20
    name: "Team Launch"
    status: "active"
    archived_at: null
    created_at: "2025-09-14T09:45:00Z"
    updated_at: "2025-09-14T09:45:00Z"
lists:
  - id: 300
    board_id: 20
    name: "Todo"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-14T10:00:00Z"
    updated_at: "2025-09-14T10:00:00Z"
```

**Parameters:**
```yaml
list_id: 300
title: "Write tests"
description: "Add coverage for kanban queries"
position: 10
```

**Expected Results:**
```yaml
- id: 1
  list_id: 300
  title: "Write tests"
  description: "Add coverage for kanban queries"
  position: 10
  created_at: [currentdate]
  updated_at: [currentdate]
```
