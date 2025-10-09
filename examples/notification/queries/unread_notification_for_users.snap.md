# Unread Notification For Users

## Description

通知が更新された際に、その通知を持つすべてのユーザーの既読状態をクリアします。

## Parameters

```yaml
notification_id: int
```

## SQL

```sql
UPDATE inbox
SET read_at = NULL,
    updated_at = NOW()
WHERE notification_id = /*= notification_id */1
```

## Test Cases

### Test: Clear read_at for all users with the notification

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
  - {id: 'EMP002', name: 'Bob', email: 'bob@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Test', body: 'Test Body', important: true, cancelable: false, created_by: 'system', updated_by: 'system'}
inbox:
  - {user_id: 'EMP001', notification_id: 1, read_at: '2024-01-01 10:00:00', delivered_at: '2024-01-01 09:00:00'}
  - {user_id: 'EMP002', notification_id: 1, read_at: '2024-01-01 11:00:00', delivered_at: '2024-01-01 09:00:00'}
```

**Parameters:**
```yaml
notification_id: 1
```

**Expected Results:**
```yaml
[]
```

**Post-Query Check:**
```sql
SELECT user_id, notification_id, read_at FROM inbox WHERE notification_id = 1 ORDER BY user_id
```

**Expected Post-Query Results:**
```yaml
- user_id: 'EMP001'
  notification_id: 1
  read_at: [null]
- user_id: 'EMP002'
  notification_id: 1
  read_at: [null]
```
