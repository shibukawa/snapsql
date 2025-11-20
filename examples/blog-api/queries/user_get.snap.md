# User Get

## Description

Retrieves a single user by their ID.

## Parameters

```yaml
user_id: int
```

## SQL

```sql
SELECT
    user_id,
    username,
    email,
    full_name,
    bio,
    created_at,
    updated_at
FROM users
WHERE user_id = /*= user_id */1;
```

## Test Cases

### Fetch existing user

**Fixtures:**
```yaml
users:
  - user_id: 1
    username: "alice"
    email: "alice@example.com"
    full_name: "Alice Smith"
    bio: "Software engineer"
    created_at: "2025-01-01T10:00:00Z"
    updated_at: "2025-01-01T10:00:00Z"
```

**Parameters:**
```yaml
user_id: 1
```

**Expected Results:**
```yaml
- user_id: 1
  username: "alice"
  email: "alice@example.com"
  full_name: "Alice Smith"
  bio: "Software engineer"
  created_at: "2025-01-01T10:00:00Z"
  updated_at: "2025-01-01T10:00:00Z"
```
