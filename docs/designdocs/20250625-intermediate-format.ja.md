# SnapSQL 中間形式仕様

**ドキュメントバージョン:** 1.3  
**日付:** 2025-07-23  
**ステータス:** 実装フェーズ

## 概要

このドキュメントは、SnapSQLテンプレートの中間JSON形式を定義します。中間形式は、SQLテンプレートパーサーとコードジェネレータ間の橋渡しとして機能し、解析されたSQLテンプレートのメタデータ、CEL式、関数定義の言語非依存表現を提供します。

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

### 2. 完全な情報保持
- テンプレートメタデータと関数定義
- CEL式の完全な抽出と型情報
- 制御フロー構造（if/forブロック）の命令表現

### 3. コード生成対応
- テンプレートベースのコード生成に適した構造化データ
- 強く型付けされた言語のための型情報
- 関数シグネチャ情報
- パラメータ順序の保持

### 4. 拡張可能
- 将来のSnapSQL機能のサポート
- 後方互換性のためのバージョン管理形式
- カスタムジェネレータ用のプラグインフレンドリー構造

## 中間形式構造

### トップレベル構造

```json
{
  "format_version": 1,
  "function_name": "get_user_by_id",
  "description": "Get user by ID",
  "parameters": [ /* パラメータ定義 */ ],
  "generators": { /* ジェネレータ設定 */ },
  "responses": { /* レスポンス型定義 */ },
  "response_affinity": "one",
  "instructions": [ /* 命令列 */ ],
  "expressions": [ /* CEL式リスト */ ],
  "environments": [ /* 環境変数の階層構造 */ ],
  "cache_keys": [ /* キャッシュキーのインデックス */ ]
}
```

## CEL式の抽出

SnapSQLでは、テンプレート内のすべてのCEL式を抽出します。これには以下が含まれます：

1. **変数置換**: `/*= expression */` 形式の式
2. **条件式**: `/*# if condition */` や `/*# elseif condition */` の条件部分
3. **ループ式**: `/*# for variable : collection */` のコレクション部分

### CEL式の例

```sql
-- 変数置換
SELECT * FROM users WHERE id = /*= user_id */123

-- 条件式
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */

-- ループ式
/*# for dept : departments */
SELECT /*= dept.name */
/*# end */

-- 複雑な式
ORDER BY /*= sort_field + " " + (sort_direction || "ASC") */
```

### 中間形式での表現

```json
{
  "expressions": [
    "user_id",
    "min_age > 0",
    "min_age",
    "departments",
    "dept",
    "dept.name",
    "sort_field + \" \" + (sort_direction || \"ASC\")"
  ],
  "environments": [
    [
      {
        "name": "dept",
        "type": "any"
      }
    ]
  ],
  "cache_keys": [1, 3]
}
```

`environments` セクションには、ループ変数の階層構造が含まれます。各レベルは、そのレベルで定義されたループ変数のリストを含みます。

`cache_keys` セクションには、if/for文で使用されるCEL式のインデックス値の配列が含まれます。これらの式は頻繁に評価される可能性があるため、キャッシュの対象となります。上記の例では、`min_age > 0`（インデックス1）と`departments`（インデックス3）がキャッシュキーとして指定されています。

## 関数定義セクション

関数定義は、テンプレートのヘッダーコメントから抽出されたメタデータを含みます。

```json
{
  "function_name": "get_user_by_id",
  "description": "Get user by ID",
  "parameters": [
    {
      "name": "user_id",
      "type": "int",
      "optional": false
    },
    {
      "name": "include_details",
      "type": "bool",
      "optional": true
    }
  ],
  "generators": {
    "go": {
      "package": "queries",
      "imports": ["context", "database/sql"]
    },
    "typescript": {
      "module": "esm",
      "types": true
    }
  }
}
```

### パラメータ定義

パラメータ定義は、テンプレートのヘッダーコメントから抽出されます。

```yaml
/*#
name: get_user_by_id
function_name: get_user_by_id
description: Get user by ID
parameters:
  user_id: int
  include_details: bool
generators:
  go:
    package: queries
    imports:
      - context
      - database/sql
  typescript:
    module: esm
    types: true
*/
```

