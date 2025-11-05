# アーキテクチャ

SnapSQL の内部構造と設計について説明します。

## 目次

### コア機能

- [パーサーフロー](./parser-flow.md) - パース処理のステップ
- [中間コード生成](./intermediate-generation.md) - AST から IR への変換
- [コード生成](./code-generation.md) - IR から各言語コードへの生成
- [型推論](./type-inference.md) - パラメータと戻り値の型推論

## 処理の流れ

```
Markdown (.snap.md)
    ↓
字句解析（Tokenizer）
    ↓
構文解析（Parser）
    ↓
AST（抽象構文木）
    ↓
中間形式（IR）
    ↓
プロセッサパイプライン
    ↓
コード生成
    ↓
言語別コード（Go/TypeScript/etc.）
```

## 設計原則

### 1. 分離と独立性

各ステップは独立しており、テスト・保守が容易です。

### 2. 拡張性

新しい言語やデータベース方言の追加が容易です。

### 3. 型安全性

パース時から型情報を保持し、コード生成時に活用します。

### 4. エラー報告

詳細な位置情報とコンテキストを含むエラーメッセージを提供します。

## カスタマイズ

### 独自ジェネレータの追加

[コード生成](./code-generation.md) を参照してください。

### 独自プロセッサの追加

`intermediate` パッケージに新しいプロセッサを実装できます。

### 独自コマンドの追加

`cmd/snapsql` に新しいサブコマンドを追加できます。

## 関連ドキュメント

- [設計ドキュメント](https://github.com/shibukawa/snapsql/tree/main/docs/designdocs/) - 詳細な設計ドキュメント（GitHub 上の設計書）
- [開発ガイド](https://github.com/shibukawa/snapsql/blob/main/docs/development.ja.md) - 開発環境のセットアップ（GitHub 上のドキュメント）
