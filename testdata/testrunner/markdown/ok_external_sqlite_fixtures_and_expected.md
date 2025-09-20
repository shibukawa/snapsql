---
name: "external_refs_sqlite"
dialect: "sqlite"
---

# External fixtures and expected results test (SQLite)

## Description

Load fixtures and expected results from external YAML files and validate SELECT output.

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
[users fixtures](fixtures_users_sqlite.yaml)
```

**Expected Results:**
```yaml
[users expected](expected_users_sqlite.yaml)
```
