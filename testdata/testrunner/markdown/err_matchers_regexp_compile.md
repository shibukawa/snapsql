----
name: "err_matchers_regexp_compile"
dialect: "sqlite"
----

# Regexp Compile Error Test

## SQL
```sql
SELECT id, name FROM users ORDER BY id;
```

## Test Cases

### Test: invalid regexp pattern

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "A"
  email: "a@example.com"
```

**Expected Results:**
```yaml
- id: 1
  name: [regexp, *invalid(
```
