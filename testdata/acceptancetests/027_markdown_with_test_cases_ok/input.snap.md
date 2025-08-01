# Find User Query

## Description

This query finds a user by their ID.

## Parameters

- user_id: int

## SQL

```sql
-- @name: find_user
-- @affinity: one
SELECT
    id,
    name,
    age
FROM
    users
WHERE
    id = /*= user_id */1;
```

## Test Cases

### Test Case 1: Find existing user
**Input Parameters:**
```json
{
    "user_id": 123
}
```

**Expected Result:**
```yaml
# Mock data for find_user
- id: 123
  name: "John Doe"
  age: 30
```

### Test Case 2: Find young user
**Input Parameters:**
```json
{
    "user_id": 456
}
```

**Expected Result:**
```yaml
# Mock data for find_user
- id: 456
  name: "Jane Smith"
  age: 25
```
