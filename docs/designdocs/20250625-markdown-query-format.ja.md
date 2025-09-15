# Markdownクエリー定義フォーマット

## 概要

SnapSQLはMarkdownベースのクエリー定義ファイル（`.snap.md`）を通じてリテラルプログラミングをサポートします。この形式は、SQLテンプレートと包括的なドキュメント、テストケース、メタデータを単一の読みやすいドキュメントに統合します。

## セクション構成（実装準拠）

| セクション/項目 | 必須 | 許容フォーマット | 代替手段 | 個数 |
|----------------|------|------------------|-----------|-------|
| フロントマター | × | YAML | 関数名 + Descriptionセクション | 0-1 |
| 関数名 | ○(自動) | 自動生成（明示指定も可） | `function_name`（明示指定）/ファイル名から自動 | 1 |
| Description | ○ | テキスト、Markdown（H2見出し） | Overview | 1 |
| Parameters | × | フェンスコード: YAML/JSON、（説明用のリストも可） | - | 0-1 |
| SQL | ○ | フェンスコード: `sql` 指定必須 | - | 1 |
| Test Cases | × | 見出し(H3+) + 強調行で小セクション区切り | - | 0-n |
| - Fixtures | × | YAML/JSON（複数テーブル）/CSV（単一テーブル）/DBUnit XML | - | 各テストケース0-複数 |
| - Parameters | ○ | YAML/JSON（フェンスコード） | - | 各テストケース1 |
| - Expected Results | ○ | YAML/JSON（配列） | - | 各テストケース1 |
| - Verify Query | × | フェンスコード: `sql` | - | 各テストケース0-1 |

## 関数名の自動決定（詳細）

関数名は次の優先順位で決定されます。

1. フロントマター `function_name` が存在すればそれを使用
2. それ以外は「ファイル名（拡張子を除く）」をそのまま使用

正規化・変換ルール（実装準拠）
- 拡張子の扱い: 末尾が `.snap.md` の場合はこれを取り除く。そうでなければ最終拡張子（例: `.md`）のみを取り除く。
- 文字種変換: 大文字/小文字・記号・空白などは一切変更しない（そのまま使用）。
- 推奨: 実運用では `snake_case` を推奨するが、強制ではない。
- 注意: 名前衝突時の扱いは未定義（同一パッケージ内で重複させないこと）。


## セクション詳細

### フロントマター（オプション）

YAMLフォーマットのメタデータ。関数名とDescriptionセクションで代替可能。

```yaml
---
function_name: "get_user_data"  # 明示的な関数名指定（省略可）
description: "Get user data"    # 説明（省略可）
dialect: postgres                # 方言のヒント（省略可）
---
```

### 関数名（自動）

関数名は以下の優先順位で決定します（明示指定がない場合は自動生成されます）。

1. フロントマター `function_name` があればそれを使用
2. ファイル名（拡張子を除く）をそのまま使用（例: `get_users.snap.md` → `get_users`）

注: H1タイトルはドキュメントの見出しとして利用し、関数名の自動生成には使用しません。

### Description（必須）

クエリの目的と説明。`Overview` というH2見出しでも可（H1はタイトル）。

```markdown
## Description

このクエリは指定されたユーザーIDに基づいてユーザーデータを取得します。
メールアドレスの取得はオプションで制御可能です。
```

### Parameters（オプション）

入力パラメータの定義。フェンスコードの言語により解釈を切り替えます。パーサは生テキストも保持します（`ParametersText`/`ParametersType`）。

完全に省略可能です。セクションを設けない場合、パラメータ定義なしとして扱います（空のフェンスブロックは非推奨）。

**YAML形式（推奨）:**
```yaml
user_id: int
include_email: bool
filters:
  active: bool
  departments: [string]
pagination:
  limit: int
  offset: int
```

**JSON形式:**
```json
{
  "user_id": "int",
  "include_email": "bool",
  "filters": {
    "active": "bool",
    "departments": ["string"]
  }
}
```

（参考）箇条書き（リスト）も記述できますが、この場合は型定義としては解釈されません（説明用途）。

### SQL（必須）

SnapSQL形式のSQLテンプレート。言語指定が `sql` のフェンスコードである必要があります（開始行番号も保持）。

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

### Test Cases（オプション）

テストケースの定義。H3（`###`）以降の見出しをテストケース名とし、直下の段落にある「強調（イタリック/ボールド）のラベル」で小セクションを切り替えます。

