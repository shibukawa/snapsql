# Database Queries

This document contains various SQL queries for our application.

## User Management

Here's how to get all active users:

```sql
select u.id,u.name,u.email,u.created_at from users u where u.active=true and u.role in ('admin','user') order by u.created_at desc
```

## Post Analytics

Get post statistics:

```sql
select date_trunc('day',p.created_at) as date,count(*) as post_count,count(distinct p.user_id) as unique_authors from posts p where p.created_at >= current_date - interval '30 days' group by date_trunc('day',p.created_at) order by date
```

## Configuration

Some configuration in YAML:

```yaml
database:
  host: localhost
  port: 5432
```

That's all!