## レスポンス型セクション

レスポンス型は、クエリの結果の型情報を示します。

```json
{
  "responses": {
    "name": "User",
    "fields": [
      {
        "name": "id",
        "type": "int",
        "database_tag": "id"
      },
      {
        "name": "name",
        "type": "string",
        "database_tag": "name"
      },
      {
        "name": "email",
        "type": "string",
        "database_tag": "email"
      }
    ]
  },
  "response_affinity": "one"  // "one", "many", "none"のいずれか
}
```

## 命令セット

命令セットは、SQLテンプレートの実行可能な表現です。命令セットは、テンプレートの実行フローを制御し、動的なSQL生成を可能にします。

### 命令タイプ

#### 基本出力命令
- **EMIT_STATIC**: 静的なSQLテキストを出力
  ```json
  {
    "op": "EMIT_STATIC",
    "value": "SELECT id, name FROM users WHERE ",
    "pos": "1:1"
  }
  ```

- **EMIT_EVAL**: CEL式を評価して結果を出力
  ```json
  {
    "op": "EMIT_EVAL",
    "exp_index": 0,
    "placeholder": "123",
    "pos": "1:43"
  }
  ```

#### 制御フロー命令
- **JUMP**: 無条件ジャンプ
  ```json
  {
    "op": "JUMP",
    "target": 5,
    "pos": "5:1"
  }
  ```

- **JUMP_IF**: CEL式の評価結果に基づくジャンプ
  ```json
  {
    "op": "JUMP_IF",
    "exp_index": 1,
    "target": 3,
    "pos": "3:1"
  }
  ```

- **LABEL**: ジャンプ先ラベル
  ```json
  {
    "op": "LABEL",
    "name": "end_if_1",
    "pos": "6:1"
  }
  ```

#### ループ命令
- **LOOP_START**: CEL式で取得したコレクションに対するループ開始
  ```json
  {
    "op": "LOOP_START",
    "variable": "dept",
    "exp_index": 3,
    "env_level": 0,
    "end_label": "loop_end_1",
    "pos": "2:3"
  }
  ```

- **LOOP_END**: ループ終了
  ```json
  {
    "op": "LOOP_END",
    "start_label": "loop_start_1",
    "pos": "17:3"
  }
  ```

#### システム命令
- **SYSTEM_EXPLAIN**: 実行オプションでEXPLAIN句が挿入される場所
  ```json
  {
    "op": "SYSTEM_EXPLAIN"
  }
  ```

- **SYSTEM_JUMP_IF_LIMIT**: 実行時オプションでLIMITが設定された場合にジャンプ
  ```json
  {
    "op": "SYSTEM_JUMP_IF_LIMIT",
    "target": 10
  }
  ```

- **SYSTEM_JUMP_IF_OFFSET**: 実行時オプションでOFFSETが設定された場合にジャンプ
  ```json
  {
    "op": "SYSTEM_JUMP_IF_OFFSET",
    "target": 12
  }
  ```

### 命令列の例

```json
{
  "instructions": [
    {
      "op": "SYSTEM_EXPLAIN"
    },
    {
      "op": "EMIT_STATIC",
      "value": "SELECT id, name FROM users WHERE active = ",
      "pos": "1:1"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 0,
      "placeholder": "true",
      "pos": "1:43"
    },
    {
      "op": "JUMP_IF",
      "exp_index": 1,
      "target": 6,
      "pos": "2:1"
    },
    {
      "op": "EMIT_STATIC",
      "value": " AND age >= ",
      "pos": "3:1"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 2,
      "placeholder": "18",
      "pos": "3:12"
    },
    {
      "op": "LABEL",
      "name": "end_if_1",
      "pos": "4:1"
    },
    {
      "op": "SYSTEM_JUMP_IF_LIMIT",
      "target": 10
    },
    {
      "op": "EMIT_STATIC",
      "value": " LIMIT 100",
      "pos": "5:1"
    },
    {
      "op": "JUMP",
      "target": 11,
      "pos": "5:12"
    },
    {
      "op": "LABEL",
      "name": "end_limit",
      "pos": "6:1"
    }
  ]
}
```

