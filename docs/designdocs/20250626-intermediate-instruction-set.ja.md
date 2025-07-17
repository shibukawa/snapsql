# 中間命令セット設計

## 概要

SnapSQLは、SQLテンプレートを解析して中間命令セットを生成し、それを各言語のランタイムライブラリで実行することで動的なSQLクエリを生成します。この文書では、中間命令セットの設計について説明します。

## 設計目標

* SQLの構文解析結果をそのまま保持するのではなく、実行時の処理に最適化された形式にする
* 各言語のランタイムライブラリの実装を容易にする
* 実行時のパフォーマンスを最適化する
* セキュリティを確保する（SQLインジェクションの防止）

## 中間命令セット形式

中間命令セットは以下のJSONフォーマットで保存されます：

```json
{
  "metadata": {
    "source_file": "queries/users.snap.sql",
    "hash": "sha256:...",
    "timestamp": "2025-06-26T10:00:00Z"
  },
  "parameters": {
    "include_email": {"type": "boolean"},
    "env": {"type": "string", "enum": ["dev", "test", "prod"]},
    "filters": {
      "type": "object",
      "properties": {
        "active": {"type": "boolean"},
        "departments": {"type": "array", "items": {"type": "string"}}
      }
    },
    "sort_fields": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "field": {"type": "string"},
          "direction": {"type": "string", "enum": ["ASC", "DESC"]}
        }
      }
    }
  },
  "instructions": [
    {
      "type": "static",
      "content": "SELECT id, name"
    },
    {
      "type": "conditional",
      "condition": "include_email",
      "instructions": [
        {
          "type": "static",
          "content": ", email"
        }
      ]
    },
    {
      "type": "static",
      "content": " FROM users_"
    },
    {
      "type": "variable",
      "name": "env",
      "validation": {
        "type": "string",
        "pattern": "^[a-z]+$"
      }
    },
    {
      "type": "conditional",
      "condition": "filters.active != null",
      "instructions": [
        {
          "type": "static",
          "content": " WHERE active = "
        },
        {
          "type": "variable",
          "name": "filters.active"
        }
      ]
    }
  ]
}
```

### メタデータセクション

* `source_file`: 元のSQLテンプレートファイルのパス
* `hash`: ソースファイルのハッシュ値（変更検知用）
* `timestamp`: 生成時のタイムスタンプ

### パラメータセクション

* JSON Schemaフォーマットでパラメータの型情報を定義
* 各パラメータに対する制約条件も定義可能
* ネストされたオブジェクトやリストもサポート

### 命令セット

以下の命令タイプをサポート：

1. `static`: 静的なSQL文字列
   ```json
   {
     "type": "static",
     "content": "SELECT * FROM"
   }
   ```

2. `variable`: 変数の展開
   ```json
   {
     "type": "variable",
     "name": "table_name",
     "validation": {
       "type": "string",
       "pattern": "^[a-z_]+$"
    }
   }
   ```

3. `conditional`: 条件分岐
   ```json
   {
     "type": "conditional",
     "condition": "include_email",
     "instructions": [...]
   }
   ```

4. `loop`: 繰り返し処理
   ```json
   {
     "type": "loop",
     "source": "sort_fields",
     "separator": ", ",
     "instructions": [...]
   }
   ```

5. `array_expansion`: 配列の展開（IN句用）
   ```json
   {
     "type": "array_expansion",
     "source": "departments",
     "separator": ", ",
     "quote": true
   }
   ```

### セキュリティ機能

1. パラメータバリデーション
   * 型チェック
   * パターンマッチング
   * 列挙値の制限
   * カスタムバリデーション関数

2. SQLインジェクション防止
   * テーブル名は完全な動的生成を禁止（サフィックスのみ許可）
   * 値はすべてプレースホルダー化
   * 特殊文字のエスケープ

## ランタイムライブラリの実装

各言語のランタイムライブラリは以下の機能を実装する必要があります：

1. 中間命令セットの読み込みとパース
2. パラメータのバリデーション
3. 命令の実行とSQL文字列の生成
4. プレースホルダーの管理
5. データベース固有の調整（プレースホルダーの形式など）

### エラー処理

以下のエラーを適切に処理する必要があります：

1. パラメータバリデーションエラー
2. 不正な命令セット
3. 実行時エラー（未定義変数など）
4. データベース固有のエラー

## 今後の拡張予定

1. より高度な条件分岐（else if、else）
2. カスタムバリデーション関数のサポート
3. データベース固有の最適化
4. キャッシュ機能
5. デバッグ支援機能

## 制限事項

1. 完全な動的テーブル名の生成は不可
2. サブクエリの動的生成は制限付き
3. ORDER BY、GROUP BYの列名は事前定義が必要
