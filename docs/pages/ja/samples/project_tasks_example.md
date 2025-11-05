# Project Tasks Example

このページはこのチュートリアルで使うサンプルクエリとテストフィクスチャをまとめたものです。

## Query: Get Project Tasks

```markdown
# Get Project Tasks

## Description

指定されたプロジェクトの全タスクを取得します。

## Parameters

```yaml
project_id: int
```

## SQL

```sql
SELECT
    t.id,
    t.title,
    t.description,
    t.status,
    u.name as assignee_name
FROM tasks t
LEFT JOIN users u ON t.assignee_id = u.id
WHERE t.project_id = /*= project_id */1
ORDER BY t.created_at DESC;
```

## Test Cases

### Get tasks for existing project

**Parameters:**
```yaml
project_id: 1
```

**Fixtures:**
```yaml
projects:
  - id: 1
    name: "Web App"
users:
  - id: 1
    name: "Alice"
tasks:
  - id: 1
    project_id: 1
    title: "Design homepage"
    assignee_id: 1
  - id: 2
    project_id: 1
    title: "Implement login"
```

**Expected Results:**
```yaml
- id: 1
  title: "Design homepage"
  description: null
  status: "todo"
  assignee_name: "Alice"
- id: 2
  title: "Implement login"
  description: null
  status: "todo"
  assignee_name: null
```
```
