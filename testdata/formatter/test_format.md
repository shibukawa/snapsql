# Database Queries

This document contains various SQL queries for our application.

## User Management

Here's how to get all active users:

```sql
SELECT
    u.id,
    u.name,
    u.email,
    u.created_at
FROM users u
WHERE u.active = true AND u.role IN('admin', 'user')
ORDER BY u.created_at desc
```

## Post Analytics

Get post statistics:

```sql
SELECT
    date_trunc('day',
    p.created_at) AS date,
    COUNT(*) AS post_count,
    COUNT(DISTINCT p.user_id) AS unique_authors
FROM posts p
WHERE p.created_at >= current_date - interval '30 days'
GROUP BY date_trunc('day', p.created_at)
ORDER BY date
```

## Configuration

Some configuration in YAML:

```yaml
database:
  host: localhost
  port: 5432
```

That's all!