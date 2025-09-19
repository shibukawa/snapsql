````markdown
---
name: "external_refs_mysql"
dialect: "mysql"
---

# External fixtures and expected results test (MySQL)

## Description

Load fixtures and expected results from external YAML files for MySQL.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: load fixtures and expected from external files

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
[users fixtures](fixtures_users_mysql.yaml)
```

**Expected Results:**
```yaml
[users expected](expected_users_mysql.yaml)
```

````