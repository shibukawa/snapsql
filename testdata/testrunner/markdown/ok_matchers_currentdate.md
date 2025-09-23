# ok_matchers_currentdate

## SQL

```sql
SELECT CURRENT_TIMESTAMP AS created_at;
```

## Test Cases

### Current timestamp matches default tolerance

**Expected Results:**
```yaml
- created_at: [currentdate]
```
