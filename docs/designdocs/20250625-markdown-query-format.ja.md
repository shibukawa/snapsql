# Markdownクエリー定義フォーマット

## 概要

SnapSQLはMarkdownベースのクエリー定義ファイル（`.snap.md`）を通じてリテラルプログラミングをサポートします。この形式は、SQLテンプレートと包括的なドキュメント、テストケース、メタデータを単一の読みやすいドキュメントに統合します。

## ファイル構造

### 基本構造

```markdown
# Query Name (任意の言語でタイトル)

## Overview (概要)
クエリーの目的と説明

## Parameters (パラメータ)
YAML形式での入力パラメータ定義

## SQL (SQL)
SnapSQL形式のSQLテンプレート

## Test Cases (テストケース) [オプション]
パラメータセットと期待結果

## Mock Data (モックデータ) [オプション]
テスト用のモックデータ定義
```

### 拡張セクション（オプション）

```markdown
## Performance (パフォーマンス) [オプション]
パフォーマンスの考慮事項と要件

## Security (セキュリティ) [オプション]
セキュリティの考慮事項とアクセス制御

## Change Log (変更履歴) [オプション]
バージョン履歴と変更内容
```

## 国際化対応

### 見出し形式

すべてのセクション見出しは英語キーワードの後にオプションのローカライズされたタイトルを使用します：

```markdown
## English Keyword (ローカライズされたタイトル)
```

例：
- `## Overview (概要)` - 日本語
- `## Parameters (Parámetros)` - スペイン語  
- `## SQL (SQL)` - 共通
- `## Test Cases (Cas de Test)` - フランス語
- `## Mock Data (Dados Simulados)` - ポルトガル語

### サポートされる見出し

| 英語キーワード | 目的 | 必須 |
|----------------|------|------|
| `Overview` | クエリーの説明と目的 | はい |
| `Parameters` | 入力パラメータ定義 | はい |
| `SQL` | SQLテンプレート | はい |
| `Test Cases` | テストシナリオ | はい |
| `Mock Data` | テスト用モックデータ | いいえ |
| `Performance` | パフォーマンス情報 | いいえ |
| `Security` | セキュリティ考慮事項 | いいえ |
| `Change Log` | バージョン履歴 | いいえ |

## 完全な例

```markdown
---
version: v1.0
function_name: "user-search"
---

# User Search Query (ユーザー検索クエリー)

## Overview (概要)

Searches for active users based on various criteria with pagination support.
Supports department filtering and sorting functionality.

アクティブなユーザーを条件に基づいて検索し、ページネーション機能を提供します。
部署フィルタリングとソート機能をサポートしています。

## Parameters (パラメータ)

```yaml
user_id: int
filters:
  active: bool
  departments: [str]
  name_pattern: str
pagination:
  limit: int
  offset: int
sort_by: str
include_email: bool
table_suffix: str
```

## SQL (SQL)

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    department,
    created_at
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
    /*# if filters.name_pattern */
    AND name ILIKE /*= filters.name_pattern */'%john%'
    /*# end */
    /*# if filters.departments */
    AND department IN (/*= filters.departments */'engineering', 'design')
    /*# end */
/*# if sort_by */
ORDER BY /*= sort_by */created_at DESC
/*# end */
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

## Test Cases (テストケース)

テストケースは[dbtestify](https://github.com/shibukawa/dbtestify)形式を使用し、YAMLベースのフィクスチャ、パラメータ、期待結果を含みます。

### Case 1: Basic Search (基本検索)

**Fixture (初期データ):**
```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    department: "marketing"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```

**Parameters (パラメータ):**
```yaml
user_id: 123
filters:
  active: true
  departments: ["engineering", "design"]
  name_pattern: null
pagination:
  limit: 20
  offset: 0
sort_by: "name"
include_email: false
table_suffix: "test"
```

**Expected Result (期待結果):**
```yaml
- id: 1
  name: "John Doe"
  department: "engineering"
  created_at: "2024-01-15T10:30:00Z"
- id: 2
  name: "Jane Smith"
  department: "design"
  created_at: "2024-02-20T14:45:00Z"
```

### Case 2: Full Options with Email (全オプション有効)

**Fixture (初期データ):**
```yaml
users:
  - id: 4
    name: "Alice Smith"
    email: "alice@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-10T08:00:00Z"
  - id: 5
    name: "Charlie Smith"
    email: "charlie@example.com"
    department: "marketing"
    active: true
    created_at: "2024-01-20T09:00:00Z"
```

**Parameters (パラメータ):**
```yaml
user_id: 456
filters:
  active: true
  departments: ["marketing"]
  name_pattern: "%smith%"
pagination:
  limit: 5
  offset: 0
sort_by: "created_at DESC"
include_email: true
table_suffix: "test"
```

**Expected Result (期待結果):**
```yaml
- id: 5
  name: "Charlie Smith"
  email: "charlie@example.com"
  department: "marketing"
  created_at: "2024-01-20T09:00:00Z"
- id: 4
  name: "Alice Smith"
  email: "alice@example.com"
  department: "marketing"
  created_at: "2024-01-10T08:00:00Z"
