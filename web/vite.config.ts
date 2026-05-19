import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@wails': path.resolve(__dirname, './wailsjs'),
    },
  },
  server: {
    port: 34115, // Wails 开发模式默认前端端口
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
