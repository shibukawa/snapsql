---
name: "syntax_error_test"
dialect: "sqlite"
---

# Syntax Error Test

## Description

Intentional SQL syntax error to ensure executor reports execution failure (should be treated as fatal, not simple validation failure).

## SQL
```sql
SELEC id, name FROM users; -- misspelled SELECT
```

## Test Cases

### Test: Syntax Error Execution

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
```

**Expected Results:**
```yaml
- dummy: "ignored"
```
