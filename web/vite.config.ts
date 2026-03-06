import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8096',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:8096',
        ws: true
      }
    }
  }
})
