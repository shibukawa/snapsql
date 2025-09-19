----
name: "fail_matchers_notnull_mismatch"
dialect: "sqlite"
----

# Matcher Failure (notnull mismatch)

## SQL
```sql
SELECT id, email FROM users ORDER BY id;
```

## Test Cases

### Test: Expect notnull but got null

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  email: null
```

**Expected Results:**
```yaml
- id: 1
  email: [notnull]
```
