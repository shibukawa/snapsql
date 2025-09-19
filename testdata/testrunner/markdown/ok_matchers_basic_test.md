----
name: "matchers_basic"
dialect: "sqlite"
----

# Matchers Basic Success Test

## Description
All four matcher types ([null], [notnull], [any], [regexp, ...]) succeed.

## SQL
```sql
SELECT id, name, email, note, comment FROM users ORDER BY id;
```

## Test Cases

### Test: All matcher types

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: null
  note: "HasValue"
  comment: "foo-xyz-bar"
```

**Expected Results:**
```yaml
- id: 1
  name: [any]
  email: [null]
  note: [notnull]
  comment: [regexp, ^foo-.*-bar$]
```
