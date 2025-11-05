````markdown
# generate コマンド

`snapsql generate` は、テンプレート（SQL/.snap.md/.snap.sql/.md）と設定から中間フォーマット（JSON）を生成し、その中間ファイルを元に言語ごとのコード生成を行います。

このドキュメントは実装に沿った挙動を説明します。将来の構想（内部プラグイン API 等）についての記載は含めていません。

## 概要

入力テンプレートやスキーマ情報を元にコードを生成します。

スキーマ情報（tbls が出力するランタイム用スキーマ JSON）はプロジェクトの設定や `--tbls-config` 等の設定から出力先を推定して利用します。スキーマが見つからない場合は生成が失敗します。

## 例

```bash
# プロジェクト設定に従って全てのジェネレータを実行
snapsql generate

# Go のみを生成
snapsql generate --lang go --package=mypkg
```

## 出力と設定

- まず intermediate JSON（デフォルト出力先 `./generated`）を生成します。`snapsql.yaml` の `generation.generators.json.output` を使って出力先を変更できます。
- `generation.generators.*.preserveHierarchy` を true にすると、入力ディレクトリの階層を保持したサブディレクトリ構成で intermediate ファイルを出力します。
- built-in ジェネレータ:
  - `go` : Go用のジェネレータ
  - `json` : 中間ファイルを出力するためのジェネレータ。現時点では他のジェネレータを実装するためのデバッグ用途
  - `mock` : モック機能のためにテストセクションのExpected Resultsを抜き出すジェネレータ

## フラグ

- `--input, -i <path>` : 入力ファイルまたはディレクトリ。`snapsql.yaml` の `input_dir` を優先するため通常は指定不要。
- `--lang <name>` : 生成対象の言語（例: `go`, `json`, `mock`）。指定しない場合は設定に基づいて有効なジェネレータをすべて実行する。
- `--package <name>` : 生成先のパッケージ/名前空間（言語依存）。
- `--const, -c <file>` : 定数定義ファイルを追加で読み込み（YAML）。複数指定可。
- `--validate` : 生成前にテンプレートの静的検証を行う。
