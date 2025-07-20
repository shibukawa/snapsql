generators:
  go:
    package: "user"
    output: "./generated/user.go"
    use_struct: true
    input_type: "map[string]any"
    output_type: "map[string]any"
  typescript:
    output: "./generated/user.ts"
    use_interface: true

# FunctionDefinition 仕様

## 概要

FunctionDefinitionは、SQLテンプレートやクエリー定義からアプリケーションコードを生成するための関数定義情報を統合的に管理する構造体です。各関数の生成に必要な属性・パラメータ・メタ情報・言語ごとのジェネレータ設定などを持ちます。

## 必須項目

- **FunctionName** (`string`)
  - 生成される関数名。ファイル名やfrontmatterから抽出され、言語ごとの命名規則に従い変換される。
- **Description** (`string`)
  - 関数の説明。ドキュメントやコードコメント等に利用。
- **Parameters** (`map[string]any`)
  - 入力パラメータ定義。型情報（int, str, bool, 配列、ネスト構造等）を持ち、YAML定義順序も保持される。型生成・バリデーション・引数順序に利用。
  - 共通型参照（大文字で始まる型名）を含む場合、Finalize時に解決される。

## オプション項目

- **Generators** (`map[string]map[string]any`)
  - 各言語ジェネレータ用の追加属性。最初のキーが言語名（例: "go", "typescript"）、値は言語ごとの任意属性（例: Goならpackage名、出力先、構造体生成有無、入出力型指定など）。
- **CommonTypesPath** (`string`)
  - 共通型定義ファイルのパス。デフォルトは処理対象ファイルと同じディレクトリの`_common.yaml`。
- **CommonTypes** (`map[string]CommonTypeDefinition`)
  - 解決された共通型定義のキャッシュ。Finalize時に構築される。

### Generators 利用例

```yaml
generators:
  go:
    package: "user"
    output: "./generated/user.go"
    use_struct: true
    input_type: "map[string]any"
    output_type: "map[string]any"
  typescript:
    output: "./generated/user.ts"
    use_interface: true
```

## 設計方針・備考

- パラメータは階層構造（ネスト、配列、オブジェクト）をサポート
- YAMLノードから順序付きでパラメータを抽出し、型生成や引数順序に反映
- 言語ごとのジェネレータ属性は拡張可能
- テンプレート定義の柔軟性と型安全性を両立

## Markdownファイルとのマッピング

FunctionDefinitionはMarkdownクエリー定義ファイル（.snap.md）から以下の3つの情報源を合成して抽出されます。

- **FrontMatter**
  - ファイル先頭のYAML frontmatter（---で囲まれた部分）から、FunctionNameやジェネレータ設定などのメタ情報を抽出します。
- **Overviewセクション**
  - `## Overview` セクションのテキスト部分が Description に割り当てられます。
- **Parametersセクション**
  - `## Parameters` セクション内のYAMLコードブロックが Parameters に割り当てられます。

この3つの情報を合成し、FunctionDefinition仕様に必要な情報を抽出・統合します。

## 共通型定義機能

### 概要

共通型定義機能により、複数のクエリー間で同じパラメータ構造（例：ユーザー情報、部署情報など）を再利用できます。共通型は各ディレクトリの`_common.yaml`ファイルに定義され、相対パスで参照できます。

### 共通型定義ファイル

- 各ディレクトリに`_common.yaml`ファイルを配置
- 大文字で始まる識別子が構造体型として認識される
- 同一ファイル内で型同士の参照も可能
- 型定義とフィールドにはコメントを付けることができる

### 共通型定義例

```yaml
# _common.yaml
User: # ユーザー情報を表す共通型
  id: int # ユーザーID
  name: string # ユーザー名
  email: string # メールアドレス
  department: Department # 所属部署

Department: # 部署情報を表す共通型
  id: int # 部署ID
  name: string # 部署名
  manager_id: int # 部署管理者ID
```

### 共通型の参照方法

パラメータ定義内で以下の形式で共通型を参照できます：

1. **同一ディレクトリの型参照**:
   ```yaml
   parameters:
     user: User        # 同一ディレクトリの_common.yamlから参照
     admin: .User      # 明示的に同一ディレクトリを指定（.で始まる）
   ```

