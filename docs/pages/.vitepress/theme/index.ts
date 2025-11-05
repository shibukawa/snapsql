// https://vitepress.dev/guide/custom-theme
import { h } from 'vue'
import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { enhanceAppWithTabs } from 'vitepress-plugin-tabs/client'
import './style.css'

export default {
  extends: DefaultTheme,
  Layout: () => {
    // restore default layout slots (no hero override) so page frontmatter hero is used
    return h(DefaultTheme.Layout, null, {
      // https://vitepress.dev/guide/extending-default-theme#layout-slots
    })
  },
  enhanceApp({ app, router, siteData }) {
    if (typeof window !== 'undefined') {
      router.onBeforeRouteChange = (to) => {
        // ルートパスの場合は /ja/ にリダイレクト
        if (to === '/' || to === '/index.html') {
          window.location.href = '/ja/'
          return false
        }
      }
      
      // 初回ロード時のリダイレクト
      if (window.location.pathname === '/' || window.location.pathname === '/index.html') {
        window.location.href = '/ja/'
      }
    }
    enhanceAppWithTabs(app);
  }
} satisfies Theme
