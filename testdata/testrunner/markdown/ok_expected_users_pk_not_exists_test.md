----
name: "expected_users_pk_not_exists"
dialect: "sqlite"
----

# Expected Results pk-not-exists Strategy Test

## Description
Validate that specified primary keys do not exist in the table after executing the main SQL and fixtures.

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: pk-not-exists rows absent

**Parameters:**
```yaml
flag: yes
```

**Fixtures: users[clear-insert]**
```yaml
- id: 100
  name: "X"
  email: "x@example.com"
```

**Expected Results: users[pk-not-exists]**
```yaml
- id: 200
- id: 300
```
