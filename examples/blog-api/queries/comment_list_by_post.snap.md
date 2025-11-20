# Comment List By Post

## Description

Lists all comments for a specific post with author information, ordered by creation date.

## Parameters

```yaml
post_id: int
```

## SQL

```sql
SELECT
    c.comment_id,
    c.post_id,
    c.author_id,
    c.content,
    c.created_at,
    c.updated_at,
    u.username as author__username,
    u.full_name as author__full_name
FROM comments c
JOIN users u ON c.author_id = u.user_id
WHERE c.post_id = /*= post_id */1
ORDER BY c.created_at ASC;
```

## Test Cases

### List comments for a post

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
  - user_id: 3
    username: "charlie"
    email: "charlie@example.com"
    full_name: "Charlie Brown"
    bio: "Writer"
    created_at: "2025-01-01T12:00:00Z"
    updated_at: "2025-01-01T12:00:00Z"
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
comments:
  - comment_id: 1
    post_id: 1
    author_id: 2
    content: "Great post!"
    created_at: "2025-01-02T11:00:00Z"
    updated_at: "2025-01-02T11:00:00Z"
  - comment_id: 2
    post_id: 1
    author_id: 3
    content: "Thanks for sharing!"
    created_at: "2025-01-02T12:00:00Z"
    updated_at: "2025-01-02T12:00:00Z"
  - comment_id: 3
    post_id: 1
    author_id: 1
    content: "Glad you liked it!"
    created_at: "2025-01-02T13:00:00Z"
    updated_at: "2025-01-02T13:00:00Z"
```

**Parameters:**
```yaml
post_id: 1
```

**Expected Results:**
```yaml
- comment_id: 1
  post_id: 1
  author_id: 2
  content: "Great post!"
  created_at: "2025-01-02T11:00:00Z"
  updated_at: "2025-01-02T11:00:00Z"
  author__username: "bob"
  author__full_name: "Bob Johnson"
- comment_id: 2
  post_id: 1
  author_id: 3
  content: "Thanks for sharing!"
  created_at: "2025-01-02T12:00:00Z"
  updated_at: "2025-01-02T12:00:00Z"
  author__username: "charlie"
  author__full_name: "Charlie Brown"
- comment_id: 3
  post_id: 1
  author_id: 1
  content: "Glad you liked it!"
  created_at: "2025-01-02T13:00:00Z"
  updated_at: "2025-01-02T13:00:00Z"
  author__username: "alice"
  author__full_name: "Alice Smith"
```
