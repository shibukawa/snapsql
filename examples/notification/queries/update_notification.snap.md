# Update Notification

## Description

通知のタイトル、本文、重要度フラグを更新します。

## Parameters

```yaml
notification_id: int
title: string
body: string
important: bool
```

## SQL

```sql
UPDATE notifications
SET 
    title = /*= title */'Updated Title',
    body = /*= body */'Updated Body',
    important = /*= important */false,
    updated_at = NOW()
WHERE id = /*= notification_id */1
RETURNING id, title, body, important, cancelable, icon_url, expires_at, created_at, updated_at
```

## Test Cases

### Test: Update notification successfully

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Old Title', body: 'Old Body', important: false, cancelable: true, created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
title: "New Title"
body: "New Body"
important: true
```

**Expected Results:**
```yaml
- id: 1
  title: "New Title"
  body: "New Body"
  important: true
  cancelable: true
  icon_url: [null]
  expires_at: [null]
  created_at: [notnull]
  updated_at: [notnull]
```

### Test: Update only title

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 2, title: 'Original', body: 'Original Body', important: true, cancelable: false, created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 2
title: "Modified Title"
body: "Original Body"
important: true
```

**Expected Results:**
```yaml
- id: 2
  title: "Modified Title"
  body: "Original Body"
  important: true
  cancelable: false
```
