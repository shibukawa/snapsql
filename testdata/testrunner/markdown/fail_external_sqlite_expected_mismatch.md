---
name: "external_expected_mismatch_sqlite"
dialect: "sqlite"
---

# External expected results mismatch (SQLite)

## Description

Load fixtures from external YAML and expected results from another external YAML that intentionally mismatches.

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
[users fixtures](fixtures_users_sqlite.yaml)
```

**Expected Results:**
```yaml
[users expected mismatch](expected_users_sqlite_mismatch.yaml)
```