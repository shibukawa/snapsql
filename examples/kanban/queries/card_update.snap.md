# Card Update

## Description

Updates the title and description of a card and refreshes its timestamp.

## Parameters

```yaml
card_id: int
title: string
description: string
```

## SQL

```sql
UPDATE cards
SET
    title = /*= title */'Updated Card',
    description = /*= description */'',
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

### Update card title and description

**Fixtures:**
```yaml
boards:
  - id: 12
    name: "UX"
    status: "active"
    archived_at: null
    created_at: "2025-09-10T08:00:00Z"
    updated_at: "2025-09-10T08:00:00Z"
lists:
  - id: 400
    board_id: 12
    name: "Design"
    stage_order: 2
    position: 1
    is_archived: 0
    created_at: "2025-09-10T09:00:00Z"
    updated_at: "2025-09-10T09:00:00Z"
cards:
  - id: 950
    list_id: 400
    title: "Initial wireframe"
    description: "Needs feedback"
    position: 1
    created_at: "2025-09-10T09:30:00Z"
    updated_at: "2025-09-10T09:30:00Z"
```

**Parameters:**
```yaml
card_id: 950
title: "Finalized wireframe"
description: "Stakeholder approved"
```

**Expected Results:**
```yaml
- id: 950
  list_id: 400
  title: "Finalized wireframe"
  description: "Stakeholder approved"
  position: 1
  created_at: "2025-09-10T09:30:00Z"
  updated_at: [currentdate]
```
