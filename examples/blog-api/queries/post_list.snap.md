# Post List

## Description

Lists published posts with author information, supports pagination.

## Parameters

```yaml
limit: int
offset: int
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
WHERE p.published = true
ORDER BY p.created_at DESC
LIMIT /*= limit */10 OFFSET /*= offset */0;
```

## Test Cases

### List published posts with pagination

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
    content: "Content 1"
    author_id: 1
    published: true
    view_count: 10
    created_at: "2025-01-02T10:00:00Z"
    updated_at: "2025-01-02T10:00:00Z"
    created_by: 1
    updated_by: null
  - post_id: 2
    title: "Second Post"
    content: "Content 2"
    author_id: 2
    published: true
    view_count: 5
    created_at: "2025-01-03T10:00:00Z"
    updated_at: "2025-01-03T10:00:00Z"
    created_by: 2
    updated_by: null
  - post_id: 3
    title: "Draft Post"
    content: "Draft content"
    author_id: 1
    published: false
    view_count: 0
    created_at: "2025-01-04T10:00:00Z"
    updated_at: "2025-01-04T10:00:00Z"
    created_by: 1
    updated_by: null
```

**Parameters:**
```yaml
limit: 10
offset: 0
```

**Expected Results:**
```yaml
- post_id: 2
  title: "Second Post"
  content: "Content 2"
  author_id: 2
  published: true
  view_count: 5
  created_at: "2025-01-03T10:00:00Z"
  updated_at: "2025-01-03T10:00:00Z"
  author__username: "bob"
  author__full_name: "Bob Johnson"
- post_id: 1
  title: "First Post"
  content: "Content 1"
  author_id: 1
  published: true
  view_count: 10
  created_at: "2025-01-02T10:00:00Z"
  updated_at: "2025-01-02T10:00:00Z"
  author__username: "alice"
  author__full_name: "Alice Smith"
```
