----
name: "fail_expected_users_pk_not_exists_present"
dialect: "sqlite"
----

# Expected Results pk-not-exists Failure (row actually present)

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: pk-not-exists row present

**Parameters:**
```yaml
dummy: false
```

**Fixtures: users[clear-insert]**
```yaml
- id: 9
  name: "Nine"
  email: "nine@example.com"
```

**Expected Results: users[pk-not-exists]**
```yaml
- id: 9
```
