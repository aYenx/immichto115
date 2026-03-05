import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8096',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8096',
        ws: true,
      },
    },
  },
  build: {
    outDir: '../cmd/server/dist',
    emptyOutDir: true,
  },
})
