# テスト概要

このページでは SnapSQL におけるテストの基本的な考え方と主要な構成要素をまとめます。

主な要素:

- フィクスチャ（テストデータ）の準備 — `fixtures.md`
- 期待結果（Expected Results）の記述方法 — `expected-results.md`
- マッチャー（アサーション）の使い方 — `assertions.md`
- 特殊リテラル（NULL・日付など）の取り扱い — `special-literals.md`

おすすめの読み順:

1. `fixtures.md` — テストデータをどう準備するか
2. `test-overview.md`（このページ） — 全体像の確認
3. `expected-results.md` — 結果検証の方法
4. `assertions.md` と `special-literals.md` — 詳細な振る舞い

実装や詳細はプロジェクト内の他のガイドや `docs/designdocs/` の設計書（GitHub 上）も参照してください。