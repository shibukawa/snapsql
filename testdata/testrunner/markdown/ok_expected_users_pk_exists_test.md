----
name: "expected_users_pk_exists"
dialect: "sqlite"
----

# Expected Results pk-exists Strategy Test

## Description
Validate that specified primary keys exist in the table after executing the main SQL and fixtures.

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: pk-exists rows present

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 10
  name: "X"
  email: "x@example.com"
- id: 20
  name: "Y"
  email: "y@example.com"
```

**Expected Results: users[pk-exists]**
```yaml
- id: 10
- id: 20
```
