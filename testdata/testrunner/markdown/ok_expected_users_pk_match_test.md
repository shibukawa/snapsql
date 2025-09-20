----
name: "expected_users_pk_match"
dialect: "sqlite"
----

# Expected Results pk-match Strategy Test

## Description
Validate pk-match strategy: only listed PK rows must match provided column values.

## SQL
```sql
-- No returning; we rely on table state validation only
SELECT 1; -- dummy
```

## Test Cases

### Test: pk-match two rows

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "a@example.com"
- id: 2
  name: "Bob"
  email: "b@example.com"
- id: 3
  name: "Charlie"
  email: "c@example.com"
```

**Expected Results: users[pk-match]**
```yaml
- id: 1
  name: "Alice"
- id: 3
  email: "c@example.com"
```
