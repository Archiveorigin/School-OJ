import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import ElementPlus from 'unplugin-element-plus/vite'
import { pdfAssetsPlugin } from './scripts/pdf-assets-plugin'

export default defineConfig({
  plugins: [
    pdfAssetsPlugin(),
    vue(),
    Components({ dirs: [], dts: false, resolvers: [ElementPlusResolver()] }),
    ElementPlus()
  ],
  build: {
    manifest: true,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          // Shared helpers must not land in a lazy editor chunk: that would make
          // the entrypoint import Monaco just to call Vite's preload helper.
          if (id.includes('vite/preload-helper') || id.includes('commonjsHelpers')) return 'runtime'
          const normalized = id.replace(/\\/g, '/')
          if (!normalized.includes('/node_modules/')) return undefined
          if (normalized.includes('/node_modules/monaco-editor/')) return 'monaco'
          if (normalized.includes('/node_modules/katex/')) return 'katex'
          if (
            normalized.includes('/node_modules/vue/') ||
            normalized.includes('/node_modules/@vue/') ||
            normalized.includes('/node_modules/vue-router/') ||
            normalized.includes('/node_modules/pinia/')
          ) {
            return 'vue-vendor'
          }
          return undefined
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080'
    }
  },
  test: {
    environment: 'jsdom',
    server: { deps: { inline: ['element-plus'] } }
  }
})
