---
function_name: find_user
response_affinity: one
---

# Find User Query

## Description

This query finds a user by their ID.

## Parameters

```yaml
user_id: int
```

## SQL

```sql
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

**Parameters:**
```yaml
user_id: 123
```

**Expected Results:**
```yaml
- id: 123
  name: "John Doe"
  age: 30
```

### Test Case 2: Find young user

**Parameters:**
```yaml
user_id: 456
```

**Expected Results:**
```yaml
- id: 456
  name: "Jane Smith"
  age: 25
```