```

### Case 3: Empty Result (空の結果)

**Fixture (初期データ):**
```yaml
users:
  - id: 6
    name: "David Wilson"
    email: "david@example.com"
    department: "hr"
    active: false
    created_at: "2024-01-05T10:00:00Z"
```

**Parameters (パラメータ):**
```yaml
user_id: 789
filters:
  active: true
  departments: ["engineering"]
  name_pattern: null
pagination:
  limit: 10
  offset: 0
sort_by: null
include_email: false
table_suffix: "test"
```

**Expected Result (期待結果):**
```yaml
[]
```

## Mock Response

```yaml
default:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    department: "engineering"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    department: "design"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    department: "marketing"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```

## Performance (パフォーマンス)

### Index Requirements (インデックス要件)
- `users.active` - Required (必須)
- `users.department` - Recommended (推奨)
- `users.name` - For LIKE searches (LIKE検索用)

### Estimated Execution Time (推定実行時間)
- Small scale (< 10K rows): < 10ms
- Medium scale (< 100K rows): < 50ms  
- Large scale (> 100K rows): < 200ms

## Security (セキュリティ)

### Access Control (アクセス制御)
- Administrators: Can search all departments
- Regular users: Can only search their own department

管理者は全部署のユーザーを検索可能、一般ユーザーは自分の部署のみ検索可能

### Data Masking (データマスキング)
- `email` field is masked based on permissions
- `email`フィールドは権限に応じてマスキング

## Change Log (変更履歴)

### v1.2.0 (2024-03-01)
- Added `name_pattern` parameter
- Support for ILIKE search
- `name_pattern`パラメータを追加
- ILIKE検索をサポート

### v1.1.0 (2024-02-15)
- Added `include_email` option
- Performance optimization
- `include_email`オプションを追加
- パフォーマンス最適化

### v1.0.0 (2024-01-01)
- Initial version
- 初期バージョン
```

## ファイル構成

### ディレクトリ構造

```
queries/
├── users/
│   ├── search.md          # ユーザー検索
│   ├── create.md          # ユーザー作成
│   └── update.md          # ユーザー更新
├── posts/
│   ├── list.md            # 投稿一覧
│   ├── detail.md          # 投稿詳細
│   └── search.md          # 投稿検索
└── analytics/
    ├── user-stats.md      # ユーザー統計
    └── daily-report.md    # 日次レポート
```

### 命名規則

- ファイル名にはケバブケースを使用
- 関連するクエリーをサブディレクトリにグループ化
- クエリーの目的を反映する説明的な名前を使用

## フロントマター

### 必須フィールド

```yaml
---
name: "user search"
dialect: "postgres"
---
```

### フィールド説明

- `name`: 生成されるコードの関数名（スペース区切りの単語、各言語の適切な命名規則に変換される）
- `dialect`: SQLダイアレクト（`postgres`, `mysql`, `sqlite`）

### 例

```yaml
---
name: "get user by id"
dialect: "postgres"
---
```

```yaml
---
name: "list active posts"
dialect: "mysql"
---
```

```yaml
---
name: "analytics daily report"
dialect: "sqlite"
---
```

## 処理ルール

### 見出し認識

1. パーサーは見出し内の英語キーワードを探す
2. 括弧内のローカライズされたタイトルは処理中に無視される
3. 各見出し下のコンテンツはその種類に応じて処理される

### コンテンツ処理

- **Parameters**: YAMLとして解析
- **SQL**: SnapSQLテンプレートとして処理
- **Test Cases**: YAMLフィクスチャ、パラメータ、期待結果を含むdbtestify形式として解析
- **Mock Data**: 開発とテスト用のYAMLとして解析

### 検証ルール

1. 必須セクションが存在する必要がある
2. Parametersセクションは有効なYAMLである必要がある
3. SQLセクションは有効なSnapSQL構文である必要がある
4. テストケースは有効なYAMLフィクスチャ、パラメータ、期待結果を含むdbtestify形式に従う必要がある
5. テストケースのパラメータはParametersセクションの構造と一致する必要がある

## SnapSQL CLIとの統合

### ファイル発見

```bash
# すべての.snap.mdファイルを処理
snapsql generate -i ./queries

# 特定のファイルを処理
snapsql generate queries/users/search.md
```

### 出力生成

各`.snap.md`ファイルは以下を生成します：
- 解析されたコンテンツを含む中間JSON
- 言語固有のコード（要求された場合）
- フィクスチャと期待結果を含むdbtestify互換のテストファイル

### 検証

```bash
# markdownクエリーファイルを検証
snapsql validate queries/users/search.md

# すべてのmarkdownファイルを検証
snapsql validate -i ./queries --format json
```

## 利点

1. **リテラルプログラミング**: コードとドキュメントの統合
2. **国際化対応**: 複数言語のサポート
3. **テスト可能性**: データベーステスト用のdbtestify形式による統合テストケース
4. **保守性**: バージョン履歴と変更追跡
5. **IDE サポート**: Markdown構文ハイライトとプレビュー
6. **コラボレーション**: レビューとコメントが容易
7. **データベーステスト**: フィクスチャと期待結果による完全なデータベース統合テスト
