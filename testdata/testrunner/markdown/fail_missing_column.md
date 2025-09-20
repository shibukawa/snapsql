---
name: "missing_column_test"
dialect: "sqlite"
---

# Missing Column Test

## Description

Expected result references a column that the query does not return.

## SQL
```sql
SELECT id, name FROM users ORDER BY id;
```

## Test Cases

### Test: Expect Extra Column

**Parameters:**
```yaml
note: "missing column"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"  # email column not selected -> mismatch expected
```
