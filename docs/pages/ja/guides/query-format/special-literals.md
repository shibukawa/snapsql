# 特殊リテラル<!-- 移行メモ: ここにあった "特殊リテラル" 記述は `fixtures.md` と `expected-results.md` に統合されました。 -->



このページでは、テスト記述でよく使われる特殊リテラル（NULL、日時、特殊トークンなど）についてまとめます。このファイルは内容を `fixtures.md` と `expected-results.md` に統合したため、参照用の短い案内ページになっています。



- `NULL` / `[null]` の扱いは厳密で、実際のデータベースの NULL（Go の `nil`）と比較します。詳細なマッチャー（`[null]`, `[notnull]`, `[any]`, `[currentdate]`, `[regexp, ...]` など）や利用例は次のドキュメントを参照してください。

- 日時系のマッチャーは `[currentdate, "<duration>"]` のように許容幅を指定できます。

- `"any"` / `[any]` は値の存在だけを検証したいときに使います。- Fixtures（フィクスチャ）: `docs/pages/ja/guides/query-format/fixtures.md`

- Expected Results（期待結果）: `docs/pages/ja/guides/query-format/expected-results.md`

実装の細かい挙動（たとえば `TEXT` カラムが `[]byte` で返る場合の吸収、数値の正規化など）は `expected-results.md` を参照してください。
もし古いドキュメントのバックアップが必要であれば git 履歴から復元できます。新しいドキュメントに不整合や追記希望があれば PR を作成してください。
  - 指定した正規表現にマッチすることを期待します。パターンは Go の `regexp` 構文に準拠します。

- その他（プロジェクトで追加可能）: `[uuid]`, `[len, >=, N]` などの拡張マッチャーを導入できます。

利用例（YAML 形式の Expected Results）:

```yaml
- id: 123
  name: "alice"
  email: [regexp, '^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$']
  deleted_at: [null]
  created_at: [currentdate]
  token: [regexp, '^[a-z0-9]{32}$']
```

Fixtures での使用例:

```yaml
Fixtures:
  users:
    - id: 1
      name: "alice"
      email: [any]
```

仕様・注意点

- 型チェック: 一部マッチャーは対象値の型に依存します（例: `[currentdate]` は日時型で評価されることを想定）。実行前に文字列を日時へ変換するなどの前処理が行われる場合があります。
- タイムゾーン: `[currentdate]` の比較はタイムゾーン差を吸収するために UTC 正規化や許容幅の設定を推奨します。
- 許容誤差: 動的値（日時、シーケンス、トークン等）は比較に許容誤差を持たせましょう。
- 正規表現の安全性: ユーザー入力を直接正規表現に埋め込むと ReDoS リスクがあるため、CI のテストデータでは単純なパターンを使うか、事前検証を行ってください。

拡張案

- よく使われるマッチャー（`[uuid]`, `[len, >=, N]`）をフレームワーク側で実装すると、テスト記述がより簡潔になります。

このページは基本仕様のまとめです。実装側のマッチャー挙動（NULL 許容や許容幅）についてはプロジェクトポリシーとして明文化してください。

