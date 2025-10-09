# Delete Old and Expired Notifications

## Description

古い通知と有効期限切れの通知を削除します。バッチ処理でのデータクリーンアップ用です。
- 指定日時以前に作成された通知
- 有効期限（expires_at）を過ぎた通知

どちらかの条件に該当する通知とその受信箱データを削除します。

## Parameters

```yaml
before: timestamp  # supports relative token like "[currentdate, -10d]"
```

## SQL

```sql
DELETE FROM notifications
WHERE created_at < /*= before */'2025-01-01 00:00:00'
  OR (expires_at IS NOT NULL AND expires_at <= NOW())
```

## Test Cases

### Test: Delete both old and expired notifications

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Old', body: 'Very old notification', created_at: [currentdate, -100d], created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Expired', body: 'Expired notification', created_at: [currentdate, -20d], expires_at: [currentdate, -1d], created_by: 'system', updated_by: 'system'}
  - {id: 3, title: 'Old and Expired', body: 'Both old and expired', created_at: [currentdate, -200d], expires_at: [currentdate, -5d], created_by: 'system', updated_by: 'system'}
  - {id: 4, title: 'Active Recent', body: 'Recent and active', created_at: [currentdate, -5d], expires_at: [currentdate, +60d], created_by: 'system', updated_by: 'system'}
  - {id: 5, title: 'Recent No Expiration', body: 'Recent with no expiration', created_at: [currentdate, -2d], created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 3, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 4, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 5, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
before: [currentdate, -10d]
```

**Expected Results: notifications[all]**
```yaml
- id: 4
  title: "Active Recent"
  body: "Recent and active"
  created_at: [currentdate, -5d]
  expires_at: [currentdate, +60d]
- id: 5
  title: "Recent No Expiration"
  body: "Recent with no expiration"
  created_at: [currentdate, -2d]
  expires_at: [null]
```

### Test: Delete only old notifications

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Old', body: 'Old notification', created_at: [currentdate, -20d], created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Recent', body: 'Recent notification', created_at: [currentdate, -5d], expires_at: [currentdate, +60d], created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
before: [currentdate, -10d]
```

**Expected Results: notifications[all]**
```yaml
- id: 2
  title: "Recent"
  body: "Recent notification"
  created_at: [currentdate, -5d]
  expires_at: [currentdate, +60d]
```

### Test: Delete only expired notifications

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Expired', body: 'Expired notification', created_at: [currentdate, -20d], expires_at: [currentdate, -1d], created_by: 'system', updated_by: 'system'}
  - {id: 2, title: 'Active', body: 'Active notification', created_at: [currentdate, -2d], expires_at: [currentdate, +60d], created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
  - {notification_id: 2, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
before: [currentdate, -10d]
```

**Expected Results: notifications[all]**
```yaml
- id: 2
  title: "Active"
  body: "Active notification"
  created_at: [currentdate, -2d]
  expires_at: [currentdate, +60d]
```

### Test: Delete with no matching notifications

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Recent Active', body: 'Recent and active', created_at: "[currentdate, -2d]", expires_at: "[currentdate, +60d]", created_by: 'system', updated_by: 'system'}
inbox:
  - {notification_id: 1, user_id: 'EMP001', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
before: [currentdate, -10d]
```

**Expected Results: notifications[all]**
```yaml
- id: 1
  title: "Recent Active"
  body: "Recent and active"
  created_at: [currentdate, -2d]
  expires_at: [currentdate, +60d]
```
