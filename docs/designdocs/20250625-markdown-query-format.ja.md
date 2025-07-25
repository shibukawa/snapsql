# Markdownクエリー定義フォーマット

## 概要

SnapSQLはMarkdownベースのクエリー定義ファイル（`.snap.md`）を通じてリテラルプログラミングをサポートします。この形式は、SQLテンプレートと包括的なドキュメント、テストケース、メタデータを単一の読みやすいドキュメントに統合します。

## ファイル構造

### 基本構造

```markdown
---
function_name: "getUserData"
description: "Get user data"
---

# Query Name (任意の言語でタイトル)

## Description (概要)
クエリーの目的と説明

## Parameters (パラメータ) [オプション]
YAML/JSON/テキスト形式での入力パラメータ定義

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

## セクション仕様

### 必須セクション

| セクション | 代替名 | 目的 | 必須 |
|------------|--------|------|------|
| `Description` | `Overview` | クエリーの説明と目的 | はい |
| `SQL` | - | SQLテンプレート | はい |

### オプションセクション

| セクション | 代替名 | 目的 | 形式 |
|------------|--------|------|------|
| `Parameters` | `Params`, `Parameter` | 入力パラメータ定義 | YAML/JSON/テキスト |
| `Test Cases` | `Tests`, `TestCases` | テストシナリオ | YAML/JSON/CSV/XML/リスト |
| `Mock Data` | `Mocks`, `TestData`, `MockData` | テスト用モックデータ | YAML/JSON/CSV/XML/Markdownテーブル |

## フロントマター

### 基本フィールド

```yaml
---
function_name: "getUserData"  # 生成される関数名
description: "Get user data"  # クエリーの説明
---
```

### フィールド説明

- `function_name`: 生成されるコードの関数名（キャメルケース推奨）
- `description`: クエリーの簡潔な説明
- その他のカスタムフィールドも追加可能

### 例

```yaml
---
function_name: "getUserById"
description: "Retrieve user information by ID"
version: "1.0.0"
author: "Development Team"
---
```

## パラメータセクション

### YAML形式

```yaml
user_id: int
include_email: bool
filters:
  active: bool
  departments: [str]
pagination:
  limit: int
  offset: int
```

### JSON形式

```json
{
  "user_id": "int",
  "include_email": "bool",
  "filters": {
    "active": "bool",
    "departments": ["string"]
  },
  "pagination": {
    "limit": "int",
    "offset": "int"
  }
}
```

### テキスト形式

```markdown
- user_id (int): The ID of the user to query
- include_email (bool): Whether to include email in results
- status (string): Filter by user status (active, inactive, pending)
- limit (int): Maximum number of results to return
- offset (int): Number of results to skip

Additional notes:
- All parameters are optional except user_id
- Default limit is 10 if not specified
```

### 混合形式

```markdown
This query accepts the following parameters:

```yaml
user_id: int
include_email: bool
```

Additional parameter notes:
- user_id is required
- include_email defaults to false

```json
{
  "filters": {
    "status": "string",
    "department": "string"
  }
}
```
```

## テストケースセクション

### YAML形式

```yaml
parameters:
  user_id: 123
  include_email: true
expected:
  status: "success"
  count: 1
```

### CSV形式

```csv
user_id,active,expected
123,true,"user found"
456,false,"user inactive"
999,true,"user not found"
```

### リスト形式

```markdown
### Test Case 1: Basic Query
- Input: user_id = 123, include_email = true
- Expected: Returns user data with email

### Test Case 2: User Not Found
- Parameters: user_id = 999, active = true
- Expected: No results returned
```

### XML形式（DBUnit互換）

```xml
<dataset>
  <test_case>
    <parameters user_id="123" include_email="true"/>
    <expected status="success" count="1"/>
  </test_case>
</dataset>
```

## モックデータセクション

### YAML形式

```yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    active: false
orders:
  - id: 101
    user_id: 1
    total: 99.99
    status: "completed"
```

### CSV形式

```csv
# users
id,name,email,active
1,"John Doe","john@example.com",true
2,"Jane Smith","jane@example.com",false
3,"Bob Wilson","bob@example.com",true
```

### XML形式（DBUnit互換）

```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" active="true"/>
  <users id="2" name="Jane Smith" email="jane@example.com" active="false"/>
  <orders id="101" user_id="1" total="99.99" status="completed"/>
  <orders id="102" user_id="2" total="149.50" status="pending"/>
</dataset>
```

### Markdownテーブル形式

```markdown
| id | name | email | active |
|----|------|-------|--------|
| 1  | John Doe | john@example.com | true |
| 2  | Jane Smith | jane@example.com | false |
| 3  | Bob Wilson | bob@example.com | true |
```

## 完全な例

```markdown
---
function_name: "searchUsers"
description: "Search for active users with filtering and pagination"
version: "1.2.0"
---

# User Search Query (ユーザー検索クエリー)

## Description (概要)

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

### Test Case 1: Basic Search (基本検索)

