# User Create

## Description

Creates a new user account and returns the created user record.

## Parameters

```yaml
username: string
email: string
full_name: string
bio: string
```

## SQL

```sql
INSERT INTO users (
    username,
    email,
    full_name,
    bio
)
VALUES (
    /*= username */'',
    /*= email */'',
    /*= full_name */'',
    /*= bio */''
)
RETURNING
    user_id,
    username,
    email,
    full_name,
    bio,
    created_at,
    updated_at;
```

## Test Cases

### Create user with full information

**Fixtures:**
```yaml
users: []
```

**Parameters:**
```yaml
username: "alice"
email: "alice@example.com"
full_name: "Alice Smith"
bio: "Software engineer and blogger"
```

**Expected Results:**
```yaml
- user_id: 1
  username: "alice"
  email: "alice@example.com"
  full_name: "Alice Smith"
  bio: "Software engineer and blogger"
  created_at: [currentdate]
  updated_at: [currentdate]
```

### Create user with minimal information

**Fixtures:**
```yaml
users: []
```

**Parameters:**
```yaml
username: "bob"
email: "bob@example.com"
full_name: null
bio: null
```

**Expected Results:**
```yaml
- user_id: 1
  username: "bob"
  email: "bob@example.com"
  full_name: null
  bio: null
  created_at: [currentdate]
  updated_at: [currentdate]
```
