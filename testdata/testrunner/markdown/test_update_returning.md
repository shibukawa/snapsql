---
name: "test_update_returning"
dialect: "sqlite"
---

# UPDATE with RETURNING Test

## Description

Test UPDATE statement with RETURNING clause for direct result validation.

## Parameters
```yaml
target_email: "adult1@example.com"
new_name: "Updated Name"
```

## SQL
```sql
UPDATE users SET name = 'Updated Name' WHERE email = 'adult1@example.com'
RETURNING id, name, email, age, status, department_id;
```

## Test Cases

### Test: UPDATE with RETURNING Direct Validation

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Original Name"
  email: "adult1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Other User"
  email: "other@example.com"
  age: 30
  status: "active"
  department_id: 2
```

**Expected Results:**
```yaml
- id: 1
  name: "Updated Name"
  email: "adult1@example.com"
  age: 25
  status: "active"
  department_id: 1
```
