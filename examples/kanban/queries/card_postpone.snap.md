# Card Postpone

## Description

Moves all unfinished cards from a source board to the corresponding lists of a destination board created from templates. The destination list is chosen by matching `stage_order`; lists in the terminal stage (the maximum `stage_order` on the source board) retain their cards.

## Parameters

```yaml
src_board_id: int
dst_board_id: int
```

## SQL

```sql
WITH
    done_stage AS (
        SELECT stage_order AS stage_limit
        FROM lists
        WHERE board_id = /*= src_board_id */99
        ORDER BY stage_order DESC, id DESC
        LIMIT 1
    ),
    new_list AS (
        SELECT id AS new_list_id
        FROM lists
        WHERE board_id = /*= dst_board_id */99
        ORDER BY stage_order ASC, id ASC
        LIMIT 1
    ),
    undone_lists AS (
        SELECT old.id AS old_list_id
        FROM lists AS old
        WHERE old.board_id = /*= src_board_id */99
            AND old.stage_order < (
            SELECT stage_limit FROM done_stage
            )
    )
UPDATE cards
SET
    list_id = (SELECT new_list_id FROM new_list),
    updated_at = CURRENT_TIMESTAMP
WHERE list_id IN (
    SELECT old_list_id
    FROM undone_lists
);
```

## Test Cases

### Move undone cards to new board

**Fixtures:**

```yaml
boards:
  - id: 10
    name: "Sprint Alpha"
    status: "archived"
    created_at: "2025-09-15T09:00:00Z"
    updated_at: "2025-09-29T09:00:00Z"
  - id: 20
    name: "Sprint Beta"
    status: "active"
    created_at: "2025-09-30T09:00:00Z"
    updated_at: "2025-09-30T09:00:00Z"
lists:
  - id: 101
    board_id: 10
    name: "Backlog"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-15T09:05:00Z"
    updated_at: "2025-09-15T09:05:00Z"
  - id: 199
    board_id: 10
    name: "Done"
    stage_order: 2
    position: 2
    is_archived: 0
    created_at: "2025-09-15T09:05:00Z"
    updated_at: "2025-09-15T09:05:00Z"
  - id: 201
    board_id: 20
    name: "Backlog"
    stage_order: 1
    position: 1
    is_archived: 0
    created_at: "2025-09-30T09:00:00Z"
    updated_at: "2025-09-30T09:00:00Z"
cards:
  - id: 1001
    list_id: 101
    title: "Spec requirements"
    description: "Collect feedback"
    position: 1
    created_at: "2025-09-15T09:10:00Z"
    updated_at: "2025-09-29T09:10:00Z"
  - id: 1002
    list_id: 199
    title: "Release"
    description: "GA rollout"
    position: 1
    created_at: "2025-09-15T09:20:00Z"
    updated_at: "2025-09-29T09:20:00Z"
```

**Parameters:**
```yaml
src_board_id: 10
dst_board_id: 20
```

**Expected Results: cards**
```yaml
- id: 1001
  list_id: 201
  title: "Spec requirements"
  description: "Collect feedback"
  position: 1
  created_at: "2025-09-15T09:10:00Z"
  updated_at: [currentdate]
- id: 1002
  list_id: 199
  title: "Release"
  description: "GA rollout"
  position: 1
  created_at: "2025-09-15T09:20:00Z"
  updated_at: "2025-09-29T09:20:00Z"
```
