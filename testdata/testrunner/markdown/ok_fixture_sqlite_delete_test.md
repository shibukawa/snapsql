---
name: "fixture_sqlite_delete"
dialect: "sqlite"
---

# SQLite Fixture Delete Strategy Test

## Description
Validate delete strategy removes rows matching primary key from prepared dataset.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: Delete one row by PK

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice-updated"
  email: "alice2@example.com"
- id: 2
  name: "Bob"
  email: "bob@example.com"
- id: 3
  name: "Charlie"
  email: "charlie@example.com"
```

**Fixtures: users[delete]**
```yaml
- id: 2
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice-updated"
  email: "alice2@example.com"
- id: 3
  name: "Charlie"
  email: "charlie@example.com"
```
