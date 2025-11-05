import { withMermaid } from 'vitepress-plugin-mermaid';
import { tabsMarkdownPlugin } from 'vitepress-plugin-tabs'
import { ja } from './config/ja'

// https://vitepress.dev/reference/site-config
const siteBase = ((process.env as any).VITEPRESS_BASE) || '/'

export default withMermaid({
  // Allow overriding the site `base` via environment variable so local dev can
  // use '/' while CI/GitHub Pages build can set '/snapsql/'.
  base: siteBase,

  title: "SnapSQL",
  description: "Markdownでデータベーステストを書く",
  
  // 多言語対応
  locales: {
    root: {
      label: '日本語',
      lang: 'ja',
      ...ja
    }
    // 将来的に英語版を追加
    // en: {
    //   label: 'English',
    //   lang: 'en',
    //   ...en
    // }
  },

  // テーマ設定
  themeConfig: {
  // small logo used in the header / navbar (relative to base)
  logo: 'snapsql.small.png',
    siteTitle: 'SnapSQL',
    
    // 検索設定
    search: {
      provider: 'local',
      options: {
        locales: {
          ja: {
            translations: {
              button: {
                buttonText: '検索',
                buttonAriaLabel: 'ドキュメントを検索'
              },
              modal: {
                noResultsText: '結果が見つかりません',
                resetButtonTitle: 'リセット',
                footer: {
                  selectText: '選択',
                  navigateText: '移動',
                  closeText: '閉じる'
                }
              }
            }
          }
        }
      }
    },

    // ソーシャルリンク
    socialLinks: [
      { icon: 'github', link: 'https://github.com/shibukawa/snapsql' }
    ],

    // フッター
    footer: {
      message: 'Released under the Apache-2.0 License.',
      copyright: 'Copyright © 2024 Yoshiki Shibukawa'
    }
  },

  // ビルド設定
  srcDir: '.',
  outDir: '.vitepress/dist',
  cacheDir: '.vitepress/cache',
  
  // Markdown設定
  markdown: {
    lineNumbers: true,
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    config(md) {
      md.use(tabsMarkdownPlugin)
    }
  },

  // リンクチェック設定
  // ローカルの開発用パスのみ無視していたデフォルトに戻します。外部のドキュメントは
  // サイト外にある場合は GitHub 上の適切な URL に差し替えてください。
  ignoreDeadLinks: ['http://localhost:5173/index'],

  // Head設定
  head: [
  // use the small PNG as favicon / header icon (relative to base)
  ['link', { rel: 'icon', type: 'image/png', href: 'snapsql.small.png' }],
  // inject hero pseudo-element background (relative url; VitePress will apply base)
  ['style', {}, `.VPHero .main::before{background-image:url('snapsql.logo.png')}`],
    ['meta', { name: 'theme-color', content: '#646cff' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:locale', content: 'ja_JP' }],
    ['meta', { property: 'og:title', content: 'SnapSQL | Markdownでデータベーステストを書く' }],
    ['meta', { property: 'og:site_name', content: 'SnapSQL' }],
  ],
  mermaid: { theme: 'forest' },
  mermaidPlugin: { class: 'mermaid my-class' }
})
