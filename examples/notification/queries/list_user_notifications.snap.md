# List User Notifications

## Description

ユーザーの受信箱にある通知一覧を取得します。未読/既読、重要度、有効期限でフィルタリングでき、ページネーションに対応しています。キャンセルされた通知は除外されます。

## Parameters

```yaml
user_id: string
unread_only: bool
since: datetime
```

## SQL

```sql
SELECT 
    n.id,
    n.title,
    n.body,
    n.icon_url,
    n.important,
    n.cancelable,
    n.expires_at,
    n.created_at,
    n.updated_at,
    i.read_at,
    i.created_at as delivered_at
FROM inbox i
    JOIN notifications n ON i.notification_id = n.id
WHERE i.user_id = /*= user_id */'EMP001'
    /*# if unread_only */
    AND i.read_at IS NULL
    /*# end */
    /*# if since */
    AND i.updated_at > /*= since */'2025-01-01 00:00:00'
    /*# end */
    AND n.canceled_at IS NULL
    AND (n.expires_at IS NULL OR n.expires_at > NOW())
ORDER BY i.updated_at DESC
```

## Test Cases

### Test: List all notifications for user

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', important: false, cancelable: false, created_at: "[currentdate, -5m]", updated_at: "[currentdate, -5m]", created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Important!', body: 'Important message', important: true, cancelable: false, created_at: "[currentdate, -1m]", updated_at: "[currentdate, -1m]", created_by: 'system', updated_by: 'system'}
  - {id: 3, title: 'Canceled', body: 'This is canceled', important: false, cancelable: true, canceled_at: "[currentdate]", created_at: "[currentdate]", updated_at: "[currentdate]", created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', read_at: "[currentdate, -1h]", created_at: "[currentdate, -5m]", updated_at: "[currentdate, -5m]", created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_at: "[currentdate, -1m]", updated_at: "[currentdate, -1m]", created_by: 'system', updated_by: 'system'}
  - {notification_id: 3, user_id: 'EMP001', created_at: "[currentdate]", updated_at: "[currentdate]", created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP001"
unread_only: false
since: null
```

**Expected Results:**
```yaml
- id: 2
  title: "Important!"
  body: "Important message"
  icon_url: [null]
  important: true
  cancelable: false
  expires_at: [null]
  created_at: [notnull]
  read_at: [null]
  delivered_at: [notnull]
- id: 1
  title: "Welcome!"
  body: "Welcome message"
  icon_url: [null]
  important: false
  cancelable: false
  expires_at: [null]
  created_at: [notnull]
  read_at: [notnull]
  delivered_at: [notnull]
```

### Test: List unread notifications only

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome message', important: false, cancelable: false, created_at: "[currentdate, -5m]", updated_at: "[currentdate, -5m]", created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Important!', body: 'Important message', important: true, cancelable: false, created_at: "[currentdate, -1m]", updated_at: "[currentdate, -1m]", created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', read_at: "[currentdate, -1h]", created_at: "[currentdate, -5m]", updated_at: "[currentdate, -5m]", created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_at: "[currentdate, -1m]", updated_at: "[currentdate, -1m]", created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP001"
unread_only: true
since: null
```

**Expected Results:**
```yaml
- id: 2
  title: "Important!"
  body: "Important message"
  icon_url: [null]
  important: true
  cancelable: false
  expires_at: [null]
  created_at: [notnull]
  read_at: [null]
  delivered_at: [notnull]
```

### Test: List notifications since specific time (polling with updated_at)

**Fixtures:**
```yaml
users:
  - {id: 'EMP002', name: 'Bob', email: 'bob@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 10, title: 'Old Notification', body: 'This is old', important: false, cancelable: false, created_at: "[currentdate, -1d]", updated_at: "[currentdate, -1d]", created_by: 'system', updated_by: 'system'}
  - {id: 11, title: 'New Notification 1', body: 'This is new', important: false, cancelable: false, created_at: "[currentdate, -2m]", updated_at: "[currentdate, -2m]", created_by: 'system', updated_by: 'system'}
  - {id: 12, title: 'New Notification 2', body: 'This is also new', important: true, cancelable: false, created_at: "[currentdate]", updated_at: "[currentdate]", created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 10, user_id: 'EMP002', created_at: '2025-10-01 10:00:00', updated_at: '2025-10-01 10:00:00', created_by: 'system', updated_by: 'system'}
  - {notification_id: 11, user_id: 'EMP002', created_at: '2025-10-02 15:00:00', updated_at: '2025-10-02 15:00:00', created_by: 'system', updated_by: 'system'}
  - {notification_id: 12, user_id: 'EMP002', created_at: '2025-10-03 09:00:00', updated_at: '2025-10-03 09:00:00', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
user_id: "EMP002"
unread_only: false
since: "2025-10-01 12:00:00"
```

**Expected Results:**
```yaml
- id: 12
  title: "New Notification 2"
  body: "This is also new"
  icon_url: [null]
  important: true
  cancelable: false
  expires_at: [null]
  created_at: [notnull]
  read_at: [null]
  delivered_at: "2025-10-03 09:00:00"
- id: 11
  title: "New Notification 1"
  body: "This is new"
  icon_url: [null]
  important: false
  cancelable: false
  expires_at: [null]
  created_at: [notnull]
  read_at: [null]
  delivered_at: "2025-10-02 15:00:00"
```
