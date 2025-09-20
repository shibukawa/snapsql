````markdown
---
name: "external_fixture_missing_mysql"
dialect: "mysql"
---

# External fixture missing file (MySQL)

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