# Mark Non-Important Notifications as Read

## Description

ユーザーが通知一覧を閲覧したタイミングで、重要でない（important=false）未読通知をまとめて既読にマークします。

## Parameters

```yaml
user_id: string
```

## SQL

```sql
UPDATE inbox
SET 
    read_at = NOW(),
    updated_at = NOW()
WHERE user_id = /*= user_id */'EMP001'
    AND read_at IS NULL
    AND notification_id IN (
        SELECT id 
        FROM notifications 
        WHERE important = false 
            AND canceled_at IS NULL
            AND (expires_at IS NULL OR expires_at > NOW())
    )
```

## Test Cases

### Test: Mark multiple non-important notifications as read

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', important: false, cancelable: false, created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Important!', body: 'Important message', important: true, cancelable: false, created_by: 'system', updated_by: 'system'}
  - {id: 3, title: 'Info', body: 'Info message', important: false, cancelable: false, created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 3, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP001"
```

**Expected Results: inbox[all]**
```yaml
- notification_id: 1
  user_id: "EMP001"
  read_at: [notnull]
- notification_id: 2
  user_id: "EMP001"
  read_at: [null]
- notification_id: 3
  user_id: "EMP001"
  read_at: [notnull]
```

### Test: Mark non-important when all are already read

**Fixtures:**
```yaml
users:
  - {id: 'EMP002', name: 'Bob', email: 'bob@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', important: false, cancelable: false, created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP002', read_at: '2025-10-01 10:00:00', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP002"
```

**Expected Results: inbox[all]**
```yaml
- notification_id: 1
  user_id: "EMP002"
  read_at: "2025-10-01 10:00:00"
```

### Test: No non-important notifications to mark

**Fixtures:**
```yaml
users:
  - {id: 'EMP003', name: 'Charlie', email: 'charlie@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Important!', body: 'Important message', important: true, cancelable: false, created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP003', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP003"
```

**Expected Results: inbox[all]**
```yaml
- notification_id: 1
  user_id: "EMP003"
  read_at: [null]
```

### Test: Exclude canceled and expired notifications

**Fixtures:**
```yaml
users:
  - {id: 'EMP004', name: 'David', email: 'david@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Active', body: 'Active notification', important: false, cancelable: false, created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Canceled', body: 'Canceled notification', important: false, cancelable: true, canceled_at: '2025-10-01 10:00:00', created_by: 'system', updated_by: 'system'}
  - {id: 3, title: 'Expired', body: 'Expired notification', important: false, cancelable: false, expires_at: '2025-01-01 00:00:00', created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP004', created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP004', created_by: 'system', updated_by: 'system'}
  - {notification_id: 3, user_id: 'EMP004', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP004"
```

**Expected Results: inbox[all]**
```yaml
- notification_id: 1
  user_id: "EMP004"
  read_at: [notnull]
- notification_id: 2
  user_id: "EMP004"
  read_at: [null]
- notification_id: 3
  user_id: "EMP004"
  read_at: [null]
```
