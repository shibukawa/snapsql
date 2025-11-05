# コマンドリファレンス

SnapSQL の CLI コマンドの詳細を説明します。

## 目次

### 初期化・設定

- [init](./init.md) - プロジェクトの初期化

### 開発

- [inspect](./inspect.md) - クエリファイルの検査と中間形式の出力
- [format](./format.md) - クエリファイルの整形

### クエリ実行

- [query](./query.md) - クエリの実行
- [test](./test.md) - テストの実行

### コード生成

- [generate](./generate.md) - 各言語のコード生成

## 共通オプション

すべてのコマンドで使用できるグローバルオプション：

```bash
--config string      設定ファイルのパス (デフォルト: snapsql.yaml)
--verbose           詳細出力
--help              ヘルプを表示
```

## 関連セクション

- [利用者向けリファレンス](../user-reference/) - 設定と機能
- [クエリーフォーマット](../query-format/) - クエリファイルの書き方
