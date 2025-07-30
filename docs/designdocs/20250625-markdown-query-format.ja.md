# Markdownクエリー定義フォーマット

## 概要

SnapSQLはMarkdownベースのクエリー定義ファイル（`.snap.md`）を通じてリテラルプログラミングをサポートします。この形式は、SQLテンプレートと包括的なドキュメント、テストケース、メタデータを単一の読みやすいドキュメントに統合します。

## セクション構成

| セクション/項目 | 必須 | 許容フォーマット | 代替手段 | 個数 |
|----------------|------|------------------|-----------|-------|
| フロントマター | × | YAML | 関数名 + Descriptionセクション | 0-1 |
| 関数名 | ○ | H1タイトル（snake_case自動生成） | フロントマターの`function_name` | 1 |
| Description | ○ | テキスト、Markdown | Overview | 1 |
| Parameters | × | YAML, JSON, リスト形式 | - | 0-1 |
| SQL | ○ | SQL（SnapSQL形式） | - | 1 |
| Test Cases | × | YAML, JSON, リスト形式 | - | 0-n |
| - Fixtures | × | YAML, JSON, CSV, DBUnit XML, リスト形式 | - | 各テストケース0-1 |
| - Parameters | ○ | YAML, JSON, リスト形式 | - | 各テストケース1 |
| - Expected Results | ○ | YAML, JSON, CSV, DBUnit XML, リスト形式 | - | 各テストケース1 |

## セクション詳細

### フロントマター（オプション）

YAMLフォーマットのメタデータ。関数名とDescriptionセクションで代替可能。

```yaml
---
function_name: "get_user_data"  # 明示的な関数名指定（オプション）
description: "Get user data"    # 説明（オプション）
version: "1.0.0"               # その他のメタデータ
---
```

### 関数名（必須）

H1タイトルから自動的にsnake_case形式の関数名を生成。

```markdown
# Get User Data Query
```
↓ 自動変換
```
function_name: "get_user_data_query"
```

### Description（必須）

クエリの目的と説明。`Overview`という見出しでも可。

```markdown
## Description

このクエリは指定されたユーザーIDに基づいてユーザーデータを取得します。
メールアドレスの取得はオプションで制御可能です。
```

### Parameters（オプション）

入力パラメータの定義。複数のフォーマットをサポート。

**YAML形式（推奨）:**
```yaml
user_id: int
include_email: bool
filters:
  active: bool
  departments: [string]
pagination:
  limit: int
  offset: int
```

**JSON形式:**
```json
{
  "user_id": "int",
  "include_email": "bool",
  "filters": {
    "active": "bool",
    "departments": ["string"]
  }
}
```

**リスト形式:**
```markdown
- user_id (int): ユーザーID
- include_email (bool): メールアドレスを含めるかどうか
- filters.active (bool): アクティブユーザーのみ
- filters.departments ([string]): 部署フィルター
```

### SQL（必須）

SnapSQL形式のSQLテンプレート。

```sql
SELECT 
    u.id,
    u.name,
    /*# if include_email */
    u.email,
    /*# end */
    d.id as departments__id,
    d.name as departments__name
FROM users u
    JOIN departments d ON u.department_id = d.id
WHERE u.id = /*= user_id */1
```

### Test Cases（オプション）

テストケースの定義。各テストケースは以下の要素を含むことができます：
- Fixtures: テスト実行前のデータベース状態
- Parameters: テスト入力値
- Expected Results: 期待される結果

#### Fixtures形式例

**YAML形式:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**CSV形式:**
```csv
# users
id,name,email,department_id
1,"John Doe","john@example.com",1
2,"Jane Smith","jane@example.com",2

# departments
id,name
1,"Engineering"
2,"Design"
```

**DBUnit XML形式:**
```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" department_id="1"/>
  <users id="2" name="Jane Smith" email="jane@example.com" department_id="2"/>
  <departments id="1" name="Engineering"/>
  <departments id="2" name="Design"/>
</dataset>
```

#### Parameters形式例

**YAML形式（推奨）:**
```yaml
user_id: 1
include_email: true
```

**JSON形式:**
```json
{
  "user_id": 1,
  "include_email": true
}
```

**リスト形式:**
```markdown
- user_id: 1
- include_email: true
```

#### Expected Results形式例

Fixturesと同じ形式をサポート。

**YAML形式:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

**CSV形式:**
```csv
id,name,email,departments__id,departments__name
1,"John Doe","john@example.com",1,"Engineering"
```

## 完全な例

````markdown
---
version: "1.0.0"
author: "Development Team"
---

# Get User Data Query

## Description

ユーザーIDに基づいてユーザーデータを取得します。
メールアドレスの取得はオプションで制御可能です。

## Parameters

```yaml
user_id: int
include_email: bool
```

## SQL

```sql
SELECT 
    u.id,
    u.name,
    /*# if include_email */
    u.email,
    /*# end */
    d.id as departments__id,
    d.name as departments__name
FROM users u
    JOIN departments d ON u.department_id = d.id
WHERE u.id = /*= user_id */1
```

## Test Cases

### Test: Basic user data

**Fixtures:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**Parameters:**
```yaml
user_id: 1
include_email: true
```

**Expected Results:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

### Test: Without email

**Parameters:**
```yaml
user_id: 2
include_email: false
```

**Expected Results:**
```yaml
- id: 2
  name: "Jane Smith"
  departments__id: 2
  departments__name: "Design"
```
````
