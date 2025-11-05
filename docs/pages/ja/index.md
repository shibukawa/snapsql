---
layout: home

hero:
  name: "SnapSQL"
  text: "Markdownでデータベーステストを書く"
  tagline: 読みやすく、保守しやすい SQLフレームワーク
  actions:
    - theme: brand
      text: Getting Started
      link: /ja/getting-started/
    - theme: alt
      text: GitHub
      link: https://github.com/shibukawa/snapsql

features:
  - icon: 🎯
    title: 記述性の高いSQLをサポートし、アプリケーションコードをシンプル化
    details: DBを単にテーブルごとのCRUD操作でしか使わずN+1クエリーなどの問題のあるコードを生み出すORMとはおさらば。DBのポテンシャルを引き出します。
  - icon: 🔧
    title: 型安全な薄いクライアントコード生成
    details: 現在はGoに対応。トランザクションを隠さないDBパフォーマンスを引き出しやすいコードを生成。結果の階層構造化、イテレーションにも対応。
  - icon: 📝
    title: 生成AIフレンドリーな文芸的SQL
    details: 1ファイルのMarkdownで1クエリーを表現。テストケースも記述可能で設計意図なども記述できる。
  - icon: ⚡
    title: 最速のフィードバックループ
    details: アプリケーションコードなしでSQLのテストを可能にします。並列テスト実行とトランザクション管理により、大量のテストを高速に実行できます。スキーマとSQLの品質アップに貢献します。
  - icon: 🧪
    title: システム全体の品質アップをサポート
    details: 常に実際のレスポンスと一致していることが保証されたモックレスポンスを実現。またクエリーのパフォーマンス劣化の予兆のインサイトを提供します。
  - icon: 🗄️
    title: 複数データベース対応
    details: PostgreSQL、MySQL、SQLiteをサポート。同じテストケースで異なるデータベースをテストできます。方言も吸収する。
---

## ワークフロー

![workflow](/assets/workflow.webp)


## クイックスタート

### 1. SQLクエリの記述

MarkdownでSQLクエリとテストケースを記述：

````markdown
# Get Cards with Comments

## Description

コメント付きでカードのリストを返します。

## Parameters

```yaml
list_id: int
```

## SQL

```sql
SELECT 
    c.id,
    c.title,
    c.description,
    c.created_at,
    cc.id AS comments__id,
    cc.body AS comments__body,
    cc.created_at AS comments__created_at
FROM cards AS c
    LEFT JOIN card_comments AS cc
        ON c.id = cc.card_id
WHERE c.list_id = /*= list_id */1
ORDER BY c.position ASC, cc.created_at ASC;
```
````

### 2. コード生成

```bash
snapsql generate
```

* 実際の実行にはレスポンス型の推論やレスポンスが主キー指定で単一行のみになるか複数行返るかの推論のために、[tbls](https://github.com/k1LoW/tbls)を使って実テーブルやSQLから抽出したschema.jsonの準備が必要

### 3. アプリケーションでの使用

生成されたGoコードを使用：

```go
// 生成された型安全なクライアントコードを使用
//
// 複数レコードレスポンスではiter.Seq2退いてレーション形式に対応
for card, err := range client.GetCardsWithComments(ctx, 1) {
    fmt.Printf("Card: %s\n", card.Title)
    // SELECT句のレスポンス名で__を使用することでJOINした子要素をリスト化
    for _, comment := range card.Comments {
        fmt.Printf("  Comment: %s\n", comment.Body)
    }
}
```

## 詳細を学ぶ

- [Getting Started](/ja/getting-started/) - ステップバイステップのチュートリアル
- [Guides](/ja/guides/command-reference/) - 詳細なガイドとリファレンス
- [Samples](/ja/samples/) - 実践的なサンプルプロジェクト

## コントリビューション

SnapSQLはオープンソースプロジェクトです。コントリビューションを歓迎します！

- [GitHub リポジトリ](https://github.com/shibukawa/snapsql)
- [Issue トラッカー](https://github.com/shibukawa/snapsql/issues)
- [コントリビューションガイド](/ja/development/contributing)
