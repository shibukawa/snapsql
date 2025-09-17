---
name: "user_operations"
dialect: "postgres"
---

# User Operations Test

## Description

A test to verify fixture insertion strategies for user operations.

## Parameters
```yaml
limit: 10
```

## SQL
```sql
SELECT id, name, email FROM users LIMIT /*= limit */10;
```

## Test Cases

### Test: Basic User Insertion

**Parameters:**
```yaml
limit: 5
```

**Fixtures: users[insert]**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
```

**Expected Results:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
```

### Test: Clear Insert Strategy

**Parameters:**
```yaml
limit: 3
```

**Fixtures: users[clear-insert]**
```yaml
- id: 10
  name: "Alice Johnson"
  email: "alice@example.com"
```

**Expected Results:**
```yaml
- id: 10
  name: "Alice Johnson"
  email: "alice@example.com"
```
