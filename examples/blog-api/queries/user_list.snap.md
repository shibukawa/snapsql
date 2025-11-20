# User List

## Description

Lists all users ordered by creation date (newest first).

## Parameters

```yaml
# No parameters
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
ORDER BY created_at DESC;
```

## Test Cases

### List multiple users

**Fixtures:**
```yaml
users:
  - user_id: 1
    username: "alice"
    email: "alice@example.com"
    full_name: "Alice Smith"
    bio: "Engineer"
    created_at: "2025-01-01T10:00:00Z"
    updated_at: "2025-01-01T10:00:00Z"
  - user_id: 2
    username: "bob"
    email: "bob@example.com"
    full_name: "Bob Johnson"
    bio: "Designer"
    created_at: "2025-01-02T10:00:00Z"
    updated_at: "2025-01-02T10:00:00Z"
  - user_id: 3
    username: "charlie"
    email: "charlie@example.com"
    full_name: "Charlie Brown"
    bio: "Writer"
    created_at: "2025-01-03T10:00:00Z"
    updated_at: "2025-01-03T10:00:00Z"
```

**Parameters:**
```yaml
# No parameters
```

**Expected Results:**
```yaml
- user_id: 3
  username: "charlie"
  email: "charlie@example.com"
  full_name: "Charlie Brown"
  bio: "Writer"
  created_at: "2025-01-03T10:00:00Z"
  updated_at: "2025-01-03T10:00:00Z"
- user_id: 2
  username: "bob"
  email: "bob@example.com"
  full_name: "Bob Johnson"
  bio: "Designer"
  created_at: "2025-01-02T10:00:00Z"
  updated_at: "2025-01-02T10:00:00Z"
- user_id: 1
  username: "alice"
  email: "alice@example.com"
  full_name: "Alice Smith"
  bio: "Engineer"
  created_at: "2025-01-01T10:00:00Z"
  updated_at: "2025-01-01T10:00:00Z"
```