2. **別ディレクトリの型参照**:
   ```yaml
   parameters:
     user: ../User     # 親ディレクトリの_common.yamlから参照
     member: ./sub/User # サブディレクトリの_common.yamlから参照
   ```

3. **配列型としての参照**:
   ```yaml
   parameters:
     users: User[]     # Userの配列
     departments: ../Department[] # 親ディレクトリのDepartmentの配列
   ```

### 型解決の仕組み

1. パラメータ型が大文字で始まる場合、共通型として解釈
2. パスが指定されている場合（`./`, `../`など）、そのパスの`_common.yaml`を参照
3. パスがない場合、現在のディレクトリの`_common.yaml`を参照
4. 共通型が見つからない場合はエラー
5. 循環参照は許容される（言語によっては循環参照が許可されるため）

### 実装上の注意点

- 共通型の解決は、`FunctionDefinition.Finalize()`メソッド内で一括して行われる
- 共通型の参照解決は、相対パスを基準に行われる
- 共通型のコメントはドキュメント生成時に活用される
- 共通型が変更された場合は、それを参照する全ての関数定義を再生成する必要がある

---

（2025-07-20 更新：共通型定義機能の追加）
## 共通型定義の実装

FunctionDefinition構造体に共通型定義を処理するための機能を追加します：

```go
type FunctionDefinition struct {
    // 既存のフィールド
    Name           string
    Description    string
    FunctionName   string
    Parameters     map[string]any
    ParameterOrder []string
    RawParameters  yaml.MapSlice
    Generators     map[string]map[string]any
    
    // 共通型関連の追加フィールド
    commonTypes    map[string]map[string]any  // 読み込んだ共通型定義（内部使用）
    basePath       string                     // 相対パス解決の基準パス
}
```

### 共通型の解決プロセス

1. `FunctionDefinition.Finalize()`メソッド内で、必要な`_common.yaml`ファイルを読み込む
2. パラメータ内の共通型参照（大文字で始まる型名）を検出
3. 相対パスを解決し、対応する共通型定義を取得
4. パラメータ定義を共通型定義で置き換える
5. 共通型内の参照も再帰的に解決する

### 処理後の状態

共通型の解決が完了すると、`Parameters`フィールドには共通型が展開された状態の型定義が格納されます。これにより、後続の処理（ダミーデータ生成、コード生成など）は、共通型を意識することなく既存のロジックをそのまま利用できます。
## 共通型定義ファイル

### ファイル形式

共通型定義ファイルは、YAMLフォーマットで記述されます。ファイル名は`_common.yaml`とし、各ディレクトリに配置します。

```yaml
# _common.yaml
User: # ユーザー情報を表す共通型
  id: int # ユーザーID
  name: string # ユーザー名
  email: string # メールアドレス
  department: Department # 所属部署（同一ファイル内の別の型を参照可能）

Department: # 部署情報を表す共通型
  id: int # 部署ID
  name: string # 部署名
  manager_id: int # 部署管理者ID
```

### 読み込み方法

共通型定義ファイルは、`FunctionDefinition.Finalize()`メソッドの実行時に一括して読み込まれます：

1. 現在のディレクトリの`_common.yaml`を読み込む
2. パラメータ内の相対パス参照（例：`../User`）を検出し、対応するディレクトリの`_common.yaml`も読み込む
3. 読み込んだ共通型定義は、FunctionDefinitionのcommonTypesマップに格納される

### パス解決

共通型参照のパス解決は、以下のルールに従います：

1. パスが指定されていない場合（例：`User`）
   - 現在のディレクトリの`_common.yaml`から型を検索
2. 相対パスが指定されている場合（例：`../User`、`./sub/User`）
   - 指定されたパスの`_common.yaml`から型を検索
3. 絶対パスは未サポート

### エラー処理

以下の場合にエラーが発生します：

1. 参照された共通型が見つからない場合
2. 共通型定義ファイルが存在しない場合
3. 共通型定義ファイルの形式が不正な場合

エラーメッセージには、問題が発生した場所（ファイルパス、行番号）と原因が含まれます。
