import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import llmstxt from 'vitepress-plugin-llms'
import { SearchPlugin } from "vitepress-plugin-search";

const commitHash = process.env.VITE_COMMIT_HASH || ''

// https://vitepress.dev/reference/site-config
export default withMermaid(defineConfig({
  title: "Virga",
  description: "Lightweight C2 Framework with Embedded AI",
  lang: 'en-US',
  base: process.env.VITE_BASE || '/virga/',

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }],
    ['meta', { name: 'theme-color', content: '#3c8772' }],
    // Google Analytics 4
    ...(process.env.VITE_GA_ID ? [
      [
        'script',
        {
          async: true,
          src: `https://www.googletagmanager.com/gtag/js?id=${process.env.VITE_GA_ID}`
        }
      ],
      [
        'script',
        {},
        `window.dataLayer = window.dataLayer || [];
        function gtag(){dataLayer.push(arguments);}
        gtag('js', new Date());
        gtag('config', '${process.env.VITE_GA_ID}');`
      ]
    ] : [])
  ],

  themeConfig: {
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/introduction' },
      { text: 'Usage', link: '/usage/beacon-generation' },
      { text: 'Reference', link: '/reference/server-config-reference' },
      { text: 'GitHub', link: 'https://github.com/r74tech/virga' }
    ],

    sidebar: {
      '/guide/': {
        text: 'Guide',
        items: [
          {
            text: 'Getting Started',
            items: [
              { text: 'Introduction', link: '/guide/introduction' },
              { text: 'Installation', link: '/guide/installation' },
              { text: 'Quick Start', link: '/guide/quick-start' },
            ]
          },
          {
            text: 'Core Concepts',
            items: [
              { text: 'Listeners', link: '/guide/listeners' },
              { text: 'Sessions', link: '/guide/sessions' },
              { text: 'Security', link: '/guide/security' },
            ]
          }
        ]
      },
      '/usage/': {
        text: 'Usage',
        items: [
          {
            text: 'Beacon Management',
            items: [
              { text: 'Beacon Generation', link: '/usage/beacon-generation' },
              { text: 'Beacon Configuration', link: '/usage/beacon-configuration' },
            ]
          },
          {
            text: 'AI & Automation',
            items: [
              { text: 'AI Features', link: '/usage/ai-features' },
              { text: 'MCP Integration', link: '/usage/mcp-integration' },
            ]
          }
        ]
      },
      '/reference/': {
        text: 'Reference',
        items: [
          { text: 'Server Config', link: '/reference/server-config-reference' },
          { text: 'Beacon Config', link: '/reference/beacon-config-reference' },
          { text: 'CLI Commands', link: '/reference/cli-command-reference' },
          { text: 'API Reference', link: '/reference/api-reference' },
          { text: 'Troubleshooting', link: '/reference/troubleshooting' },
        ]
      }
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/r74tech/virga' }
    ],


    footer: {
      message: commitHash
        ? `For authorized security testing only • Built from <a href="https://github.com/r74tech/virga/tree/${commitHash}" target="_blank" rel="noopener">${commitHash.substring(0, 7)}</a>`
        : 'For authorized security testing only',
      copyright: 'Copyright © 2025 r74tech',
    }
  },

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    lineNumbers: true
  },
  vite: {
    plugins: [
      llmstxt(),
      SearchPlugin({
        tokenize: "tolerant",
        previewLength: 62,
        minLength: 2,
        buttonLabel: "Search",
        placeholder: "Search docs",
        buttonText: "Search",
        ignore: [
          '.vitepress/**',
          'node_modules/**',
          'dist/**',
          '**/*.cast',
          '**/assets/**',
          '**/.vitepress/dist/**'
        ]
      })
    ],
    ssr: {
      noExternal: [
        "@nolebase/ui-asciinema",
      ],
    },
    optimizeDeps: {
      include: ["dayjs", "@braintree/sanitize-url", "debug", "cytoscape-cose-bilkent", "cytoscape"]
    }
  },

  // Mermaid configuration
  mermaid: {
    theme: 'dark'
  }
}))