小セクションのラベル（コロン区切り・大小無視）
- Parameters: `Parameters:` / `Params:` / `Input Parameters:`
- Expected Results: `Expected:` / `Expected Results:` / `Expected Result:` / `Results:`
- Verify Query: `Verify Query:` / `Verification Query:`（任意）
- Fixtures: `Fixtures:`（CSVは `Fixtures: <table>[strategy]` の形式でテーブル名/戦略を指定）

厳密仕様（実装準拠）
- 各テストケースは次を満たす必要がある:
  - Parameters: 必須・1回のみ（重複するとエラー）
  - Expected Results: 必須・1回のみ（重複するとエラー）
  - Fixtures: 任意・複数可（YAML/JSON/CSV/XML を混在可能）
  - Verify Query: 任意・0または1回
- ラベルは段落内の強調（イタリック/ボールド）で記述する（見出しではない）。
- フェンスコードの直前直後にテキストがある場合も可だが、ラベル→フェンスコードの並びを保つこと。
- エラー例:
  - `Parameters` や `Expected` が複数回出現
  - `SQL` セクションや `Description` セクションの欠落
  - フェンス言語不一致（例: SQLが `sql` 指定でない）

## Fixtures の戦略とフォーマット

サポートされる挿入戦略（`[strategy]`）
- `clear-insert`（既定）: テーブルを空にしてから挿入
- `insert`: 既存データを残したまま挿入のみ
- `upsert`: 既存行があれば更新、なければ挿入
- `delete`: 指定データに一致する行を削除（主キー等に基づく）

フォーマット別の扱い
- YAML/JSON（複数テーブル）:
  - 例のようにテーブル名をキーにしたマップ形式を想定。
  - ラベルで `Fixtures: <table>[strategy]` と書いた場合、そのテーブルに対して戦略を適用して挿入。
  - ラベルで戦略を指定しない場合は `clear-insert` を使用。
- CSV（単一テーブル）:
  - ラベルで `Fixtures: <table>[strategy]` を必須（テーブル名と戦略を指定）。
  - フェンスコードはヘッダ行＋データ行で構成。
- DBUnit XML:
  - `<dataset>` 直下にテーブル名タグで行を列挙。

注意
- 複数の Fixtures セクションを組み合わせた場合、同一テーブルへのデータは結合される（先に指定したものから順次追加）。
- 戦略はテーブル単位で解釈される。混在する場合は、各セクションの指定が優先される。

#### Fixtures 形式例

**YAML/JSON（複数テーブル）:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
```

**CSV（単一テーブル、戦略付き）:**

強調ラベル（例）:

```
**Fixtures: users[insert]**
```

続くフェンスコード:

```csv
id,name,email,department_id
1,John Doe,john@example.com,1
2,Jane Smith,jane@example.com,2
```
戦略: `clear-insert`（既定）/`insert`/`upsert`/`delete` をサポート。

**DBUnit XML形式:**
```xml
<dataset>
  <users id="1" name="John Doe" email="john@example.com" department_id="1"/>
  <users id="2" name="Jane Smith" email="jane@example.com" department_id="2"/>
  <departments id="1" name="Engineering"/>
  <departments id="2" name="Design"/>
</dataset>
```

#### Parameters 形式例（テストケース内）

**YAML/JSON（推奨）:**
```yaml
user_id: 1
include_email: true
```

（JSONも可。箇条書きは不可）

#### Expected Results 形式例

YAML/JSON の配列をサポート。

**YAML:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

（CSV/DBUnit XMLは対象外）

## 完全な例

````markdown
---
function_name: "get_user_data"
description: "Get user data"
dialect: postgres
---

# Get User Data Query

## Description

ユーザーIDに基づいてユーザーデータを取得します。
メールアドレスの取得はオプションで制御可能です。

## Parameters

```yaml
user_id: int
include_email: bool
```

## SQL

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

## Test Cases

### Test: Basic user data

**Fixtures:**
```yaml
users:
  - {id: 1, name: "John Doe", email: "john@example.com", department_id: 1}
  - {id: 2, name: "Jane Smith", email: "jane@example.com", department_id: 2}
departments:
  - {id: 1, name: "Engineering"}
  - {id: 2, name: "Design"}
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
  departments__id: 1
  departments__name: "Engineering"
```

### Test: Without email

**Parameters:**
```yaml
user_id: 2
include_email: false
```

**Expected Results:**
```yaml
- id: 2
  name: "Jane Smith"
  departments__id: 2
  departments__name: "Design"
```
````
