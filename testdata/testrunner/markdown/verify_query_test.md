---
name: "verify_query_test"
dialect: "sqlite"
---

# Verify Query Test

## Description

Test Verify Query functionality with custom SELECT queries.

## Parameters
```yaml
```

## SQL
```sql
INSERT INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com');
```

## Test Cases

### Test: Insert with Verify Query

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User"
  email: "existing@example.com"
```

**Verify Query:**
```sql
SELECT COUNT(*) as total_users FROM users;
SELECT name, email FROM users WHERE email = 'alice@example.com';
```

**Expected Results:**
```yaml
- total_users: 2
- name: "Alice Johnson"
  email: "alice@example.com"
```
