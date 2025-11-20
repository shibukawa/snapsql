# Post Create

## Description

Creates a new blog post and returns the created post record.

## Parameters

```yaml
title: string
content: string
author_id: int
published: bool
created_by: int
```

## SQL

```sql
INSERT INTO posts (
    title,
    content,
    author_id,
    published,
    created_by
)
VALUES (
    /*= title */'',
    /*= content */'',
    /*= author_id */1,
    /*= published */false,
    /*= created_by */1
)
RETURNING
    post_id,
    title,
    content,
    author_id,
    published,
    view_count,
    created_at,
    updated_at,
    created_by,
    updated_by;
```

## Test Cases

### Create published post

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
posts: []
```

**Parameters:**
```yaml
title: "Getting Started with FastAPI"
content: "FastAPI is a modern web framework..."
author_id: 1
published: true
created_by: 1
```

**Expected Results:**
```yaml
- post_id: 1
  title: "Getting Started with FastAPI"
  content: "FastAPI is a modern web framework..."
  author_id: 1
  published: true
  view_count: 0
  created_at: [currentdate]
  updated_at: [currentdate]
  created_by: 1
  updated_by: null
```

### Create draft post

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
posts: []
```

**Parameters:**
```yaml
title: "Draft Post"
content: "This is a draft..."
author_id: 1
published: false
created_by: 1
```

**Expected Results:**
```yaml
- post_id: 1
  title: "Draft Post"
  content: "This is a draft..."
  author_id: 1
  published: false
  view_count: 0
  created_at: [currentdate]
  updated_at: [currentdate]
  created_by: 1
  updated_by: null
```
