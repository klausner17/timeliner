import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  // Caminho onde o app e servido. Em producao fica sob klausner.site/app/timeliner/
  // (VITE_BASE no build da imagem); o router usa o mesmo valor via BASE_URL.
  base: process.env.VITE_BASE ?? '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // Em desenvolvimento, /api vai para o backend Go (cd backend && go run ./cmd/api).
    proxy: { '/api': 'http://localhost:8080' },
  },
  test: {
    environment: 'jsdom',
  },
})
