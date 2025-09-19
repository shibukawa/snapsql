---
name: "fixture_sqlite_clear_insert_twice"
dialect: "sqlite"
---

# SQLite Fixture Clear-Insert Twice Strategy Test

## Description
Validate that applying clear-insert twice results only in the second dataset remaining.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: Second clear-insert overrides first

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "OldAlice"
  email: "old-alice@example.com"
- id: 2
  name: "OldBob"
  email: "old-bob@example.com"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 10
  name: "NewAlice"
  email: "new-alice@example.com"
- id: 20
  name: "NewBob"
  email: "new-bob@example.com"
```

**Expected Results:**
```yaml
- id: 10
  name: "NewAlice"
  email: "new-alice@example.com"
- id: 20
  name: "NewBob"
  email: "new-bob@example.com"
```
