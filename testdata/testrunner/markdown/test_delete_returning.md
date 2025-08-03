---
name: "test_delete_returning"
dialect: "sqlite"
---

# DELETE with RETURNING Test

## Description

Test DELETE statement with RETURNING clause for direct result validation.

## Parameters
```yaml
target_status: "inactive"
```

## SQL
```sql
DELETE FROM users WHERE status = 'inactive'
RETURNING id, name, email, age, status, department_id;
```

## Test Cases

### Test: DELETE with RETURNING Direct Validation

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Active User 1"
  email: "active1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Inactive User"
  email: "inactive@example.com"
  age: 28
  status: "inactive"
  department_id: 2
- id: 3
  name: "Active User 2"
  email: "active2@example.com"
  age: 30
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- id: 2
  name: "Inactive User"
  email: "inactive@example.com"
  age: 28
  status: "inactive"
  department_id: 2
```
