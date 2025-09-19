----
name: "fail_matchers_null_mismatch"
dialect: "sqlite"
----

# Matcher Failure (null mismatch)

## SQL
```sql
SELECT id, email FROM users ORDER BY id;
```

## Test Cases

### Test: Expect null but got value

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  email: "non-null@example.com"
```

**Expected Results:**
```yaml
- id: 1
  email: [null]
```
