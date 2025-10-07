# FunctionDefinition型システム仕様

## 概要

SnapSQLのFunctionDefinitionにおけるパラメータ型システムの仕様を定義します。パラメータ型はYAMLで柔軟に記述でき、型エイリアス・配列・ネスト・型推論・ダミー値生成・エラー処理などをサポートします。

## サポートされる型とエイリアス


### プリミティブ型・エイリアス

| 型名      | エイリアス                      | Go型         | CEL型         | 説明         |
|-----------|-------------------------------|--------------|--------------|--------------|
| `string`  | `text`, `varchar`, `str`      | string       | string        | 文字列型      |
| `int`     | `integer`, `long`             | int64        | int           | 64bit整数型   |
| `int32`   |                               | int32        | int           | 32bit整数型   |
| `int16`   | `smallint`                    | int16        | int           | 16bit整数型   |
| `int8`    | `tinyint`                     | int8         | int           | 8bit整数型    |
| `float`   | `double`                      | float64      | double        | 64bit浮動小数 |
| `decimal` | `numeric`                     | github.com/shopspring/decimal.Decimal | double        | 10進数高精度 |
| `float32` |                               | float32      | double        | 32bit浮動小数 |
| `bool`    | `boolean`                     | bool         | bool          | 真偽値型      |


### 特殊型

| 型名        | Go型           | CEL型         | 説明                         |
|-------------|----------------|--------------|------------------------------|
| `timestamp`（エイリアス: `datetime`, `date`, `time`） | time.Time | string | タイムスタンプ型（内部では統一） |
| `email`     | string         | string        | メールアドレス型             |
| `uuid`      | github.com/google/uuid.UUID | string        | UUID型         |
| `json`      | map[string]any | map(string, dyn) | JSON型                   |
| `any`       | any            | dyn           | 任意型（推論・未定義用）     |


### 配列型

- `int[]`, `string[]`, `float32[]` など、型名の末尾に`[]`を付けることで配列型を表現します。
- YAMLリスト記法 `[int]` や `- id: int` `name: string` のような配列オブジェクトもサポートします。

## 型定義の記法

### 1. シンプルな型指定

```yaml
parameters:
  id: int
  name: string
  scores: float[]
  flags: [bool]
  prices: decimal[]
```

### 2. オブジェクト・ネスト

```yaml
parameters:
  user:
    id: int
    name: string
    profile:
      email: email
      age: int
```

### 3. 配列の書き方

- プリミティブ型配列: `[int]` → `int[]`
- オブジェクト配列:

```yaml
parameters:
  users:
    - id: int
      name: string
  tags: [string]
```

### 4. 型推論

型が明示されていない場合、値から型を自動推論します:

```yaml
parameters:
  user_id: 123          # → int
  user_name: "John"     # → string
  is_active: true       # → bool
  price: 99.99          # → float
  items:
    - id: 1
      value: 2.0
```

## 型エイリアス・正規化

- `integer`, `long` → `int`
- `double` → `float`
- `decimal`, `numeric` → `decimal`
- `text`, `varchar`, `str` → `string`
- `boolean` → `bool`

## 型解決アルゴリズム

1. **変数名の検証**: 英数字・アンダースコアで始まる必要あり
2. **型名の正規化**: エイリアスを正規化し、配列型も`[]`で統一
3. **配列・オブジェクトの再帰的処理**: ネストや配列も再帰的に型解決
4. **型推論**: 値から型を自動推論
5. **エラー時はデフォルト型(string)にフォールバック**


## ダミーリテラル生成について

- 型ごとのダミー値は内部実装で自動生成されます（詳細は省略）。

## エラーハンドリング

- 無効な変数名はスキップされ、エラーが返る
- 型不明・未定義は`any`型・`nil`ダミー値となる
- ネスト・配列の循環参照は検出しエラー

## 使用例

```yaml
parameters:
  user_id: int
  user_name: string
  is_active: bool
  users:
    - id: int32
      name: string
  tags: [string]
  meta:
    created_at: datetime
    updated_at: datetime
```

## SQLテンプレートでの利用例

```sql
SELECT /*= user_id */, /*= user_name */ FROM users WHERE is_active = /*= is_active */
```

## 今後の拡張

- カスタム型(enum等)や制約(min/max)も将来的にサポート予定
    name:
      type: string
      description: "ユーザー名"
    profile:
      email:
        type: email
        description: "メールアドレス"
      age:
        type: int
        description: "年齢"
