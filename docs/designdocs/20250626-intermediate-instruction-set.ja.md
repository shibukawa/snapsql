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
    "format_version": "1"
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
      "pos": [1, 1, 0],
      "content": "SELECT id, name"
    },
    {
      "type": "jump_if_false",
      "pos": [1, 18, 17],
      "condition": "include_email",
      "target": 3
    },
    {
      "type": "static",
      "pos": [1, 42, 41],
      "content": ", email"
    },
    {
      "type": "static",
      "pos": [1, 65, 64],
      "content": " FROM users_"
    },
    {
      "type": "variable",
      "pos": [1, 76, 75],
      "name": "env",
      "validation": {
        "type": "string",
        "pattern": "^[a-z]+$"
      }
    },
    {
      "type": "jump_if_false",
      "pos": [1, 85, 84],
      "condition": "filters.active != null",
      "target": 8
    },
    {
      "type": "static",
      "pos": [1, 110, 109],
      "content": " WHERE active = "
    },
    {
      "type": "variable",
      "pos": [1, 125, 124],
      "name": "filters.active"
    }
  ]
}
```

### メタデータセクション

* `format_version`: フォーマットバージョン。現在は1のみ。

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
     "pos": [1, 1, 0],
     "content": "SELECT * FROM"
   }
   ```

2. `variable`: 変数の展開
   ```json
   {
     "type": "variable",
     "pos": [1, 15, 14],
     "name": "table_name",
     "validation": {
       "type": "string",
       "pattern": "^[a-z_]+$"
    }
   }
   ```

3. `jump_if_false`: 条件が偽の場合に指定されたインデックスにジャンプ
   ```json
   {
     "type": "jump_if_false",
     "pos": [1, 25, 24],
     "condition": "include_email",
     "target": 5
   }
   ```

4. `loop_start`: ループの開始
   ```json
   {
     "type": "loop_start",
     "pos": [1, 30, 29],
     "variable": "item",
     "collection": "sort_fields",
     "end_target": 10
   }
   ```

5. `loop_end`: ループの終了（ループ開始に戻る）
   ```json
   {
     "type": "loop_end",
     "pos": [1, 100, 99],
     "start_target": 4
   }
   ```

6. `array_expansion`: 配列の展開（IN句用）
   ```json
   {
     "type": "array_expansion",
     "pos": [1, 50, 49],
     "source": "departments",
     "separator": ", ",
     "quote": true
   }
   ```

7. `emit_if_not_boundary`: 境界でない場合のみ出力（カンマ、AND、ORなど）
   ```json
   {
     "type": "emit_if_not_boundary",
     "pos": [1, 60, 59],
     "content": ", "
   }
   ```

8. `emit_static_boundary`: 境界を示す静的テキスト（閉じかっこ、clauseの境界など）
   ```json
   {
     "type": "emit_static_boundary",
     "pos": [1, 70, 69],
     "content": ") FROM"
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
