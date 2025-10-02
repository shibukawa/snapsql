# Board Tree

## Description

Returns a denormalised tree of a board, including all non-archived lists and their cards. Nested column aliases follow the SnapSQL hierarchy convention.

## Parameters

```yaml
board_id: int
```

## SQL

```sql
SELECT
    b.id,
    b.name,
    b.status,
    b.archived_at,
    b.created_at,
    b.updated_at,
    l.id AS lists__id,
    l.board_id AS lists__board_id,
    l.name AS lists__name,
    l.stage_order AS lists__stage_order,
    l.position AS lists__position,
    l.is_archived AS lists__is_archived,
    l.created_at AS lists__created_at,
    l.updated_at AS lists__updated_at,
    c.id AS lists__cards__id,
    c.list_id AS lists__cards__list_id,
    c.title AS lists__cards__title,
    c.description AS lists__cards__description,
    c.position AS lists__cards__position,
    c.created_at AS lists__cards__created_at,
    c.updated_at AS lists__cards__updated_at
FROM boards b
LEFT JOIN lists l ON l.board_id = b.id AND l.is_archived = 0
LEFT JOIN cards c ON c.list_id = l.id
WHERE b.id = /*= board_id */1
ORDER BY l.stage_order ASC, l.position ASC, c.position ASC;
```

## Test Cases

### Board tree includes active lists and their cards

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
  - id: 51
    board_id: 5
    name: "Done"
    stage_order: 4
    position: 2
    is_archived: 0
    created_at: "2025-09-18T09:20:00Z"
    updated_at: "2025-09-18T09:20:00Z"
cards:
  - id: 500
    list_id: 50
    title: "Set up project"
    description: "Install dependencies"
    position: 1
    created_at: "2025-09-18T09:30:00Z"
    updated_at: "2025-09-18T09:30:00Z"
  - id: 501
    list_id: 50
    title: "Design wireframes"
    description: "Sketch main screens"
    position: 2
    created_at: "2025-09-18T10:00:00Z"
    updated_at: "2025-09-18T10:00:00Z"
```

**Parameters:**
```yaml
board_id: 5
```

**Expected Results:**
```yaml
- id: 5
  name: "Release Plan"
  status: "active"
  archived_at: null
  created_at: "2025-09-18T09:00:00Z"
  updated_at: "2025-09-18T09:00:00Z"
  lists__id: 50
  lists__board_id: 5
  lists__name: "Todo"
  lists__stage_order: 1
  lists__position: 1
  lists__is_archived: 0
  lists__created_at: "2025-09-18T09:10:00Z"
  lists__updated_at: "2025-09-18T09:10:00Z"
  lists__cards__id: 500
  lists__cards__list_id: 50
  lists__cards__title: "Set up project"
  lists__cards__description: "Install dependencies"
  lists__cards__position: 1
  lists__cards__created_at: "2025-09-18T09:30:00Z"
  lists__cards__updated_at: "2025-09-18T09:30:00Z"
- id: 5
  name: "Release Plan"
  status: "active"
  archived_at: null
  created_at: "2025-09-18T09:00:00Z"
  updated_at: "2025-09-18T09:00:00Z"
  lists__id: 50
  lists__board_id: 5
  lists__name: "Todo"
  lists__stage_order: 1
  lists__position: 1
  lists__is_archived: 0
  lists__created_at: "2025-09-18T09:10:00Z"
  lists__updated_at: "2025-09-18T09:10:00Z"
  lists__cards__id: 501
  lists__cards__list_id: 50
  lists__cards__title: "Design wireframes"
  lists__cards__description: "Sketch main screens"
  lists__cards__position: 2
  lists__cards__created_at: "2025-09-18T10:00:00Z"
  lists__cards__updated_at: "2025-09-18T10:00:00Z"
- id: 5
  name: "Release Plan"
  status: "active"
  archived_at: null
  created_at: "2025-09-18T09:00:00Z"
  updated_at: "2025-09-18T09:00:00Z"
  lists__id: 51
  lists__board_id: 5
  lists__name: "Done"
  lists__stage_order: 4
  lists__position: 2
  lists__is_archived: 0
  lists__created_at: "2025-09-18T09:20:00Z"
  lists__updated_at: "2025-09-18T09:20:00Z"
  lists__cards__id: null
  lists__cards__list_id: null
  lists__cards__title: null
  lists__cards__description: null
  lists__cards__position: null
  lists__cards__created_at: null
  lists__cards__updated_at: null
```
