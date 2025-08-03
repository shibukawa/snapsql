---
name: "test_select"
dialect: "sqlite"
---

# SELECT Query Test

## Description

Test SELECT query with JOIN and WHERE conditions.

## Parameters
```yaml
min_age: 18
```

## SQL
```sql
SELECT u.id, u.name, u.email, u.age, u.status, u.department_id
FROM users u
WHERE u.age >= 18 AND u.status = 'active'
ORDER BY u.id;
```

## Test Cases

### Test: SELECT Active Adult Users

**Parameters:**
```yaml
min_age: 18
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
  age: 30
  status: "active"
  department_id: 2
- id: 3
  name: "Bob Wilson"
  email: "bob@example.com"
  age: 16
  status: "inactive"
  department_id: 1
```

**Expected Results:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
  age: 30
  status: "active"
  department_id: 2
```
