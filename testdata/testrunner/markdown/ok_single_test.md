---
name: "single_test"
dialect: "postgres"
---

# Single Test Example

## Description

A single test case for fixture-only mode testing.

## Parameters
```yaml
limit: 10
```

## SQL
```sql
SELECT id, name, email FROM users LIMIT /*= limit */10;
```

## Test Cases

### Test: Single User Test

**Parameters:**
```yaml
limit: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 100
  name: "Single User"
  email: "single@example.com"
```

**Expected Results:**
```yaml
- id: 100
  name: "Single User"
  email: "single@example.com"
```
