````markdown
---
name: "external_fixture_missing_sqlite"
dialect: "sqlite"
---

# External fixture missing file (SQLite)

## Description

Reference a missing external fixtures YAML file to ensure runtime error handling.

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: missing external fixtures file should error

**Parameters:**
```yaml
x: 1
```

**Fixtures: users[clear-insert]**
```yaml
[missing fixtures](no_such_fixture.yaml)
```

**Expected Results:**
```yaml
- {}
```

````