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
      // Runtime fix for GH-Pages: some pages request the logo images under
      // the localized path (e.g. /snapsql/ja/snapsql.small.png). Ensure the
      // header favicon/logo and hero background always point to the absolute
      // repository path without the locale prefix so GH-Pages serves them.
      try {
        const ensureAbsoluteLogos = () => {
          try {
            const p = window.location.pathname || ''
            if (!p.includes('/ja/')) return

            // header logo
            const logoSel = '.VPNavBarTitle .title img.VPImage.logo'
            const el = document.querySelector<HTMLImageElement>(logoSel)
            if (el) el.src = '/snapsql/snapsql.small.png'

            // favicon(s)
            document.querySelectorAll('link[rel~="icon"]').forEach((lnk) => {
              ;(lnk as HTMLLinkElement).href = '/snapsql/snapsql.small.png'
            })

            // ensure hero pseudo-element background is absolute by injecting
            // an overriding style block (safe and idempotent)
            const id = 'snapsql-hero-fix'
            if (!document.getElementById(id)) {
              const s = document.createElement('style')
              s.id = id
              s.textContent = `.VPHero .main::before{background-image:url('/snapsql/snapsql.logo.png') !important}`
              document.head.appendChild(s)
            }
          } catch (e) {
            // noop
          }
        }

        // run now
        ensureAbsoluteLogos()
        // re-run on history navigation and DOM mutations
        window.addEventListener('popstate', ensureAbsoluteLogos)
        const mo = new MutationObserver(ensureAbsoluteLogos)
        mo.observe(document.body, { childList: true, subtree: true })
      } catch (e) {
        // noop
      }
    }
    enhanceAppWithTabs(app);
  }
} satisfies Theme
