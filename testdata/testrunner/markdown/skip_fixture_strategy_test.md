---
description: "Fixture strategy integration test"
dialect: postgres
---

# Fixture Strategy Integration Test

## SQL
```sql
SELECT * FROM users ORDER BY id
```

## Test Cases

### Test: clear-insert strategy
**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
- id: 2
  name: "Bob"
```
**Expected Results:**
```yaml
- id: 1
  name: "Alice"
- id: 2
  name: "Bob"
```

### Test: upsert strategy
**Fixtures: users[upsert]**
```yaml
- id: 1
  name: "Alice-updated"
- id: 3
  name: "Charlie"
```
**Expected Results:**
```yaml
- id: 1
  name: "Alice-updated"
- id: 2
  name: "Bob"
- id: 3
  name: "Charlie"
```

### Test: delete strategy (主キー指定)
**Fixtures: users[delete]**
```yaml
- id: 2
```
**Expected Results:**
```yaml
- id: 1
  name: "Alice-updated"
- id: 3
  name: "Charlie"
```
> ※ delete戦略は主キーのみ指定可能。主キー以外のカラムを指定した場合はエラーとなります。
