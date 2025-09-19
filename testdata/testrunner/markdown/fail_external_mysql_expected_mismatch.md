````markdown
---
name: "external_expected_mismatch_mysql"
dialect: "mysql"
---

# External expected results mismatch (MySQL)

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
[users fixtures](fixtures_users_mysql.yaml)
```

**Expected Results:**
```yaml
[users expected mismatch](expected_users_mysql_mismatch.yaml)
```

````