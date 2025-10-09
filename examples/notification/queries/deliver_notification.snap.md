# Deliver Notification to Users

## Description

特定の通知を複数のユーザーの受信箱に一括配信します。既に配信済みのユーザーには何もしません。

## Parameters

```yaml
notification_id: int
user_ids: [string]
```

## SQL

```sql
INSERT INTO inbox (
    notification_id,
    user_id
) VALUES
/*# for user_id : user_ids */
    (
        /*= notification_id */1,
        /*= user_id */'EMP001'
    ),
/*# end */
RETURNING notification_id, user_id, read_at, created_at
```

## Test Cases

### Test: Deliver notification to multiple users

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
  - {id: 'EMP002', name: 'Bob', email: 'bob@example.com', created_by: 'system', updated_by: 'system'}
  - {id: 'EMP003', name: 'Charlie', email: 'charlie@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome to the system.', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
user_ids: ["EMP001", "EMP002", "EMP003"]
```

**Expected Results:**
```yaml
- notification_id: 1
  user_id: "EMP001"
  read_at: [null]
  created_at: [notnull]
- notification_id: 1
  user_id: "EMP002"
  read_at: [null]
  created_at: [notnull]
- notification_id: 1
  user_id: "EMP003"
  read_at: [null]
  created_at: [notnull]
```

### Test: Deliver to single user

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 2, title: 'Update', body: 'System update.', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 2
user_ids: ["EMP001"]
```

**Expected Results:**
```yaml
- notification_id: 2
  user_id: "EMP001"
  read_at: [null]
  created_at: [notnull]
```

### Test: Deliver notification with some users already exist (conflict)

**Fixtures:**
```yaml
users:
  - {id: 'EMP001', name: 'Alice', email: 'alice@example.com', created_by: 'system', updated_by: 'system'}
  - {id: 'EMP002', name: 'Bob', email: 'bob@example.com', created_by: 'system', updated_by: 'system'}
notifications:
  - {id: 1, title: 'Welcome!', body: 'Welcome to the system.', created_by: 'system', updated_by: 'system'}
```

**Parameters:**
```yaml
notification_id: 1
user_ids: ["EMP001", "EMP002"]
```

**Expected Results:**
```yaml
- notification_id: 1
  user_id: "EMP001"
  read_at: [null]
  created_at: [notnull]
- notification_id: 1
  user_id: "EMP002"
  read_at: [null]
  created_at: [notnull]
```
