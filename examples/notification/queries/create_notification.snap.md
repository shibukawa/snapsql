# Create Notification

## Description

新しい通知を作成します。通知のタイトル、本文、重要度、キャンセル可能フラグ、アイコンURL、有効期限を指定できます。

## Parameters

```yaml
title: string
body: string
important: bool
cancelable: bool
icon_url: string
expires_at: timestamp
```

## SQL

```sql
INSERT INTO notifications (
    title,
    body,
    important,
    cancelable,
    icon_url,
    expires_at
) VALUES (
    /*= title */'Sample Title',
    /*= body */'Sample Body',
    /*= important */false,
    /*= cancelable */false,
    /*= icon_url */'',
    /*= expires_at */''
)
RETURNING id, title, body, important, cancelable, icon_url, expires_at, created_at
```

## Test Cases

### Test: Create basic notification

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
title: "Welcome!"
body: "Welcome to the notification system."
important: false
cancelable: false
icon_url: null
expires_at: null
```

**Expected Results:**
```yaml
- id: [notnull]
  title: "Welcome!"
  body: "Welcome to the notification system."
  important: false
  cancelable: false
  icon_url: [null]
  expires_at: [null]
  created_at: [notnull]
```

### Test: Create important notification with expiration

**Parameters:**
```yaml
title: "System Maintenance"
body: "Scheduled maintenance on Sunday."
important: true
cancelable: true
icon_url: "https://example.com/icons/maintenance.png"
expires_at: "2025-12-31 23:59:59"
```

**Expected Results:**
```yaml
- id: [notnull]
  title: "System Maintenance"
  body: "Scheduled maintenance on Sunday."
  important: true
  cancelable: true
  icon_url: "https://example.com/icons/maintenance.png"
  expires_at: "2025-12-31 23:59:59"
  created_at: [notnull]
```
