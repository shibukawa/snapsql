# 中間コード生成

このページでは、ASTから中間形式（IR）への変換プロセスを説明します。

## 概要

中間コード生成は以下のステップで行われます：

```
AST → 中間命令列（Instructions） → プロセッサ群による拡張 → 最終IR
```

## 中間形式（Intermediate Representation）

IR は以下の要素から構成されます：

### 1. Instructions（命令列）

SQLテンプレートの実行手順を表す命令のリストです。

```go
type Instruction struct {
    Type     InstructionType
    Value    string
    Children []Instruction
    Metadata map[string]any
}
```

主な命令タイプ：

- `LITERAL`: 固定のSQL文字列
- `PARAMETER`: パラメータ参照（`/*= param */`）
- `CONDITIONAL`: 条件分岐（`/*# if */`）
- `LOOP`: ループ（`/*# for */`）
- `EMIT_SYSTEM_VALUE`: システムカラム値の注入

### 2. Parameters（パラメータ定義）

```go
type Parameter struct {
    Name        string
    Type        string
    Required    bool
    Description string
    Default     any
}
```

### 3. Response（レスポンス定義）

```go
type Response struct {
    Type       string
    Properties map[string]Property
    IsArray    bool
}
```

### 4. Metadata（メタデータ）

```go
type Metadata struct {
    Name         string
    Description  string
    Tables       []string
    Dependencies []string
}
```

## ASTから中間命令への変換

### 基本的な変換

```go
// AST の SQL リテラルノード
ASTNode{Type: "Literal", Value: "SELECT * FROM users"}

// ↓ 変換

// IR の命令
Instruction{Type: LITERAL, Value: "SELECT * FROM users"}
```

### パラメータ参照の変換

```go
// AST のパラメータノード
ASTNode{Type: "Parameter", Value: "user_id"}

// ↓ 変換

// IR の命令
Instruction{
    Type: PARAMETER,
    Value: "user_id",
    Metadata: {
        "paramType": "integer",
        "placeholder": "?"
    }
}
```

### 条件分岐の変換

```go
// AST の条件ノード
ASTNode{
    Type: "If",
    Condition: "age_filter",
    Children: [...]
}

// ↓ 変換

// IR の命令
Instruction{
    Type: CONDITIONAL,
    Value: "age_filter",  // CEL 式
    Children: [...]       // 条件が真の時の命令列
}
```

## プロセッサパイプライン

IR 生成後、複数のプロセッサが順次実行されます：

### 1. SystemFieldProcessor

システムカラムを検出・注入します。

- 設定からシステムフィールド定義を読み込む
- INSERT/UPDATE 文に `EMIT_SYSTEM_VALUE` 命令を挿入
- 暗黙パラメータを登録

詳細は [パーサーフロー](./parser-flow.md) を参照してください。

### 2. ResponseAffinityProcessor

レスポンス型とクエリ結果の整合性を検証します。

### 3. MetadataProcessor

テーブル依存関係やサブクエリ情報を付加します。

### 4. OptimizerProcessor

不要な命令を削除し、命令列を最適化します。

## JSON IR フォーマット

IRはJSON形式で出力できます：

```bash
snapsql inspect --format json query.snap.md > query.ir.json
```

出力例：

```json
{
  "name": "get_user_by_id",
  "instructions": [
    {
      "type": "LITERAL",
      "value": "SELECT id, name FROM users WHERE id = "
    },
    {
      "type": "PARAMETER",
      "value": "user_id",
      "metadata": {
        "type": "integer"
      }
    }
  ],
  "parameters": [
    {
      "name": "user_id",
      "type": "integer",
      "required": true
    }
  ],
  "response": {
    "type": "object",
    "properties": {
      "id": {"type": "integer"},
      "name": {"type": "string"}
    }
  }
}
```

## IR スキーマ

IR の JSON スキーマは `docs/intermediate-format-schema.json` で定義されています。

## パーサーとの統合

パーサーフローの全体像については [パーサーフロー](./parser-flow.md) を参照してください。

## 関連ドキュメント

- [パーサーフロー](./parser-flow.md) - パース処理の全体像
- [コード生成](./code-generation.md) - IR からコードを生成する方法
- [型推論](./type-inference.md) - パラメータと戻り値の型推論
