# Post Get

## Description

Retrieves a single post by ID with author information (hierarchical response).

## Parameters

```yaml
post_id: int
```

## SQL

```sql
SELECT
    p.post_id,
    p.title,
    p.content,
    p.author_id,
    p.published,
    p.view_count,
    p.created_at,
    p.updated_at,
    u.username as author__username,
    u.full_name as author__full_name
FROM posts p
JOIN users u ON p.author_id = u.user_id
WHERE p.post_id = /*= post_id */1;
```

## Test Cases

### Fetch post with author

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
posts:
  - post_id: 1
    title: "My First Post"
    content: "This is my first blog post"
    author_id: 1
    published: true
    view_count: 10
    created_at: "2025-01-02T10:00:00Z"
    updated_at: "2025-01-02T10:00:00Z"
    created_by: 1
    updated_by: null
```

**Parameters:**
```yaml
post_id: 1
```

**Expected Results:**
```yaml
- post_id: 1
  title: "My First Post"
  content: "This is my first blog post"
  author_id: 1
  published: true
  view_count: 10
  created_at: "2025-01-02T10:00:00Z"
  updated_at: "2025-01-02T10:00:00Z"
  author__username: "alice"
  author__full_name: "Alice Smith"
```
