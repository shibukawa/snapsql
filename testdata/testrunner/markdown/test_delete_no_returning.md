---
name: "test_delete_no_returning"
dialect: "sqlite"
---

# DELETE without RETURNING Test

## Description

Test DELETE statement without RETURNING clause using validation strategies.

## Parameters
```yaml
max_age: 25
```

## SQL
```sql
DELETE FROM users WHERE age <= 25;
```

## Test Cases

### Test: DELETE with Validation Strategies

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Young User 1"
  email: "young1@example.com"
  age: 22
  status: "active"
  department_id: 1
- id: 2
  name: "Young User 2"
  email: "young2@example.com"
  age: 25
  status: "active"
  department_id: 2
- id: 3
  name: "Old User"
  email: "old@example.com"
  age: 35
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- rows_affected: 2
- users[count]: 1
- users[exists]:
  - age: 22
    exists: false
  - age: 25
    exists: false
  - age: 35
    exists: true
```
