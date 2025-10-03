# Board Create

## Description

Creates a new board using the provided name and returns the persisted row including timestamps.

## Parameters

```yaml
name: string
```

## SQL

```sql
INSERT INTO boards (name, status)
VALUES (/*= name */'New Board', 'active')
RETURNING
    id,
    name,
    status,
    archived_at,
    created_at,
    updated_at;
```

## Test Cases

### Create board returns persisted row

**Parameters:**
```yaml
name: "Sprint Planning"
```

**Expected Results:**
```yaml
- id: 1
  name: "Sprint Planning"
  status: "active"
  archived_at: [null]
  created_at: [currentdate]
  updated_at: [currentdate]
```
