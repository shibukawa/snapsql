# SnapSQL 中間形式仕様

**ドキュメントバージョン:** 1.0  
**日付:** 2025-06-25  
**ステータス:** 設計フェーズ

## 概要

このドキュメントは、SnapSQLテンプレートの中間JSON形式を定義します。中間形式は、SQLテンプレートパーサーとコードジェネレータ間の橋渡しとして機能し、解析されたSQLテンプレートのメタデータ、AST、インターフェーススキーマの言語非依存表現を提供します。

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

### 2. 完全な情報保持
- 位置情報を含む完全なAST表現
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
  "source": {
    "file": "queries/users.snap.sql",
    "content": "SELECT id, name FROM users WHERE active = /*= active */true"
  },
  "interface_schema": { /* InterfaceSchemaオブジェクト */ },
  "ast": { /* ASTルートノード */ }
}
```

### インターフェーススキーマセクション

```json
{
  "interface_schema": {
    "name": "UserQuery",
    "description": "オプションフィルタリング付きユーザークエリ",
    "function_name": "queryUsers",
    "parameters": [
      {
        "name": "active",
        "type": "bool"
      },
      {
        "name": "filters",
        "type": "object",
        "children": [
          {
            "name": "department",
            "type": "array",
            "children": [
              {
                "name": "o1",
                "type": "string"
              }
            ]
          },
          {
            "name": "role",
            "type": "string"
          }
        ]
      }
    ]
  }
}
```

### ASTセクション

```json
{
  "ast": {
    "type": "SELECT_STATEMENT",
    "pos": [1, 1, 0],
    "select_clause": {
      "type": "SELECT_CLAUSE",
      "pos": [1, 1, 0],
      "fields": [
        {
          "type": "IDENTIFIER",
          "pos": [1, 8, 7],
          "name": "id"
        },
        {
          "type": "IDENTIFIER", 
          "pos": [1, 12, 11],
          "name": "name"
        }
      ]
    },
    "from_clause": {
      "type": "FROM_CLAUSE",
      "pos": [1, 17, 16],
      "tables": [
        {
          "type": "IDENTIFIER",
          "pos": [1, 22, 21],
          "name": "users"
        }
      ]
    },
    "where_clause": {
      "type": "WHERE_CLAUSE",
      "pos": [1, 28, 27],
      "condition": {
        "type": "EXPRESSION",
        "pos": [1, 34, 33],
        "left": {
          "type": "IDENTIFIER",
          "name": "active"
        },
        "operator": "=",
        "right": {
          "type": "VARIABLE_SUBSTITUTION",
          "pos": [1, 43, 42],
          "variable_name": "active",
          "dummy_value": "true",
          "variable_type": "bool"
        }
      }
    },
    "implicit_if_block": {
      "type": "IMPLICIT_CONDITIONAL",
      "pos": [-1, -1, -1],
      "condition": "active != null",
      "target_clause": "WHERE_CLAUSE"
    }
  }
}
```

#### 位置情報
- **通常ノード**: `pos: [line, column, offset]` ソースコードからの位置
- **暗黙ノード**: `pos: [-1, -1, -1]` 自動挿入された要素

## JSONスキーマ定義

中間形式には検証用のJSONスキーマ定義が含まれます：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL中間形式",
  "description": "SnapSQLテンプレートの中間JSON形式",
  "type": "object",
  "required": ["source", "ast"],
  "properties": {
    "source": {
      "type": "object",
      "required": ["file", "content"],
      "properties": {
        "file": { "type": "string" },
        "content": { "type": "string" }
      }
    },
    "interface_schema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "description": { "type": "string" },
        "function_name": { "type": "string" },
        "parameters": { "type": "array" }
      }
    },
    "ast": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": { "type": "string" },
        "pos": {
          "type": "array",
          "items": { "type": "integer" },
          "minItems": 3,
          "maxItems": 3
        }
      }
    }
  }
}
```

## 実装計画

### フェーズ1: コア構造
1. 中間形式データ構造の定義
2. ASTノードのJSONシリアライゼーション実装
3. パーサー出力から中間形式への基本変換作成

### フェーズ2: スキーマ統合
1. 中間形式にInterfaceSchemaを含める
2. 再帰的パラメータ構造の追加
3. パラメータ型分析の実装

### フェーズ3: AST拡張
1. 暗黙ノード位置処理の追加（pos: [-1, -1, -1]）
2. 完全なノード情報を含むASTシリアライゼーションの拡張
3. ASTノードでの変数置換詳細の実装

### フェーズ4: 検証
1. JSONスキーマ検証の実装
2. 形式検証ユーティリティの追加
3. 検証エラー報告の作成

### フェーズ5: CLI統合
1. `rawparse`サブコマンドの作成
2. 出力形式オプションの追加
3. ファイル処理パイプラインの実装

## 使用例

### コマンドライン使用法
```bash
# 単一ファイルを中間形式に解析
snapsql rawparse queries/users.snap.sql

# 整形出力付きで解析
snapsql rawparse --pretty queries/users.snap.sql

# 複数ファイルの解析
snapsql rawparse queries/*.snap.sql --output-dir generated/

# スキーマに対する検証
snapsql rawparse --validate queries/users.snap.sql
```

### プログラム使用法
```go
package main

import (
    "github.com/shibukawa/snapsql/intermediate"
    "github.com/shibukawa/snapsql/parser"
)

func main() {
    // SQLテンプレートの解析
    ast, schema, err := parser.ParseTemplate(sqlContent)
    if err != nil {
        panic(err)
    }
    
    // 中間形式への変換
    format := intermediate.NewFormat()
    format.SetSource("queries/users.snap.sql", sqlContent)
    format.SetAST(ast)
    format.SetInterfaceSchema(schema)
    
    // JSONへのシリアライゼーション
    jsonData, err := format.ToJSON()
    if err != nil {
        panic(err)
    }
    
    // スキーマに対する検証
    if err := format.Validate(); err != nil {
        panic(err)
    }
}
```

## 利点

### コードジェネレータ向け
- 全ターゲット言語で一貫した入力形式
- コード生成のための豊富な型情報
- エラー報告のための位置情報
- 分析のための完全なテンプレート構造

### ツール向け
- 言語非依存のテンプレート分析
- IDE統合の可能性
- テンプレート検証とリンティング
- ドキュメント生成

### デバッグ向け
- 完全な解析ツリーの可視化
- 変数参照の追跡
- 制御フロー分析
- テンプレート複雑度メトリクス

## 将来の拡張

### テンプレート分析
- デッドコード検出
- 変数使用分析
- パフォーマンス推定
- セキュリティ分析

### 多言語サポート
- 言語固有メタデータセクション
- カスタム型マッピング
- フレームワーク固有アノテーション
- コードスタイル設定

### 最適化
- テンプレート簡素化
- クエリ最適化ヒント
- キャッシュ戦略
- パフォーマンスメトリクス

## 結論

中間形式は、SnapSQLのコード生成パイプラインの堅牢な基盤を提供します。解析とコード生成を分離することで、以下を実現します：

1. **柔軟性**: 複数のターゲット言語とフレームワークのサポート
2. **保守性**: 関心の明確な分離
3. **拡張性**: 新機能とジェネレータの簡単な追加
4. **ツール**: 分析と開発ツールの豊富なエコシステム

この形式は、SnapSQLの多言語コード生成機能の礎石となります。
