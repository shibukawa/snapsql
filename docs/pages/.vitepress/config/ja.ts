import { defineConfig, type DefaultTheme } from 'vitepress'

export const ja = defineConfig({
  lang: 'ja-JP',
  description: 'Markdownでデータベーステストを書く',

  themeConfig: {
    nav: [
      { text: 'Getting Started', link: '/ja/getting-started/' },
      { text: 'Guides', link: '/ja/guides/user-reference/' },
      { text: 'Samples', link: '/ja/samples/' },
      { text: 'Development', link: '/ja/development/architecture' }
    ],

    sidebar: {
      '/ja/getting-started/': [
        {
          text: 'Getting Started',
          items: [
            { text: '0. 概要と全体の流れ', link: '/ja/getting-started/' },
            { text: '1. インストール', link: '/ja/getting-started/installation' },
            { text: '2. プロジェクトセットアップ', link: '/ja/getting-started/project-setup' },
            { text: '3. SQLクエリーの作成', link: '/ja/getting-started/write-sql-query' },
            { text: '4. クエリーの実行', link: '/ja/getting-started/query' },
            { text: '5. テストケースの作成と実行', link: '/ja/getting-started/testing' },
            { text: '6. コード生成と呼び出し', link: '/ja/getting-started/code-generation' }
          ]
        }
      ],
      '/ja/guides/': [
        {
          text: '利用者向けリファレンス',
          collapsed: false,
          items: [
            { text: '概要', link: '/ja/guides/user-reference/' },
            { text: '設定', link: '/ja/guides/user-reference/configuration' },
            { text: 'システムカラム', link: '/ja/guides/user-reference/system-columns' },
            { text: 'データベース方言', link: '/ja/guides/user-reference/dialects' },
            { text: 'トランザクション', link: '/ja/guides/user-reference/transactions' },
            { text: 'ユニットテスト', link: '/ja/guides/user-reference/test' },
            { text: 'モック機能', link: '/ja/guides/user-reference/mock' }
          ]
        },
        {
          text: 'コマンドリファレンス',
          collapsed: false,
          items: [
            { text: '概要', link: '/ja/guides/command-reference/' },
            { text: 'init', link: '/ja/guides/command-reference/init' },
            { text: 'inspect', link: '/ja/guides/command-reference/inspect' },
            { text: 'query', link: '/ja/guides/command-reference/query' },
            { text: 'test', link: '/ja/guides/command-reference/test' },
            { text: 'generate', link: '/ja/guides/command-reference/generate' },
            { text: 'format', link: '/ja/guides/command-reference/format' }
          ]
        },
        {
          text: 'クエリーフォーマット',
          collapsed: false,
          items: [
            { text: '概要', link: '/ja/guides/query-format/' },
            { text: 'Markdownフォーマット', link: '/ja/guides/query-format/markdown-format' },
            { text: 'テンプレート構文', link: '/ja/guides/query-format/template-syntax' },
            { text: 'パラメータ', link: '/ja/guides/query-format/parameters' },
            { text: '共通型', link: '/ja/guides/query-format/common-types' },
            { text: 'レスポンス型', link: '/ja/guides/query-format/response-types' },
            { text: 'ネストしたレスポンス', link: '/ja/guides/query-format/nested-response' },
            { text: 'テストフィクスチャ', link: '/ja/guides/query-format/fixtures' },
            { text: 'テスト結果の期待値', link: '/ja/guides/query-format/expected-results' },
            { text: 'エラーテスト', link: '/ja/guides/query-format/error-testing' }
          ]
        },
        {
          text: '言語別リファレンス',
          collapsed: false,
          items: [
            { text: '概要', link: '/ja/guides/language-reference/' },
            { text: 'Go', link: '/ja/guides/language-reference/go' }
          ]
        },
        {
          text: 'アーキテクチャ',
          collapsed: false,
          items: [
            { text: '概要', link: '/ja/guides/architecture/' },
            { text: 'パーサーフロー', link: '/ja/guides/architecture/parser-flow' },
            { text: '中間コード生成', link: '/ja/guides/architecture/intermediate-generation' },
            { text: 'コード生成', link: '/ja/guides/architecture/code-generation' },
            { text: '型推論', link: '/ja/guides/architecture/type-inference' }
          ]
        }
      ],
      '/ja/samples/': [
        {
          text: 'Samples',
          items: [
            { text: '概要', link: '/ja/samples/' },
            { text: 'Basic CRUD', link: '/ja/samples/basic-crud' },
            { text: 'Kanbanボード', link: '/ja/samples/kanban' },
            { text: 'ブログシステム', link: '/ja/samples/blog' },
            { text: 'Eコマース', link: '/ja/samples/ecommerce' },
            { text: 'エラーパターン', link: '/ja/samples/error-patterns' }
          ]
        }
      ],
      '/ja/development/': [
        {
          text: 'Development',
          items: [
            { text: 'アーキテクチャ', link: '/ja/development/architecture' },
            { text: 'ビルド方法', link: '/ja/development/building' },
            { text: 'コントリビューション', link: '/ja/development/contributing' },
            { text: 'パーサー内部', link: '/ja/development/parser' },
            { text: 'エグゼキューター内部', link: '/ja/development/executor' },
            { text: 'エラー分類', link: '/ja/development/error-classification' },
            { text: 'データベース追加', link: '/ja/development/adding-database' }
          ]
        }
      ]
    },

    editLink: {
      pattern: 'https://github.com/shibukawa/snapsql/edit/main/docs/pages/:path',
      text: 'このページを編集'
    },

    outline: {
      label: '目次',
      level: [2, 3]
    },

    docFooter: {
      prev: '前へ',
      next: '次へ'
    },

    lastUpdated: {
      text: '最終更新',
      formatOptions: {
        dateStyle: 'short',
        timeStyle: 'short'
      }
    },

    returnToTopLabel: 'トップへ戻る',
    sidebarMenuLabel: 'メニュー',
    darkModeSwitchLabel: 'ダークモード',
    lightModeSwitchTitle: 'ライトモードに切り替え',
    darkModeSwitchTitle: 'ダークモードに切り替え'
  }
})
