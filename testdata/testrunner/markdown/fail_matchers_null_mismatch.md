----
name: "fail_matchers_null_mismatch"
dialect: "sqlite"
----

# Matcher Failure (null mismatch)

## Description
Expect null but data contains a non-null value to trigger a failure.

## SQL
```sql
SELECT id, email FROM users ORDER BY id;
```

## Test Cases

### Test: Expect null but got value

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Bob"
  email: "non-null@example.com"
```

**Expected Results:**
```yaml
- id: 1
  email: [null]
```
