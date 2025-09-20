----
name: "fail_matchers_regexp_not_match"
dialect: "sqlite"
----

# Matcher Failure (regexp not match)

## Description
Expect a string to match the given regular expression, but it does not.

## SQL
```sql
SELECT id, comment FROM users ORDER BY id;
```

## Test Cases

### Test: Regexp not matching

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Carol"
  email: "carol@example.com"
  comment: "abc"
```

**Expected Results:**
```yaml
- id: 1
  comment: [regexp, ^foo.*bar$]
```
