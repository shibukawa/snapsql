----
name: "fail_expected_users_pk_match_missing_row"
dialect: "sqlite"
----

# Expected Results pk-match Failure (missing row)

## Description
Validate pk-match fails when a referenced PK row is missing in the table.

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: missing pk row triggers failure

**Parameters:**
```yaml
dummy: 0
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "A"
  email: "a@example.com"
```

**Expected Results: users[pk-match]**
```yaml
- id: 1
- id: 2
```
