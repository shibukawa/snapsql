----
name: "matchers_basic"
dialect: "sqlite"
----

# Matchers Basic Success Test

## Description
All four matcher types ([null], [notnull], [any], [regexp, ...]) succeed.

## SQL
```sql
SELECT id, name, email, note, comment, status FROM users ORDER BY id;
```

## Test Cases

### Test: All matcher types

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
  note: "HasValue"
  comment: "foo-xyz-bar"
  status: null
```

**Expected Results:**
```yaml
- id: 1
  name: [any]
  email: "alice@example.com"
  note: [notnull]
  comment: [regexp, ^foo-.*-bar$]
  status: [null]
```
