# SnapSQL 中間形式仕様

**ドキュメントバージョン:** 1.2  
**日付:** 2025-07-18  
**ステータス:** 実装フェーズ

## 概要

このドキュメントは、SnapSQLテンプレートの中間JSON形式を定義します。中間形式は、SQLテンプレートパーサーとコードジェネレータ間の橋渡しとして機能し、解析されたSQLテンプレートのメタデータ、命令列、インターフェーススキーマの言語非依存表現を提供します。

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

### 2. 完全な情報保持
- 命令列による効率的なテンプレート表現
- テンプレートメタデータとインターフェーススキーマ
- 変数参照とその型情報
- 制御フロー構造（if/forブロック）

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
  "interface_schema": { /* InterfaceSchemaオブジェクト */ },
  "response_type": { /* ResponseTypeオブジェクト */ },
  "response_affinity": "one",
  "instructions": [ /* 命令列 */ ],
  "cel_expressions": [ /* 複雑なCEL式リスト */ ],
  "envs": [ /* 環境変数の階層構造 */ ]
}
```

## CEL式と単純変数の区別

### 単純変数と複雑なCEL式

SnapSQLでは、テンプレート内の変数参照を2つのカテゴリに分類します：

1. **単純変数**: ピリオドや演算子を含まない単純な変数名（例: `active`, `min_age`, `include_email`）
2. **複雑なCEL式**: 演算子、メソッド呼び出し、または複合参照を含む式（例: `min_age > 0`, `departments != null && departments.size() > 0`, `dept.employees`）

この区別により、実行時の効率が向上します：

- **単純変数**: `map[string]any` から直接アクセスでき、CEL環境の評価が不要
- **複雑なCEL式**: CEL環境での評価が必要

### 命令タイプの拡張

単純変数と複雑なCEL式を区別するために、以下の命令タイプが追加されています：

- **JUMP_IF_PARAM**: 単純変数の真偽値に基づくジャンプ
- **LOOP_START_PARAM**: 単純変数のコレクションに対するループ開始
- **LOOP_START_EXP**: CEL式で取得したコレクションに対するループ開始

### 中間形式での表現

```json
{
  "instructions": [
    {
      "op": "JUMP_IF_PARAM",
      "param": "active",  // 単純変数名を直接指定
      "target": 3,
      "pos": [1, 1, 0]
    },
    {
      "op": "JUMP_IF_EXP",
      "exp_index": 0,  // CEL式リストの参照
      "target": 5,
      "pos": [2, 1, 50]
    },
    {
      "op": "LOOP_START_PARAM",
      "variable": "item",
      "collection": "items",  // 単純変数名を直接指定
      "end_label": "loop_end_1",
      "env_level": 1,
      "pos": [3, 1, 100]
    },
    {
      "op": "LOOP_START_EXP",
      "variable": "emp",
      "collection_exp_index": 1,  // CEL式リストの参照
      "end_label": "loop_end_2",
      "env_level": 2,
      "pos": [4, 1, 150]
    }
  ],
  "cel_expressions": [
    "departments != null && departments.size() > 0",  // 複雑なCEL式のみ
    "dept.employees"  // 複雑なCEL式のみ
  ]
}
```

この方法により、単純変数は命令に直接埋め込まれ、複雑なCEL式のみがCEL式リストに含まれます。これにより、実行時のパフォーマンスが向上し、不要なCEL評価のオーバーヘッドを避けることができます。

### インターフェーススキーマセクション

```json
{
  "interface_schema": {
    "name": "getUserById",
    "function_name": "getUserById",
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
    ]
  }
}
```

### レスポンスタイプセクション

```json
{
  "response_type": {
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

命令セットは、SQLテンプレートの実行可能な表現です。完全なASTではなく、固定値部分は結合されて1つの命令になります。

### 命令タイプと引数

#### 基本出力命令
- **EMIT_LITERAL**
  - `value`: 出力するSQLリテラル文字列
  - `pos`: ソースコード上の位置 [行, 列, オフセット]

- **EMIT_PARAM**
  - `param`: パラメータ変数名（単純変数）
  - `placeholder`: 開発時のダミー値
  - `pos`: ソースコード上の位置

- **EMIT_EVAL**
  - `exp_index`: CELExpressionsリスト内のインデックス
  - `placeholder`: 開発時のダミー値
  - `pos`: ソースコード上の位置

#### 制御フロー命令
- **JUMP**
  - `target`: 遷移先の命令インデックス
  - `pos`: ソースコード上の位置

- **JUMP_IF_EXP**
  - `exp_index`: CELExpressionsリスト内のインデックス
  - `target`: 条件が真の場合の遷移先インデックス
  - `pos`: ソースコード上の位置

- **JUMP_IF_PARAM**
  - `param`: パラメータ変数名（単純変数）
  - `target`: 条件が真の場合の遷移先インデックス
  - `pos`: ソースコード上の位置

- **LABEL**
  - `name`: ラベル名
  - `pos`: ソースコード上の位置

#### ループ命令
- **LOOP_START_PARAM**
  - `variable`: ループ変数名
  - `collection`: コレクション変数名（単純変数）
  - `end_label`: ループ終了ラベル
  - `env_level`: 環境レベルのインデックス
  - `pos`: ソースコード上の位置

- **LOOP_START_EXP**
  - `variable`: ループ変数名
  - `collection_exp_index`: CELExpressionsリスト内のインデックス
  - `end_label`: ループ終了ラベル
  - `env_level`: 環境レベルのインデックス
  - `pos`: ソースコード上の位置

- **LOOP_NEXT**
  - `start_label`: ループ開始ラベル
  - `pos`: ソースコード上の位置

- **LOOP_END**
  - `label`: 対応するループのラベル
  - `env_level`: 環境レベルのインデックス
  - `pos`: ソースコード上の位置

#### システムディレクティブ命令
- **EMIT_EXPLAIN**
  - `analyze`: ANALYZE句を含めるかどうか（真偽値）

- **JUMP_IF_FORCE_LIMIT**
  - `target`: 強制LIMITが設定されている場合の遷移先インデックス

- **JUMP_IF_FORCE_OFFSET**
  - `target`: 強制OFFSETが設定されている場合の遷移先インデックス

- **EMIT_SYSTEM_FIELDS**
  - `fields`: システムカラム名の配列

- **EMIT_SYSTEM_VALUES**
  - `fields`: システムカラム名の配列（EMIT_SYSTEM_FIELDSと対応）

### 命令列セクション

```json
{
  "instructions": [
    {
      "op": "EMIT_LITERAL",
      "pos": [1, 1, 0],
      "value": "SELECT id, name FROM users WHERE active = "
    },
    {
      "op": "EMIT_PARAM",
      "pos": [1, 43, 42],
      "param": "active",
      "placeholder": "true"
    },
    {
      "op": "JUMP_IF_EXP",
      "pos": [2, 1, 50],
      "exp_index": 0,
      "target": 5
    },
    {
      "op": "EMIT_LITERAL",
      "pos": [2, 24, 73],
      "value": ", email"
    },
    {
      "op": "JUMP",
      "pos": [4, 3, 100],
      "target": 6
    },
    {
      "op": "EMIT_LITERAL",
      "pos": [5, 1, 110],
      "value": ""
    },
    {
      "op": "EMIT_LITERAL",
      "pos": [6, 1, 120],
      "value": " FROM users"
    }
  ]
}
```

### CEL式セクション

```json
{
  "cel_expressions": [
    "filters.department != null && filters.department.size() > 0"
  ]
}
```

### 環境変数セクション

```json
{
  "envs": [
    [  // レベル0（最初のループ内）
      {
        "name": "department",
        "type": "string"
      }
    ],
    [  // レベル1（ネストされたループ内）
      {
        "name": "employee",
        "type": "object"
      }
    ]
  ]
}
```

この情報を使用して、CEL環境を事前に構築することができます。実行時には、適切な環境レベルで式を評価するだけで済みます。

### システムディレクティブの実装

以前の設計では、システムディレクティブは別のセクションに保持されていましたが、現在の実装では命令列の中に含まれています。これにより、以下の利点があります：

1. **一貫性**: すべての実行制御が命令列内で統一的に表現される
2. **効率性**: 実行時の追加の処理ステップが不要
3. **柔軟性**: システムディレクティブとユーザーディレクティブを同じ方法で処理できる

システムディレクティブは、クエリの実行計画（EXPLAIN）やLIMITの上書きなど、クエリ実行に関する追加の制御を提供します。これらは命令列内の特定の命令として表現されます：

```json
[
  {
    "op": "EMIT_EXPLAIN",
    "analyze": false
  },
  {
    "op": "EMIT_LITERAL",
    "value": "SELECT id, name FROM users",
    "pos": [1, 9, 8]
  },
  {
    "op": "JUMP_IF_FORCE_LIMIT",
    "target": 5
  },
  {
    "op": "EMIT_LITERAL",
    "value": " LIMIT 100",
    "pos": [1, 32, 31]
  },
  {
    "op": "JUMP",
    "target": 6,
    "pos": [1, 43, 42]
  },
  {
    "op": "EMIT_LITERAL",
    "value": "",
    "pos": [1, 43, 42]
  }
]
```

また、INSERT文やUPDATE文では、システム共通で更新日時やバージョン情報などをシステムカラムとして挿入する必要がある場合があります。これらは以下のように表現されます：

```json
[
  {
    "op": "EMIT_LITERAL",
    "value": "INSERT INTO users (name, email",
    "pos": [1, 1, 0]
  },
  {
    "op": "EMIT_SYSTEM_FIELDS",
    "fields": ["created_at", "updated_at", "version"]
  },
  {
    "op": "EMIT_LITERAL",
    "value": ") VALUES ('John', 'john@example.com'",
    "pos": [1, 28, 27]
  },
  {
    "op": "EMIT_SYSTEM_VALUES",
    "fields": ["created_at", "updated_at", "version"]
  },
  {
    "op": "EMIT_LITERAL",
    "value": ")",
    "pos": [1, 62, 61]
  }
]
```

トランザクション管理やタイムアウトなどの外部制御は、このコード生成の範囲外であり、アプリケーションコードやcontextを通じて処理されます。

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

### レスポンスaffinityの使用方法

レスポンスaffinityは、言語固有のコード生成時に以下のように使用されます：

- **one**: 単一のオブジェクトを返す関数を生成
  ```go
  func GetUserById(id int) (*User, error)
  ```

- **many**: オブジェクトのスライス/配列/リストを返す関数を生成
  ```go
  func GetUsersByDepartment(department string) ([]*User, error)
  ```

- **none**: 影響を受けた行数のみを返す関数を生成
  ```go
  func UpdateUserName(id int, name string) (int, error)
  ```

これにより、クエリの結果の形式に合わせた適切な関数シグネチャが生成され、型安全性が向上します。

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

### 型推論の例

```sql
-- テーブル情報が利用可能な場合
SELECT users.id, users.name FROM users
-- id: int, name: string

-- 関数の戻り値型推論
SELECT COUNT(*) AS count, SUM(total) AS total_sum FROM orders
-- count: int, total_sum: number

-- リテラル値の型推論
SELECT 'Constant' AS text, 42 AS num, true AS flag
-- text: string, num: int, flag: bool

-- 複雑なフィールドの型推論
SELECT data->>'name' AS json_name, CASE WHEN active THEN 'Yes' ELSE 'No' END AS status
-- json_name: any, status: any
```

### 型推論の使用方法

型推論の結果は、言語固有のコード生成時に以下のように使用されます：

- **Go**:
  ```go
  type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
  }
  ```

- **TypeScript**:
  ```typescript
  interface User {
    id: number;
    name: string;
  }
  ```

- **Python**:
  ```python
  @dataclass
  class User:
      id: int
      name: str
  ```

これにより、クエリの結果の型に合わせた適切なデータ構造が生成され、型安全性が向上します。

## JSONスキーマ定義

中間形式には検証用のJSONスキーマ定義が含まれます：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL中間形式",
  "description": "SnapSQLテンプレートの中間JSON形式",
  "type": "object",
  "required": ["instructions"],
  "properties": {
    "interface_schema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "function_name": { "type": "string" },
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
        }
      },
      "required": ["name", "parameters"]
    },
    "response_type": {
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
      },
      "required": ["name", "fields"]
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
          "op": { "type": "string" },
          "pos": {
            "type": "array",
            "items": { "type": "integer" },
            "minItems": 3,
            "maxItems": 3
          }
        },
        "required": ["op"]
      }
    },
    "cel_expressions": {
      "type": "array",
      "items": { "type": "string" }
    },
    "envs": {
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
    }
  }
}