```

### 型推論

型が明示的に指定されていない場合、値から型を推論します：

```yaml
parameters:
  # 型推論の例
  user_id: 123          # → int
  user_name: "John"     # → string
  is_active: true       # → bool
  price: 99.99          # → float
```

## 変数参照での型解決

### ドット記法

ネストしたオブジェクトはドット記法でアクセスします：

```sql
SELECT /*= user.name */ FROM users WHERE id = /*= user.profile.user_id */
```

### 型解決のアルゴリズム

1. **変数名の分解**: `user.profile.name` → `["user", "profile", "name"]`
2. **階層的な型解決**: 各階層で型情報を取得
3. **最終型の決定**: 最後の要素の型を返す

```go
// 型解決の例
variableName := "user.profile.email"
// user → map[string]any
// user.profile → map[string]any  
// user.profile.email → type: "email" → ダミー値: "user@example.com"
```

## ダミーリテラル生成ルール

### 型からダミー値への変換

```go
func generateDummyValueFromType(paramType string) string {
    switch strings.ToLower(paramType) {
    case "int", "integer", "long":
        return "1"
    case "float", "double", "decimal", "number":
        return "1.0"
    case "bool", "boolean":
        return "true"
    case "string", "text":
        return "'dummy'"
    case "date":
        return "'2024-01-01'"
    case "datetime", "timestamp":
        return "'2024-01-01 00:00:00'"
    case "email":
        return "'user@example.com'"
    case "uuid":
        return "'00000000-0000-0000-0000-000000000000'"
    case "json":
        return "'{}'"
    case "array":
        return "'[]'"
    default:
        return "'dummy'"  // デフォルトは文字列
    }
}
```

### トークン型マッピング

ダミー値からTokenTypeへのマッピング：

```go
func inferTokenTypeFromValue(dummyValue string) tokenizer.TokenType {
    switch {
    case strings.HasPrefix(dummyValue, "'"):
        return tokenizer.STRING
    case dummyValue == "true" || dummyValue == "false":
        return tokenizer.BOOLEAN
    case strings.Contains(dummyValue, "."):
        return tokenizer.NUMBER // 浮動小数点数
    default:
        return tokenizer.NUMBER // 整数
    }
}
```

## エラーハンドリング

### 型解決エラー

1. **パラメータが見つからない**: `ErrParameterNotFound`
2. **ネストオブジェクトではない**: `ErrParameterNotNestedObject`
3. **型情報が不正**: デフォルト型（string）にフォールバック

### エラー処理の方針

```go
// エラー時はデフォルト値を使用
paramType, err := getParameterType(variableName, functionDef.Parameters)
if err != nil {
    paramType = "string"  // デフォルト型
}
```

## 使用例

### 完全なFunctionDefinition例

```yaml
# snapsql.yaml
functions:
  getUserProfile:
    description: "ユーザープロファイル取得"
    parameters:
      user_id:
        type: int
        description: "対象ユーザーID"
      include_profile:
        type: bool
        description: "プロファイル情報を含むか"
      filters:
        status:
          type: string
          description: "ステータス"
        created_after:
          type: date
          description: "作成日時フィルタ"
```

### SQLテンプレートでの使用

```sql
-- getUserProfile.snap.sql
SELECT 
    id,
    name,
    /*= filters.status */ as status,
    created_at
FROM users 
WHERE 
    id = /*= user_id */
    /*@ if include_profile */
    AND profile_id IS NOT NULL
    /*@ end */
    /*@ if filters.created_after */
    AND created_at >= /*= filters.created_after */
    /*@ end */
```

## 実装段階

### parserstep1（現在実装済み）

- DUMMY_LITERALトークンの挿入
- 変数名の抽出と保存
- 基本的な構文検証

### parserstep6（今後実装）

- FunctionDefinitionを使用した型解決
- 実際のリテラル値への置換
- CEL式の評価
- Namespace管理

## 将来の拡張

### カスタム型

```yaml
parameters:
  user_status:
    type: enum
    values: ["active", "inactive", "pending"]
    default: "active"
```

### 配列型の詳細指定

```yaml
parameters:
  user_ids:
    type: array
    element_type: int
    description: "ユーザーIDの配列"
```

### 制約条件

```yaml
parameters:
  age:
    type: int
    min: 0
    max: 150
    description: "年齢（0-150）"
```


この型システムにより、SnapSQLテンプレートでの型安全性と開発効率が向上します。
