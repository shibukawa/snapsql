# Get User Posts

## Description

This query retrieves user posts with optional filtering and conditional fields.

## Parameters

```yaml
user_id: int
include_drafts: bool
include_metadata: bool
```

## SQL

```sql
select u.id,u.name,u.email /*# if include_metadata */ ,u.created_at,u.updated_at /*# end */ ,p.id as post_id,p.title /*# if include_drafts */ ,p.status /*# end */ from users u join posts p on u.id=p.user_id where u.id=/*= user_id */ /*# if !include_drafts */ and p.status='published' /*# end */ order by p.created_at desc limit /*= limit != 0 ? limit : 10 */
```

## Test Cases

### Test: Basic user posts

**Parameters:**
```yaml
user_id: 123
include_drafts: false
include_metadata: true
```

**Expected Results:**
```yaml
- {id: 123, name: "John Doe", email: "john@example.com", created_at: "2024-01-15T10:30:00Z"}
```
