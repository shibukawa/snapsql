# Cancel Notification

## Description

キャンセル可能な通知をキャンセルします。キャンセルメッセージと共に記録されます。

## Parameters

```yaml
notification_id: int
cancel_message: string
```

## SQL

```sql
UPDATE notifications
SET 
    cancel_message = /*= cancel_message */'Issue resolved',
    canceled_at = NOW(),
    updated_at = NOW()
WHERE id = /*= notification_id */1
    AND cancelable = true
    AND canceled_at IS NULL
```

## Test Cases

### Test: Cancel a cancelable notification

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'System Error', body: 'Database connection failed', important: true, cancelable: true, created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
cancel_message: "Issue resolved. Connection restored."
```

**Expected Results: notifications[pk-match]**
```yaml
- id: 1
  cancel_message: "Issue resolved. Connection restored."
  canceled_at: [notnull]
```

### Test: Cannot cancel non-cancelable notification

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 2, title: 'Welcome!', body: 'Welcome message', important: false, cancelable: false, created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 2
cancel_message: "Trying to cancel"
```

**Expected Results: notifications[pk-match]**
```yaml
- id: 2
  cancel_message: [null]
  canceled_at: [null]
```

### Test: Cannot cancel already canceled notification

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 3, title: 'Old Error', body: 'Old error message', important: true, cancelable: true, cancel_message: 'Already resolved', canceled_at: '2025-10-01 10:00:00', created_by: 'system', updated_by: 'admin'}
```

**Parameters:**
```yaml
notification_id: 3
cancel_message: "Trying to cancel again"
```

**Expected Results: notifications[pk-match]**
```yaml
- id: 3
  cancel_message: "Already resolved"
  canceled_at: "2025-10-01 10:00:00"
```
