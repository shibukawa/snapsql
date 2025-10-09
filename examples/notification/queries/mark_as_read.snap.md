# Mark Notification as Read

## Description

特定の通知を既読にマークします。既に既読の場合は更新されません。

## Parameters

```yaml
notification_id: int
user_id: string
```

## SQL

```sql
UPDATE inbox
SET 
    read_at = NOW(),
    updated_at = NOW()
WHERE notification_id = /*= notification_id */1
    AND user_id = /*= user_id */'EMP001'
    AND read_at IS NULL
```

## Test Cases

### Test: Mark unread notification as read

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
user_id: "EMP001"
```

**Expected Results: inbox[pk-match]**
```yaml
- notification_id: 1
  user_id: "EMP001"
  read_at: [notnull]
```

### Test: Mark already read notification (no update)

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', read_at: '2025-10-01 10:00:00', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
user_id: "EMP001"
```

**Expected Results: inbox[pk-match]**
```yaml
- notification_id: 1
  user_id: "EMP001"
  read_at: "2025-10-01 10:00:00"
```
