----
name: "fail_matchers_notnull_mismatch"
dialect: "sqlite"
----

# Matcher Failure (notnull mismatch)

## Description
Expect notnull but data contains null to trigger a failure.

## SQL
```sql
SELECT id, status FROM users ORDER BY id;
```

## Test Cases

### Test: Expect notnull but got null

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
  status: null
```

**Expected Results:**
```yaml
- id: 1
  status: [notnull]
```
