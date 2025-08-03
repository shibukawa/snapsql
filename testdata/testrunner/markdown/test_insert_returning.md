---
name: "test_insert_returning"
dialect: "sqlite"
---

# INSERT with RETURNING Test

## Description

Test INSERT statement with RETURNING clause for direct result validation.

## Parameters
```yaml
new_name: "Charlie Brown"
new_email: "charlie@example.com"
new_age: 35
new_dept_id: 2
```

## SQL
```sql
INSERT INTO users (name, email, age, status, department_id) 
VALUES ('Charlie Brown', 'charlie@example.com', 35, 'inactive', 2)
RETURNING id, name, email, age, status, department_id;
```

## Test Cases

### Test: INSERT with RETURNING Direct Validation

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User"
  email: "existing@example.com"
  age: 30
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- id: 2
  name: "Charlie Brown"
  email: "charlie@example.com"
  age: 35
  status: "inactive"
  department_id: 2
```