## レスポンスaffinityの決定

レスポンスaffinity（カーディナリティ）は、クエリの結果が何行返されるかを示します。以下の3つの値があります：

- **one**: クエリが単一行を返す場合
- **many**: クエリが複数行を返す場合
- **none**: クエリが行を返さない場合（例：INSERT, UPDATE, DELETE）

### レスポンスaffinityの決定ルール

レスポンスaffinityは、以下のルールに基づいて決定されます：

#### SELECT文

1. **LIMIT 1が指定されている場合**: `one`
   ```sql
   SELECT * FROM users LIMIT 1
   ```

2. **WHERE句に一意キー条件がある場合**: `one`
   ```sql
   SELECT * FROM users WHERE id = 1
   ```

3. **上記以外の場合**: `many`
   ```sql
   SELECT * FROM users WHERE department = 'Engineering'
   ```

#### INSERT文

1. **RETURNINGがあり、バルクインサートの場合**: `many`
   ```sql
   INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com') RETURNING id
   ```

2. **RETURNINGがあり、単一行インサートの場合**: `one`
   ```sql
   INSERT INTO users (name, email) VALUES ('John', 'john@example.com') RETURNING id
   ```

3. **RETURNINGがない場合**: `none`
   ```sql
   INSERT INTO users (name, email) VALUES ('John', 'john@example.com')
   ```

#### UPDATE文

1. **RETURNINGがある場合**: `many`
   ```sql
   UPDATE users SET name = 'John' WHERE department = 'Engineering' RETURNING id, name
   ```

2. **RETURNINGがない場合**: `none`
   ```sql
   UPDATE users SET name = 'John' WHERE department = 'Engineering'
   ```

#### DELETE文

1. **RETURNINGがある場合**: `many`
   ```sql
   DELETE FROM users WHERE department = 'Engineering' RETURNING id, name
   ```

2. **RETURNINGがない場合**: `none`
   ```sql
   DELETE FROM users WHERE department = 'Engineering'
   ```

## レスポンスタイプの型推論

レスポンスタイプは、クエリの結果の型情報を示します。型推論は、以下のルールに基づいて行われます：

### フィールドの型推論ルール

1. **明示的な型指定がある場合**: 指定された型を使用
   ```sql
   SELECT CAST(id AS INT8) AS id
   ```

2. **テーブル情報が利用可能な場合**: テーブル情報から型を取得
   ```sql
   SELECT users.id, users.name FROM users
   ```

3. **フィールドの種類に基づく推論**:
   - **単純フィールド**: デフォルトで `string` 型
   - **テーブルフィールド**: デフォルトで `string` 型
   - **関数フィールド**: 関数名に基づいて型を推論
   - **リテラルフィールド**: リテラル値に基づいて型を推論
   - **複雑なフィールド**: デフォルトで `any` 型

### 関数の戻り値型推論

関数の戻り値型は、関数名に基づいて推論されます：

| 関数パターン | 推論される型 |
|------------|------------|
| `count(*)` | `int` |
| `sum(...)` | `number` |
| `avg(...)` | `number` |
| `min(...)`, `max(...)` | `any` |
| `json_...` | `any` |
| `to_char(...)`, `to_text(...)` | `string` |
| `to_number(...)`, `to_decimal(...)` | `number` |
| `to_date(...)`, `to_timestamp(...)` | `datetime` |
| `coalesce(...)` | `any` |
| その他 | `any` |

### リテラル値の型推論

リテラル値の型は、値の形式に基づいて推論されます：

| リテラルパターン | 推論される型 |
|--------------|------------|
| `'...'` | `string` |
| 整数 | `int` |
| 小数 | `number` |
| `true`, `false` | `bool` |
| `NULL` | `null` |
| その他 | `any` |

## 実装例

### 単純な変数置換

```sql
SELECT id, name, email FROM users WHERE id = /*= user_id */123
```

中間形式：

