# フィクスチャ

テスト用データ（Fixtures）の構造、管理、ロード戦略についてのガイドです。ここでは実装に合わせて、対応フォーマット、特殊リテラル、外部ファイル共有、ロード戦略、正規化の振る舞いまで詳述します。

- テストは `.snap.md` や `.md` の `## Test Cases` セクション内に記述します。各テストケースは `###`（H3）で開始します。
- サブセクション（Fixtures / Parameters / Expected Results / Verify Query / Expected Error 等）は見出しではなく、太字（例: `**Fixtures:**`）で記述してください。実装のパーサはこの太字ラベルを探して各セクションを抽出します。

例（テストケースの最小構成）:

````markdown
## Test Cases

### Test: Basic user retrieval

**Fixtures:**
```yaml
users:
  - id: 1
    name: "太郎"
```

**Parameters:**
```yaml
user_id: 1
```

**Expected Results:**
```yaml
  name: "太郎"
```
````

テストケースの書式（セクションのラベル、サンプルの最小構成）、ライフサイクル、実行コマンド、ベストプラクティスは利用者向けドキュメントにまとめました。フォーマット（Fixtures の YAML/CSV の扱いや特殊リテラルなど）のみをこのページで説明します。

参照: `docs/pages/ja/guides/user-reference/test.md`

## Fixtures の指定方法

このページでは主に次の 3 方式を想定して説明します。新規にテストを書く場合は YAML/JSON を推奨します。

### YAML / JSON

最も汎用的で推奨される方法です。1 つの YAML/JSON ブロック内に複数テーブルのデータをネストして記述できます。たとえば:

```yaml
users:
  - id: 1
    name: "太郎"
posts:
  - id: 10
    user_id: 1
    title: "最初の投稿"
```

利点:

- 複数テーブルをまとめて指定でき、テストケースの可読性が高い
- 階層データやネストしたオブジェクトを表現しやすい

注意点:

- テーブル名と列名の正確さは重要です。スキーマがある場合は未知の列や型不一致でエラーになります。

### CSV

CSV ブロックは 1 セクションにつき 1 テーブル分を記述する形式です。既存の CSV 資産や簡易的なテーブルを素早く流用する際に便利です。

例:

```csv
table: users
id,name,email
1,太郎,taro@example.com
2,花子,hanako@example.com
```

注意事項:

- CSV ブロックは単一テーブルの記述に適しており、複数テーブルを同一ブロックで表現できません（複数テーブルが必要な場合はブロックを分けてください）。
- CSV の列とテーブル名のルールが実装側のパーサと一致している必要があります。

注意（CSV の扱い）:

- CSV を使う場合、テーブル名は CSV ブロック内ではなく「セクションのラベル」で指定する必要があります。ラベルは次の形式で記述してください: `**Fixtures: <table>[strategy]**`。

  例（parser のテストで使われている形式）:

  **Fixtures: departments**
  ```csv
  id,name
  1,Engineering
  2,Design
  ```

  上記のとおり、CSV ブロックの先頭行は列ヘッダであり、テーブル名ではありません。パーサはラベルの文字列を `parseTableNameAndStrategy` で解析してテーブル名とロード戦略を決定します（`markdownparser/testcase.go` と `markdownparser/insert_strategy_test.go` を参照）。

### DBUnit / XML

既存の DBUnit や XML フォーマットを取り込みたい場合に使います。こちらも 1 セクションにつき 1 テーブル（または DBUnit が想定する単位）を記述する形になります。

注意事項:

- XML の読み込みはプロジェクトで対応している場合に限ります。`parseStructuredData` がサポートしているか確認してください。
- マイグレーション目的で既存資産を使うには有用ですが、新規記述では YAML/JSON を推奨します。

例（DBUnit XML のサンプル）:

```xml
<dataset>
  <roles id="1" name="Admin" />
  <roles id="2" name="User" />
</dataset>
```

DBUnit XML では `<dataset>` の直下にあるタグ名がテーブル名として扱われ、各タグの属性が列になります。実装上は `parseDBUnitXML` がこの形式をパースします（`markdownparser/dataformat.go` を参照）。

### 外部ファイルの参照

フィクスチャを外部ファイルに分離して共有することができます。Markdown 内で次のようにコードブロックの代わりにリンク記法を使うと、実行時にそのファイルが読み込まれます:

```
[common-users.yaml](fixtures/common-users.yaml)
```

外部ファイルは YAML/JSON の配列やテーブル名付き構造を返すべきです。CSV/XML の外部参照も可能ですが、参照先のフォーマットに合わせてセクションを分けてください。

ポイント:

- `**Parameters:**` は通常 1 回のみ（テストケースあたり単一のパラメータセット）。
- `**Fixtures:**` は複数記述可能で、外部ファイル参照もサポートします。
- `**Expected Results:**` または `**Expected Error:**` のどちらかを指定して結果を検証します。

ポイント:

