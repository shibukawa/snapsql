---
name: "simple_dml_test"
dialect: "sqlite"
---

# Simple DML Test

## Description

Test INSERT statement without parameters.

## Parameters
```yaml
```

## SQL
```sql
INSERT INTO users (name, email) VALUES ('Simple User', 'simple@example.com');
```

## Test Cases

### Test: Simple Insert

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
[]
```

**Expected Results:**
```yaml
- rows_affected: 1
- users[count]: 1
- users:
  - id: 1
    name: "Simple User"
    email: "simple@example.com"
```
