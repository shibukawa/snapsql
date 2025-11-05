# Markdownフォーマット

SnapSQLクエリは `.snap.md` 拡張子を持つMarkdown形式で記述します。このページではファイル全体の構造を説明します。

## 基本構造

`.snap.md` ファイルは以下のセクションで構成されます：

1. **フロントマター**（オプション） - メタデータ（YAML形式）
2. **Description**（必須） - クエリの説明
3. **Parameters**（オプション） - パラメータ定義
4. **SQL**（必須） - SQLクエリ本体
5. **Test Cases**（オプション） - テストケース

### 完全な例

````markdown
---
function_name: find_user  # ファイル名と異なる場合のみ指定
---

# Find User Query

## Description

This query finds a user by their ID.

## Parameters

```yaml
user_id: int
include_email: bool
```

## SQL

```sql
SELECT
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    created_at
FROM
    users
WHERE
    id = /*= user_id */1;
```

## Test Cases

### Test Case 1: Find existing user

**Fixtures:**
```yaml
users:
users:
  - id: 123
    name: "John Doe"
    email: "john@example.com"
    age: 30
```

**Parameters:**
```yaml
user_id: 123
include_email: true
```

**Expected Results:**
```yaml
- id: 123
  name: "John Doe"
  email: "john@example.com"
```

### Test Case 2: User without email

**Parameters:**
```yaml
user_id: 456
include_email: false
```

**Expected Results:**
```yaml
- id: 456
  name: "Jane Smith"
```
````

## セクション詳細

### フロントマター（オプション）

YAML形式でメタデータを記述します。**`function_name`はファイル名から自動生成されるため、ファイル名と異なる名前にしたい場合のみ指定してください。**

```yaml
---
function_name: get_user_data  # ファイル名と異なる場合のみ指定
description: "Get user data"   # オプション
dialect: postgres              # オプション
---
```

**注意事項:**
- 関数名はファイル名（拡張子を除く）から自動決定されます
  - 例: `get_users.snap.md` → 関数名: `get_users`
- フロントマターを省略し、Descriptionセクションだけで記述することも可能です

### Description セクション（必須）

クエリの目的と説明を記述します。H2見出し（`## Description`）または`## Overview`を使用します。

```markdown
## Description

このクエリは指定されたユーザーIDに基づいてユーザーデータを取得します。
メールアドレスの取得はオプションで制御可能です。
```

### Parameters セクション（オプション）

入力パラメータの型を定義します。**型のみをシンプルに記述します**。

**YAML形式（推奨）:**
```yaml
user_id: int
include_email: bool
status: string
```

**ネストしたパラメータ:**
```yaml
user_id: int
filters:
  active: bool
  departments: string[]
pagination:
  limit: int
  offset: int
```

**JSON形式も可能:**
```json
{
  "user_id": "int",
  "include_email": "bool",
  "status": "string"
}
```

利用できる型などの詳細は [parameters.md](./parameters.md) を参照してください。

### SQL セクション（必須）

SnapSQL形式のSQLテンプレートを記述します。**言語指定 `sql` のフェンスコードが必須です。**

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

テンプレート構文の詳細は [template-syntax.md](./template-syntax.md) を参照してください。

### Test Cases セクション（オプション）

テストケースを定義します。**H3見出し（`###`）でテストケース名を記述し、その下に太字（`**...**`）でサブセクションを記述します。**

#### サブセクションのラベル

以下の太字ラベルを使用します（見出しではありません）：

- **`**Fixtures:**`** - テストデータ（オプション、複数可）
- **`**Parameters:**`** - 入力パラメータ（必須、1回のみ）
- **`**Expected Results:**`** - 期待される結果（必須、1回のみ）
- **`**Verify Query:**`** - 検証用クエリ（オプション）

#### 基本例

````markdown
## Test Cases

### Test: Basic user data

**Fixtures:**
```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
departments:
  - id: 1
    name: "Engineering"
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
```
````

#### Fixturesの挿入戦略

テーブルごとに挿入戦略を指定できます：

- **`[clear-insert]`** - テーブルをクリアしてから挿入（デフォルト）
- **`[upsert]`** - 既存行があれば更新、なければ挿入
- **`[delete]`** - 指定データに一致する行を削除

````markdown
**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
```

**Fixtures: users[upsert]**
```yaml
- id: 2
  name: "Bob"
```
````

#### CSV形式のFixtures

CSV形式も使用できます。**テーブル名と戦略の指定が必須です。**

````markdown
**Fixtures: users[clear-insert]**
```csv
id,name,email,department_id
1,John Doe,john@example.com,1
2,Jane Smith,jane@example.com,2
```
````

#### 外部ファイルの参照

フィクスチャは外部ファイル化でき、テストケース間で共有できます。

```markdown
**Fixtures:**
[共通ユーザーデータ](../fixtures/common_users.yaml)

**Fixtures: users[upsert]**
[追加ユーザーデータ](../fixtures/additional_users.yaml)
```

詳細は [fixtures.md](./fixtures.md) を参照してください。

## ファイル命名規則

- `.snap.md` 拡張子を使用
- ファイル名は小文字とアンダースコアを推奨
- 例: `get_user_by_id.snap.md`, `create_task.snap.md`

## ディレクトリ構成例

```
queries/
├── users/
│   ├── get_user_by_id.snap.md
│   ├── list_users.snap.md
│   └── create_user.snap.md
├── tasks/
│   ├── get_task.snap.md
│   └── update_task.snap.md
└── shared/
    └── common_fixtures.yaml
```

## 関連ドキュメント

- [テンプレート構文](./template-syntax.md)
- [パラメータ](./parameters.md)
- [レスポンス型](./response-types.md)
- [フィクスチャ](./fixtures.md)
- [Expected Results](./expected-results.md)
- [エラーテスト](./error-testing.md)