- フィクスチャは Markdown 内で YAML/JSON として埋め込む方式が標準です（`## Test Cases` 内）。
- CSV（`csv` コードブロック）や DBUnit 形式等、複数フォーマットの読み込みをサポートします。
- デフォルトのロード戦略は `clear-insert`（テーブルをクリアしてから挿入）です。必要に応じて `upsert` や `delete` を選べます。
 - デフォルトのロード戦略は `clear-insert`（テーブルをクリアしてから挿入）です。必要に応じて `upsert` や `delete` を選べます。
 - 大きなデータセットでは `upsert` を使うとテスト時間が短縮できます。

## ロード戦略

ロード戦略

- clear-insert: テーブルを TRUNCATE/DELETE してから挿入（デフォルト）。確実にクリーンな状態にします。
- upsert: 主キーが一致する場合は更新、存在しない場合は挿入。大量の共通データを維持しつつ、テスト固有の行だけ差分で用意したい場合に有効です。
- delete: データセットに記載された主キーを削除します（削除検証用）。
- transaction-wrapped: テスト中にトランザクションでラップして最後にロールバックする運用。ドライバや実行モードによっては制約があります。

実装側では `TableFixture.Strategy` により各テーブルごとに戦略を指定できます（`markdownparser/testcase.go` と `testrunner/fixtureexecutor/executor.go` を参照）。

### フィクスチャの分割と共有

テストの規模や再利用性に応じてフィクスチャを分割・整理してください。共通データの扱いはプロジェクト方針に合わせて設計し、必要に応じてテストケースごとに読み込む方式や共通セットを採用してください。

### パフォーマンスのヒント

- テストスイートで毎回大量データをロードすると CI が遅くなる。可能なら必要最小限のデータのみをロードする
- 並列実行時はデータ競合を避けるために、各ワーカーごとにプレフィックス付きテーブルやスキーマを用いる

加えて、`upsert` 戦略を多用すると挿入コストが下がることがある反面、実装によっては複雑さが増します。CI では小さいセットを作りつつ、必要な境界条件だけを含めることが安定化のコツです。

### 実践例：upsert 戦略の YAML

```yaml
- name: "ensure post exists"
	fixtures:
		posts:
			- id: 100
				title: "seeded"
load_strategy: upsert
```

上の例は簡易的な表現です。実装では各テーブルの戦略を `fixtures:` 内で指定できます（`TestSection` の `Strategy` を参照）。

### 対応フォーマット

（YAML/JSON、CSV、DBUnit/XML の各フォーマットは本文の `Fixtures の指定方法` セクションで詳述しています）

## 特殊リテラル

フィクスチャに書いた値は挿入前に `normalizeFixtureRows` → `resolveFixtureValue` を経て正規化されます。
ここでは実務でよく使う特殊リテラル（マッチャー）とその具体的な振る舞いを実装に沿ってまとめます。

主な特殊リテラル／マッチャー（Fixtures と Expected Results の両方で利用可能）:

- `[null]` / `"null"`
  - DB の NULL を挿入または期待する場合に使います。YAML 側で配列表記（`[null]`）とすることで `nil` に変換されます。

- `[notnull]`
  - 値が NULL でないことを示します（期待値検証時に使用）。

- `[any]`
  - 任意の値を許容します。値そのものは検証せず存在のみ確認したい場合に便利です。実装により NULL を許容するかは方針が異なるため注意してください。

- `[currentdate]` / `[currentdate, <offset>]`
  - 実行時の現在時刻を表す特殊値です。挿入用・検証用どちらでも使えます。`<offset>` は `-1h`, `+30s`, `1d` などの相対指定が可能で、内部でパースして時刻に適用します。比較時は許容誤差（デフォルトは短時間）を持たせる設計です。

- `[regexp, <pattern>]`
  - 指定の正規表現にマッチすることを期待します。Go の `regexp` 構文に従います。複雑なパターンは CI 側で事前検証してください（ReDoS リスク等）。

実装上のポイント:

- 型と正規化: 数値は比較時に float64 に正規化されるため、`1` と `1.0` は等価とみなされます。日時は複数フォーマットでパースし時刻オブジェクトで比較されます。
- 再帰解決: ネストした配列やオブジェクトも再帰的に `resolveFixtureValue` が処理します。
- エラー: マッチャーの引数が不正（不正な日付表記、無効な正規表現等）だとパース/実行前にエラーになります。

使用例（Fixtures 内）:

```yaml
fixtures:
  users:
    - id: 1
      name: "alice"
      email: [regexp, '^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$']
      created_at: [currentdate]
      deleted_at: [null]
```

実装参照:

- フィクスチャの正規化と値解決: `testrunner/fixtureexecutor/executor.go` (`normalizeFixtureRows`, `resolveFixtureValue`)
- 外部ファイル読み込み: `markdownparser/testcase.go` の外部リンク抽出と `parseStructuredData`

運用のヒント:

- タイムゾーン: `[currentdate]` の比較は環境差で失敗しやすいので UTC 正規化や許容幅の調整を検討してください。
- 正規表現: 単純なパターンを使う、あるいは事前検証することでテストの安定性を高められます。
- 大量データ: `upsert` を併用するとセットアップコストを削減できますが、期待結果との整合性に注意してください。
