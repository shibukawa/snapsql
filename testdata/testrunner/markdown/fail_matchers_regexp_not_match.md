----
name: "fail_matchers_regexp_not_match"
dialect: "sqlite"
----

# Matcher Failure (regexp not match)

## SQL
```sql
SELECT id, comment FROM users ORDER BY id;
```

## Test Cases

### Test: Regexp not matching

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  comment: "abc"
```

**Expected Results:**
```yaml
- id: 1
  comment: [regexp, ^foo.*bar$]
```
