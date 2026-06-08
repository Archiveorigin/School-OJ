import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('vite/preload-helper')) return 'vendor'
          const normalized = id.replace(/\\/g, '/')
          if (!normalized.includes('/node_modules/')) return undefined
          if (normalized.includes('/node_modules/monaco-editor/')) return 'monaco'
          if (
            normalized.includes('/node_modules/element-plus/') ||
            normalized.includes('/node_modules/@element-plus/')
          ) {
            return 'element-plus'
          }
          if (normalized.includes('/node_modules/katex/')) return 'katex'
          if (
            normalized.includes('/node_modules/vue/') ||
            normalized.includes('/node_modules/@vue/') ||
            normalized.includes('/node_modules/vue-router/') ||
            normalized.includes('/node_modules/pinia/')
          ) {
            return 'vue-vendor'
          }
          return 'vendor'
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
    environment: 'jsdom'
  }
})
