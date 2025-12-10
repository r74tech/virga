// https://vitepress.dev/guide/custom-theme
import { h, nextTick, watch } from 'vue'
import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { useRoute } from 'vitepress'
import './style.css'
import 'svg-toolbelt/dist/svg-toolbelt.css'
import { NuAsciinemaPlayer } from "@nolebase/ui-asciinema";
import "asciinema-player/dist/bundle/asciinema-player.css";


export default {
  extends: DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      // https://vitepress.dev/guide/extending-default-theme#layout-slots
    })
  },
  enhanceApp({ app }) {
    const initializeMermaidZoom = async () => {
      if (typeof window === 'undefined') return

      // Dynamically import svg-toolbelt only in browser
      const { initializeSvgToolbelt } = await import('svg-toolbelt')

      await nextTick()
      // Wait a bit more for Mermaid to render
      setTimeout(() => {
        const containers = document.querySelectorAll('.mermaid')
        if (containers.length > 0) {
          initializeSvgToolbelt('.mermaid')
        }
      }, 500)
    }

    app.component("NuAsciinemaPlayer", NuAsciinemaPlayer);

    // Initialize on mount using app mixin
    app.mixin({
      mounted() {
        initializeMermaidZoom()
      },
      setup() {
        const route = useRoute()
        // Re-initialize on route change
        watch(() => route.path, () => {
          initializeMermaidZoom()
        })
      }
    })
  }
} satisfies Theme