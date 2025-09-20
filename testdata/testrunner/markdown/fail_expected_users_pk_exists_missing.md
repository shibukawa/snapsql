----
name: "fail_expected_users_pk_exists_missing"
dialect: "sqlite"
----

# Expected Results pk-exists Failure (one row absent)

## Description
Validate pk-exists fails when one of the specified PK rows does not exist.

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: pk-exists missing row

**Parameters:**
```yaml
dummy: 2
```

**Fixtures: users[clear-insert]**
```yaml
- id: 5
  name: "Foo"
  email: "f@example.com"
```

**Expected Results: users[pk-exists]**
```yaml
- id: 5
- id: 6
```
