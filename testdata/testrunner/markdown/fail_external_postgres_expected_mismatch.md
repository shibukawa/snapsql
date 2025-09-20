---
name: "external_expected_mismatch_postgres"
dialect: "postgres"
---

# External expected results mismatch (PostgreSQL)

## Description

Load fixtures/expected from external YAML; expected data intentionally mismatches to trigger comparison failure.

## SQL
```sql
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: expected mismatch from external file

**Parameters:**
```yaml
note: mismatch
```

**Fixtures: users[clear-insert]**
```yaml
[users fixtures](fixtures_users_postgres.yaml)
```

**Expected Results:**
```yaml
[users expected mismatch](expected_users_postgres_mismatch.yaml)
```