---
name: "test_insert_no_returning"
dialect: "sqlite"
---

# INSERT without RETURNING Test

## Description

Test INSERT statement without RETURNING clause using various validation strategies.

## Parameters
```yaml
new_name: "Alice Johnson"
new_email: "alice@example.com"
new_age: 28
new_dept_id: 1
```

## SQL
```sql
INSERT INTO users (name, email, age, status, department_id) 
VALUES ('Alice Johnson', 'alice@example.com', 28, 'inactive', 1);
```

## Test Cases

### Test: INSERT with Basic Validation

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
  department_id: 2
```

**Expected Results:**
```yaml
- rows_affected: 1
- users[count]: 2
- users[exists]:
  - name: "Alice Johnson"
    email: "alice@example.com"
    exists: true
```
