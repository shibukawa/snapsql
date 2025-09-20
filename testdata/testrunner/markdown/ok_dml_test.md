---
name: "dml_insert_test"
dialect: "sqlite"
---

# DML Insert Test

## Description

Test INSERT statement with DML validation features.

## Parameters
```yaml
name: "Test User"
email: "test@example.com"
```

## SQL
```sql
INSERT INTO users (name, email) VALUES (/*= name */'Test User', /*= email */'test@example.com');
```

## Test Cases

### Test: Insert with Numeric Validation

**Parameters:**
```yaml
name: "John Doe"
email: "john@example.com"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User"
  email: "existing@example.com"
```

**Expected Results:**
```yaml
- rows_affected: 1
- users[count]: 2
- users[exists]:
  - name: "John Doe"
    email: "john@example.com"
    exists: true
```

### Test: Insert with Table State Validation

**Parameters:**
```yaml
name: "Jane Smith"
email: "jane@example.com"
```

**Fixtures: users[clear-insert]**
```yaml
[]
```

**Expected Results:**
```yaml
- rows_affected: 1
- users:
  - id: 1
    name: "Jane Smith"
    email: "jane@example.com"
```