```yaml
parameters:
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
expected:
  count: 2
  status: "success"
```

### Test Case 2: CSV Format Test

```csv
user_id,active,include_email,expected
123,true,false,"user found"
456,false,true,"user inactive"
999,true,false,"user not found"
```

### Test Case 3: List Format Test

- Input: user_id = 789, include_email = true, active = true
- Expected: Returns user data with email included

## Mock Data (モックデータ)

### YAML Format

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

### CSV Format

```csv
# users
id,name,email,department,active,created_at
1,"John Doe","john@example.com","engineering",true,"2024-01-15T10:30:00Z"
2,"Jane Smith","jane@example.com","design",true,"2024-02-20T14:45:00Z"
3,"Bob Wilson","bob@example.com","marketing",false,"2024-03-10T09:15:00Z"
```

### Markdown Table Format

| id | name | email | department | active | created_at |
|----|------|-------|------------|--------|------------|
| 1  | John Doe | john@example.com | engineering | true | 2024-01-15T10:30:00Z |
| 2  | Jane Smith | jane@example.com | design | true | 2024-02-20T14:45:00Z |
| 3  | Bob Wilson | bob@example.com | marketing | false | 2024-03-10T09:15:00Z |

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

## 実装詳細

### AST活用型解析

パーサーはgoldmarkのASTを直接活用して以下を実現：

1. **堅牢な解析**: コードブロック内のMarkdown構文に影響されない
2. **正確な行番号**: SQLコードブロックの正確な行番号情報を取得
3. **構造化データ**: ASTノードから直接データを抽出

### 多形式対応

各セクションで複数の形式をサポート：

- **YAML**: 構造化データに最適
- **JSON**: API仕様との互換性
- **CSV**: 表形式データに最適
- **XML**: DBUnit互換形式
- **Markdownテーブル**: 視覚的に分かりやすい
- **テキスト**: 自由形式の説明

### エラーハンドリング

- 必須セクション（Description/Overview、SQL）の検証
- 不正なfront matterの検出
- 構文エラーの詳細なレポート

## ファイル構成

### ディレクトリ構造

```
queries/
├── users/
│   ├── search.snap.md          # ユーザー検索
│   ├── create.snap.md          # ユーザー作成
│   └── update.snap.md          # ユーザー更新
├── posts/
│   ├── list.snap.md            # 投稿一覧
│   ├── detail.snap.md          # 投稿詳細
│   └── search.snap.md          # 投稿検索
└── analytics/
    ├── user-stats.snap.md      # ユーザー統計
    └── daily-report.snap.md    # 日次レポート
```

### 命名規則

- ファイル名にはケバブケースを使用
- 拡張子は`.snap.md`
- 関連するクエリーをサブディレクトリにグループ化
- クエリーの目的を反映する説明的な名前を使用

## 処理ルール

### セクション認識

1. パーサーは見出し内の英語キーワードを探す
2. 大文字小文字を区別しない
3. 複数の代替名をサポート（例：`Parameters`, `Params`, `Parameter`）

### コンテンツ処理

- **Parameters**: YAML/JSON/テキスト形式として解析し、`ParameterBlock`フィールドに格納
- **SQL**: SnapSQLテンプレートとして処理し、行番号情報も取得
- **Test Cases**: 複数形式（YAML/JSON/CSV/XML/リスト）をサポート
- **Mock Data**: 複数形式（YAML/JSON/CSV/XML/Markdownテーブル）をサポート

### 検証ルール

1. 必須セクション（Description/Overview、SQL）が存在する必要がある
2. Front matterは有効なYAMLである必要がある
3. SQLセクションは有効なSnapSQL構文である必要がある
4. 各形式のデータは適切な構文に従う必要がある

## SnapSQL CLIとの統合

### ファイル発見

```bash
# すべての.snap.mdファイルを処理
snapsql generate -i ./queries

# 特定のファイルを処理
snapsql generate queries/users/search.snap.md
```

### 出力生成

各`.snap.md`ファイルは以下を生成します：
- 解析されたコンテンツを含む中間JSON
- 言語固有のコード（要求された場合）
- テストケースとモックデータを含むテストファイル

### 検証

```bash
# markdownクエリーファイルを検証
snapsql validate queries/users/search.snap.md

# すべてのmarkdownファイルを検証
snapsql validate -i ./queries --format json
```

## 利点

1. **リテラルプログラミング**: コードとドキュメントの統合
2. **多形式対応**: YAML、JSON、CSV、XML、Markdownテーブルなど複数形式をサポート
3. **AST活用**: goldmarkのASTを直接活用した堅牢な解析
4. **テスト可能性**: 包括的なテストケースとモックデータのサポート
5. **保守性**: バージョン履歴と変更追跡
6. **IDE サポート**: Markdown構文ハイライトとプレビュー
7. **コラボレーション**: レビューとコメントが容易
8. **型安全性**: 構造化されたパラメータ定義
9. **国際化対応**: 複数言語でのドキュメント作成をサポート
