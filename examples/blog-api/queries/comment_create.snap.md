# Comment Create

## Description

Creates a new comment on a post and returns the created comment record.

## Parameters

```yaml
post_id: int
author_id: int
content: string
```

## SQL

```sql
INSERT INTO comments (
    post_id,
    author_id,
    content
)
VALUES (
    /*= post_id */1,
    /*= author_id */1,
    /*= content */''
)
RETURNING
    comment_id,
    post_id,
    author_id,
    content,
    created_at,
    updated_at;
```

## Test Cases

### Create comment on post

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
    created_at: "2025-01-01T11:00:00Z"
    updated_at: "2025-01-01T11:00:00Z"
posts:
  - post_id: 1
    title: "First Post"
    content: "Content"
    author_id: 1
    published: true
    view_count: 10
    created_at: "2025-01-02T10:00:00Z"
    updated_at: "2025-01-02T10:00:00Z"
    created_by: 1
    updated_by: null
comments: []
```

**Parameters:**
```yaml
post_id: 1
author_id: 2
content: "Great post!"
```

**Expected Results:**
```yaml
- comment_id: 1
  post_id: 1
  author_id: 2
  content: "Great post!"
  created_at: [currentdate]
  updated_at: [currentdate]
```
