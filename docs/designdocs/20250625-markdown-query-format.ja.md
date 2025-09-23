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

> **補足:** `function_name` はファイル名（拡張子を除く）から自動導出されます。同一値の指定のみを目的としたフロントマターは省略してください。また `response_affinity` などの追加メタデータは現在のパーサでは参照しないため記述不要です。


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

YAMLフォーマットのメタデータ。関数名とDescriptionセクションで代替可能です。`function_name` はファイル名から自動決定されるため、同一値を明示する目的だけでフロントマターを残す必要はありません。個別のダイアレクト指定など、明確なオーバーライドが必要な場合のみ利用してください。

```yaml
---
function_name: "get_user_data"  # ファイル名と異なる場合のみ指定
description: "Get user data"    # 説明（省略可）
dialect: postgres                # 方言のヒント（省略可）
---
```

> 備考: 旧仕様で使用していた `response_affinity` などのメタデータは現行パーサでは解釈しません。

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

YAML/JSON の配列、または外部ファイル参照（Markdownリンク）をサポート。

**YAML:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  departments__id: 1
  departments__name: "Engineering"
```

**外部ファイル参照（YAML/JSON）:**
```markdown
**Expected Results:**
[[期待値データ](../expected/expected_user.yaml)]
```
または
```markdown
**Expected Results:**
[[期待値データ](../expected/expected_user.json)]
```

外部ファイルの内容はYAML/JSON配列形式で記述します。

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

````markdown
## Fixtures（テーブルデータ挿入戦略・外部ファイル参照）

テストケースごとに、テーブルごとのフィクスチャ挿入戦略（truncate+insert, insert, upsert, delete）を指定できます。

### 基本構文

```markdown
### Test: ユーザー挿入テスト

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
- id: 2
  name: "Bob"
```

**Fixtures: users[upsert]**
```yaml
- id: 3
  name: "Charlie"
```

**Fixtures: users[upsert]**
```yaml
- id: 4
  name: "David"
```

**Fixtures: users[delete]**
```yaml
- id: 2
```
```

- `[clear-insert]` : テーブルをtruncate（またはdelete）してからinsert（デフォルト）
- `[upsert]` : 既存データがあればupdate、なければinsert
- `[delete]` : 指定した主キーでdelete

### 外部ファイル参照による共通データの共有

複数テストケース間で共通フィクスチャデータを使いたい場合、外部ファイル（YAML/CSV）を参照できます。

#### 記述例

```markdown
**Fixtures: users[clear-insert]**
[共通ユーザーデータ](../fixtures/common_users.yaml)
```

- `[]()`のMarkdownリンクでファイルパスを指定
- YAML/CSV形式のファイルをサポート
- ファイルの内容は自動的に読み込まれ、テーブルデータとして挿入されます

#### 外部ファイルの内容例

`fixtures/common_users.yaml`:
```yaml
- id: 1
  name: "Alice"
- id: 2
  name: "Bob"
```

### 複数テーブル一括インポート（YAML）

1ファイルで複数テーブルのデータを一括で取り込みたい場合、
`**Fixtures[upsert]**` のラベルとMarkdownリンクでYAMLファイルを指定します。

**例: `fixtures/common_data.yaml`**
```yaml
users:
  - id: 1
    name: "Alice"
  - id: 2
    name: "Bob"
orders:
  - id: 100
    user_id: 1
    item: "Book"
  - id: 101
    user_id: 2
    item: "Pen"
```

Markdownテストケース側:
```markdown
**Fixtures[upsert]**
[共通データ](../fixtures/common_data.yaml)
```

- この場合、YAMLファイル内の全テーブルデータがupsert戦略で一括投入されます。
- テーブルごとに個別指定したい場合は従来通り `**Fixtures: users[upsert]**` のように記述してください。

---

### 外部ファイル方式まとめ（修正版）

- **YAML**: 1ファイルで複数テーブル（トップレベルキーで分割、一括または個別指定）
- **CSV**: 1ファイル1テーブル（ファイル名や指定でテーブル名を明示）
- Markdownリンクで参照し、テストケースごとに柔軟にインポート可能

---

### 戦略の推奨

