# アサーション（マッチャー）の使い方# Assertions（アサーション）



このページでは、期待結果内で使うマッチャー（assertions）の主な使い方をまとめます。多くの基本的なマッチャーは `expected-results.md` の「マッチャー」節で説明されていますが、簡単にここでもまとめます。クエリテストで利用できるアサーションの手法とベストプラクティスをまとめます。



主なマッチャー:主な項目:



- `[null]` / `"null"` — 実際の値が NULL であることを期待します。- Expected Results における値の比較（リテラル、マッチャーの使い分け）

- `[notnull]` / `"notnull"` — 実際の値が NULL でないことを期待します。- マッチャー一覧（`[notnull]`, `[any]`, `[currentdate]`, `[regexp, ...]` など）と使用例

- `[any]` — 任意の値を許容します（存在チェック用）。- UPDATE/DELETE などの非行返却クエリの検証戦略（影響行数、状態検証など）

- `[currentdate, "<duration>"]` — 値が日時で、現在時刻との差が許容値以内であることを検証します。

- `[regexp, "<pattern>"]` — 正規表現でマッチすることを検証します。このページでは具体的な YAML 例やパターンを示します。`Expected Results` はテストの可読性と堅牢性を左右するため、変動要素にはマッチャーを使い、固定値はリテラルで検証する方針を推奨します。



使い方の例:### 基本方針



```yaml- 固定値: 明確に期待する値をリテラルで書く（例: `name: "alice"`）

- id: 42- 可変値: 時刻や UUID、シーケンス等の変動する値はマッチャーを使う（例: `created_at: [currentdate]`）

  name: [regexp, "^Alice.*"]- 存在チェック: 値の有無だけ確認したい場合は `[any]` / `[notnull]` を使う

  updated_at: [currentdate, "2m"]- 厳密比較が必要な場合はリテラル → それ以外はマッチャー、の境界を明確にする

```

<!-- 移行メモ: このファイルの内容は `fixtures.md` と `expected-results.md` に統合されました。 -->

詳細と実装の挙動（NULL の扱い、数値比較の正規化など）は `expected-results.md` を参照してください。
このファイルはアサーションに関する旧来のまとめでしたが、情報を分散せず `fixtures.md` と `expected-results.md` に統合しました。

主な参照先:

- Fixtures（フィクスチャ）: `docs/pages/ja/guides/query-format/fixtures.md`
- Expected Results（期待結果）: `docs/pages/ja/guides/query-format/expected-results.md`

必要があればこのページを削除するか、さらに詳しい例（例: 階層レスポンスや UPDATE/DELETE 検証パターン）を統合して戻すこともできます。変更希望があれば指示してください。
- `[len, >=, N]`: 長さチェック（存在する実装用の提案例）
