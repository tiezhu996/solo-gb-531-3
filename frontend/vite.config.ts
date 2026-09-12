import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 18531,
    proxy: { '/api': 'http://127.0.0.1:19531' },
  },
  build: { chunkSizeWarningLimit: 600 },
  test: { environment: 'jsdom' },
})
