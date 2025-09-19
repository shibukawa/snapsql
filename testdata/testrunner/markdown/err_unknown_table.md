---
name: "unknown_table_test"
dialect: "sqlite"
---

# Unknown Table Test

## Description

Query references a non-existent table to trigger execution error.

## SQL
```sql
SELECT id, name FROM non_existing_users;
```

## Test Cases

### Test: Select From Unknown Table

**Parameters:**
```yaml
dummy: true
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Jane Doe"
  email: "jane@example.com"
```

**Expected Results:**
```yaml
- dummy: "ignored"
```
