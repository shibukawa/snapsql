---
name: "external_refs_postgres"
dialect: "postgres"
---

# External fixtures and expected results test (PostgreSQL)

## Description

Load fixtures and expected results from external YAML files for PostgreSQL.

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
[users fixtures](fixtures_users_postgres.yaml)
```

**Expected Results:**
```yaml
[users expected](expected_users_postgres.yaml)
```