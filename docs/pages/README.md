# SnapSQL Documentation

このディレクトリにはSnapSQLのユーザードキュメントが含まれています。

## 開発

### 前提条件

- Node.js 18以上
- npm または yarn

### セットアップ

```bash
cd docs/pages
npm install
```

### 開発サーバー起動

```bash
npm run docs:dev
```

ブラウザで http://localhost:5173/ を開いてください。

### ビルド

```bash
npm run docs:build
```

ビルド結果は `.vitepress/dist/` に出力されます。

### プレビュー

ビルド後のドキュメントをプレビュー：

```bash
npm run docs:preview
```

## ディレクトリ構造

```
docs/pages/
├── .vitepress/          # VitePress設定
│   ├── config.ts        # メイン設定
│   ├── config/
│   │   └── ja.ts        # 日本語設定
│   └── theme/           # カスタムテーマ
│       ├── index.ts
│       └── style.css
├── public/              # 静的ファイル
├── ja/                  # 日本語ドキュメント
│   ├── index.md         # トップページ
│   ├── getting-started/ # Getting Started
│   ├── guides/          # ガイド
│   │   └── commands/    # コマンド別ガイド
│   ├── samples/         # サンプル
│   └── development/     # 開発者ガイド
└── package.json
```

## ドキュメント作成ガイドライン

### フロントマター

各ページの先頭にはフロントマターを追加できます：

```yaml
---
title: ページタイトル
description: ページの説明
---
```

### コードブロック

VitePressは行番号表示をサポートしています：

\```go
package main

func main() {
    // コード
}
\```

### カスタムコンテナ

```
::: tip
ヒント
:::

::: warning
警告
:::

::: danger
危険
:::

::: info
情報
:::
```

### リンク

- 内部リンク: `[テキスト](./path/to/page)`
- 外部リンク: `[テキスト](https://example.com)`

### ナビゲーション

ページ下部の「前へ」「次へ」ボタンは自動的に生成されます。
サイドバーの設定は `.vitepress/config/ja.ts` で管理しています。

## デプロイ

### GitHub Pages

GitHub Actionsを使用して自動デプロイできます。
設定ファイルは `.github/workflows/deploy-docs.yml` を参照してください。

## トラブルシューティング

### ポート競合

別のポートで起動する場合：

```bash
npm run docs:dev -- --port 3000
```

### キャッシュクリア

ビルドキャッシュをクリアする場合：

```bash
rm -rf .vitepress/cache
```

## 参考リンク

- [VitePress公式ドキュメント](https://vitepress.dev/)
- [VitePress GitHub](https://github.com/vuejs/vitepress)
