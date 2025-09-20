---
name: "row_count_mismatch_test"
dialect: "sqlite"
---

# Row Count Mismatch Test

## Description

Expected result has different number of rows than actual.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: Expect Too Many Rows

**Parameters:**
```yaml
note: "expect more"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
- id: 2
  name: "Bob"
  email: "bob@example.com"
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
- id: 2
  name: "Bob"
  email: "bob@example.com"
- id: 3
  name: "Carol"
  email: "carol@example.com"
```