```json
{
  "format_version": 1,
  "function_name": "get_user_by_id",
  "parameters": [
    {
      "name": "user_id",
      "type": "int"
    }
  ],
  "expressions": [
    "user_id"
  ],
  "instructions": [
    {
      "op": "EMIT_STATIC",
      "value": "SELECT id, name, email FROM users WHERE id = ",
      "pos": "1:1"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 0,
      "placeholder": "123",
      "pos": "1:43"
    }
  ]
}
```

### 条件付きクエリ

```sql
SELECT id, name, age, department 
FROM users
WHERE 1=1
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */
/*# if max_age > 0 */
AND age <= /*= max_age */65
/*# end */
```

中間形式：

```json
{
  "format_version": 1,
  "function_name": "get_filtered_users",
  "parameters": [
    {
      "name": "min_age",
      "type": "int"
    },
    {
      "name": "max_age",
      "type": "int"
    }
  ],
  "expressions": [
    "min_age > 0",
    "min_age",
    "max_age > 0",
    "max_age"
  ],
  "cache_keys": [0, 2],
  "instructions": [
    {
      "op": "EMIT_STATIC",
      "value": "SELECT id, name, age, department \nFROM users\nWHERE 1=1",
      "pos": "1:1"
    },
    {
      "op": "JUMP_IF",
      "exp_index": 0,
      "target": 5,
      "pos": "4:1"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\nAND age >= ",
      "pos": "5:1"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 1,
      "placeholder": "18",
      "pos": "5:11"
    },
    {
      "op": "JUMP",
      "target": 5,
      "pos": "6:1"
    },
    {
      "op": "LABEL",
      "name": "end_if_1",
      "pos": "6:1"
    },
    {
      "op": "JUMP_IF",
      "exp_index": 2,
      "target": 10,
      "pos": "7:1"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\nAND age <= ",
      "pos": "8:1"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 3,
      "placeholder": "65",
      "pos": "8:11"
    },
    {
      "op": "JUMP",
      "target": 10,
      "pos": "9:1"
    },
    {
      "op": "LABEL",
      "name": "end_if_2",
      "pos": "9:1"
    }
  ]
}
```

### ネストされたループ

```sql
SELECT id, name FROM (
  /*# for dept : departments */
  SELECT 
    /*= dept.id */ as dept_id,
    /*= dept.name */ as dept_name,
    (
      /*# for emp : dept.employees */
      SELECT /*= emp.id */, /*= emp.name */
      /*# if !for.last */
      UNION ALL
      /*# end */
      /*# end */
    ) as employees
  /*# if !for.last */
  UNION ALL
  /*# end */
  /*# end */
)
```

中間形式：

```json
{
  "format_version": 1,
  "function_name": "get_nested_data",
  "parameters": [
    {
      "name": "departments",
      "type": "string[]"
    }
  ],
  "expressions": [
    "departments",
    "dept.id",
    "dept.name",
    "dept.employees",
    "emp.id",
    "emp.name",
    "!for.last"
  ],
  "environments": [
    [
      {
        "name": "dept",
        "type": "any"
      }
    ],
    [
      {
        "name": "emp",
        "type": "any"
      }
    ]
  ],
  "cache_keys": [0, 3, 6],
  "instructions": [
    {
      "op": "EMIT_STATIC",
      "value": "SELECT id, name FROM (",
      "pos": "1:1"
    },
    {
      "op": "LOOP_START",
      "variable": "dept",
      "exp_index": 0,
      "env_level": 0,
      "end_label": "loop_end_1",
      "pos": "2:3"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n  SELECT \n    ",
      "pos": "3:3"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 1,
      "placeholder": "1",
      "pos": "4:5"
    },
    {
      "op": "EMIT_STATIC",
      "value": " as dept_id,\n    ",
      "pos": "4:19"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 2,
      "placeholder": "'Engineering'",
      "pos": "5:5"
    },
    {
      "op": "EMIT_STATIC",
      "value": " as dept_name,\n    (",
      "pos": "5:21"
    },
    {
      "op": "LOOP_START",
      "variable": "emp",
      "exp_index": 3,
      "env_level": 1,
      "end_label": "loop_end_2",
      "pos": "7:7"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n      SELECT ",
      "pos": "8:7"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 4,
      "placeholder": "1",
      "pos": "8:14"
    },
    {
      "op": "EMIT_STATIC",
      "value": ", ",
      "pos": "8:24"
    },
    {
      "op": "EMIT_EVAL",
      "exp_index": 5,
      "placeholder": "'John'",
      "pos": "8:29"
    },
    {
      "op": "JUMP_IF",
      "exp_index": 6,
      "target": 15,
      "pos": "9:7"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n      UNION ALL",
      "pos": "10:7"
    },
    {
      "op": "JUMP",
      "target": 16,
      "pos": "11:7"
    },
    {
      "op": "LABEL",
      "name": "end_if_1",
      "pos": "11:7"
    },
    {
      "op": "LOOP_END",
      "start_label": "loop_start_2",
      "pos": "12:7"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n    ) as employees",
      "pos": "13:5"
    },
    {
      "op": "JUMP_IF",
      "exp_index": 6,
      "target": 21,
      "pos": "14:3"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n  UNION ALL",
      "pos": "15:3"
    },
    {
      "op": "JUMP",
      "target": 22,
      "pos": "16:3"
    },
    {
      "op": "LABEL",
      "name": "end_if_2",
      "pos": "16:3"
    },
    {
      "op": "LOOP_END",
      "start_label": "loop_start_1",
      "pos": "17:3"
    },
    {
      "op": "EMIT_STATIC",
      "value": "\n)",
      "pos": "18:1"
    }
  ]
}
```

