---
name: "test_update_no_returning"
dialect: "sqlite"
---

# UPDATE without RETURNING Test

## Description

Test UPDATE statement without RETURNING clause using validation strategies.

## Parameters
```yaml
min_age: 18
new_status: "active"
```

## SQL
```sql
UPDATE users SET status = 'active' WHERE age >= 18;
```

## Test Cases

### Test: UPDATE with Validation Strategies

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Minor User"
  email: "minor@example.com"
  age: 16
  status: "inactive"
  department_id: 1
- id: 2
  name: "Adult User 1"
  email: "adult1@example.com"
  age: 25
  status: "inactive"
  department_id: 1
- id: 3
  name: "Adult User 2"
  email: "adult2@example.com"
  age: 30
  status: "inactive"
  department_id: 2
```

**Expected Results:**
```yaml
- rows_affected: 2
- users[count]: 3
- users[exists]:
  - age: 25
    status: "active"
    exists: true
  - age: 30
    status: "active"
    exists: true
  - age: 16
    status: "inactive"
    exists: true
```
