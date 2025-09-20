---
name: "value_mismatch_test"
dialect: "sqlite"
---

# Value Mismatch Test

## Description

Expected value differs from actual for a column.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: Wrong Email Expectation

**Parameters:**
```yaml
note: "value mismatch"
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
  email: "wrong@example.com"
```
