# ok_matchers_currentdate

## Description

Validate currentdate matcher behavior for SQLite dialect.

## SQL

```sql
SELECT CURRENT_TIMESTAMP AS created_at;
```

## Test Cases

### Current timestamp matches default tolerance

**Parameters:**
```yaml
dummy: true
```

**Expected Results:**
```yaml
- created_at: [currentdate]
```