## JSONスキーマ定義

中間形式には検証用のJSONスキーマ定義が含まれます：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL中間形式",
  "description": "SnapSQLテンプレートの中間JSON形式",
  "type": "object",
  "properties": {
    "format_version": {
      "type": "integer",
      "enum": [1]
    },
    "function_name": {
      "type": "string"
    },
    "description": {
      "type": "string"
    },
    "parameters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "type": { "type": "string" },
          "optional": { "type": "boolean" }
        },
        "required": ["name", "type"]
      }
    },
    "generators": {
      "type": "object",
      "additionalProperties": {
        "type": "object"
      }
    },
    "responses": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "fields": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": { "type": "string" },
              "type": { "type": "string" },
              "database_tag": { "type": "string" }
            },
            "required": ["name", "type"]
          }
        }
      }
    },
    "response_affinity": {
      "type": "string",
      "enum": ["one", "many", "none"]
    },
    "instructions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "op": { 
            "type": "string",
            "enum": [
              "EMIT_STATIC", 
              "EMIT_EVAL", 
              "JUMP", 
              "JUMP_IF", 
              "LABEL", 
              "LOOP_START", 
              "LOOP_END",
              "SYSTEM_EXPLAIN",
              "SYSTEM_JUMP_IF_LIMIT",
              "SYSTEM_JUMP_IF_OFFSET"
            ]
          },
          "value": { "type": "string" },
          "exp_index": { "type": "integer" },
          "placeholder": { "type": "string" },
          "target": { "type": "integer" },
          "name": { "type": "string" },
          "variable": { "type": "string" },
          "env_level": { "type": "integer" },
          "end_label": { "type": "string" },
          "start_label": { "type": "string" },
          "analyze": { "type": "boolean" },
          "pos": { "type": "string" }
        },
        "required": ["op"]
      }
    },
    "expressions": {
      "type": "array",
      "items": { "type": "string" }
    },
    "environments": {
      "type": "array",
      "items": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": { "type": "string" },
            "type": { "type": "string" }
          },
          "required": ["name", "type"]
        }
      }
    },
    "cache_keys": {
      "type": "array",
      "items": { "type": "integer" }
    }
  },
  "required": ["format_version", "instructions"]
}
```

## 今後の拡張

現在の中間形式は、基本的なCEL式の抽出と命令セットの実装に焦点を当てています。将来的には、以下の機能が追加される予定です：

1. **命令セットの最適化**: 命令列の最適化による実行効率の向上
2. **型推論の強化**: より正確な型情報の提供
3. **最適化情報**: クエリの最適化に関する情報
4. **テーブルスキーマ統合**: データベーススキーマ情報との統合

これらの拡張により、より強力なコード生成と実行時の最適化が可能になります。
