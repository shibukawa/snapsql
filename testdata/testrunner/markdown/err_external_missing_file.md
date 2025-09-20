---
name: "external_missing_file"
dialect: "sqlite"
---

# External file error test

## SQL
```sql
SELECT 1;
```

## Test Cases

### Test: missing external expected file should error

**Parameters:**
```yaml
x: 1
```

**Expected Results:**
```yaml
[missing expected](no_such_file.yaml)
```
