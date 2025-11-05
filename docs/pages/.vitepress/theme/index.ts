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
      // prefer site base so redirects work when hosted under a repo path (GitHub Pages)
  const base = ((siteData as any)?.value?.base) || '/'
      const rootPaths = [base, base + 'index.html', '/']
      router.onBeforeRouteChange = (to) => {
        // If navigating to the site's root, redirect to localized /ja/ under base
        if (rootPaths.includes(to)) {
          window.location.href = `${base}ja/`
          return false
        }
      }

      // initial load redirect
      if (rootPaths.includes(window.location.pathname)) {
        window.location.href = `${base}ja/`
      }
    }
    enhanceAppWithTabs(app);
  }
} satisfies Theme