- `upsert` のみを標準運用とし、他の戦略は必要に応じて明示的に使う設計が合理的です。
- `insert` 戦略は廃止（主キー重複時のエラー回避のため）。

### Expected Results値比較の特殊指定

YAML配列内の値として、以下の特殊値・マッチ条件を記述できます（ダブルクオート不要）。

| 記法 | 意味 |
|------|------|
| [null] | 値がnullであること |
| [notnull] | 値がnull以外であること |
| [any] | どんな値でも一致（ワイルドカード） |
| [regexp, (正規表現)] | 正規表現で一致 |
| [currentdate (, 許容差)] | 現在時刻との差が許容範囲内であること（許容差省略時は±1分） |

#### 記述例

```yaml
- id: 1
  name: [any]
  email: [notnull]
  phone: [null]
  comment: [regexp, ^foo.*bar$]
```

#### 仕様詳細

- YAML配列の値として `[null]` を指定した場合、そのカラム値がnullであることを検証
- `[notnull]` はnull以外であれば一致
- `[any]` はどんな値でも一致（値の有無は問わない）
- `[regexp, ...]` は指定した正規表現で値が一致することを検証
  - 例: `[regexp, ^foo.*bar$]` → `foo...bar` で始まり終わる文字列に一致
- `[currentdate]` は現在時刻から±1分以内で一致（例: `[currentdate, "10m"]` で±10分まで許容）

> **Note:** Fixtures でも `[currentdate, +1d]` のような配列表記を使用でき、現在時刻を基準に相対的な日時を挿入できます。`d` サフィックスは 24 時間単位として扱われます。

#### 注意事項

- YAML配列の値としてのみ利用可能（他の形式ではサポート外）
- 配列・オブジェクト型の値には適用不可（スカラー値のみ）
### Expected Results（DML結果比較・テーブル名指定）

DML（update/insert/delete）クエリの結果比較には、テーブル名を明示して期待値を記述します。

#### 記法

- SELECT/RETURNING の場合：従来通り、無名の表（配列）で結果を比較
  - 例：`**Expected Results:**`
- UPDATE/INSERT/DELETE の場合：テーブル名を明示
  - 例：`**Expected Results: users**`
  - 期待値は「users」テーブルの内容・状態を比較

#### 比較戦略（strategy）

テストケースごとに、以下の比較戦略を選択可能です（デフォルトは「all」）。

| 戦略名 | 概要 | 記法例 |
|--------|------|--------|
| all     | テーブルの内容が期待値と完全一致（行・主キー・値すべて） | `**Expected Results: users[all]**` |
| pk-match | 指定主キーの行のみ内容一致（他の行は無視） | `**Expected Results: users[pk-match]**` |
| pk-exists | 指定主キーの行が存在することのみ検証 | `**Expected Results: users[pk-exists]**` |
| pk-not-exists | 指定主キーの行が存在しないことのみ検証 | `**Expected Results: users[pk-not-exists]**` |

#### 期待値データの記述

- YAML/JSON配列、または外部ファイル参照（Markdownリンク）で記述
- 主キーはテーブル定義に従い自動判定

#### 具体例

```markdown
**Expected Results: users[all]**
```yaml
- id: 1
  name: "Alice"
- id: 2
  name: "Bob"
```

**Expected Results: users[pk-match]**
```yaml
- id: 2
  name: "Bob"
```

**Expected Results: users[pk-exists]**
```yaml
- id: 2
```

**Expected Results: users[pk-not-exists]**
```yaml
- id: 3
```

**外部ファイル参照例**
```markdown
**Expected Results: users[all]**
[期待値データ](../expected/expected_users.yaml)
```

#### 比較戦略の意味

- **all**: テーブルの内容が期待値と完全一致（行数・主キー・値すべて）
- **pk-match**: 指定主キーの行のみ内容一致（他の行は無視）
- **pk-exists**: 指定主キーの行が存在することのみ検証（値は問わない）
- **pk-not-exists**: 指定主キーの行が存在しないことのみ検証

#### デフォルト動作

- テーブル名のみ指定（`**Expected Results: users**`）の場合は「all」戦略と同じ扱い
