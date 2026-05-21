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
      // react-router v7 的 package.json exports 全部指向 development build，
      // 强制指向 production 构建以减小 bundle 体积。
      // 注意：更具体的 alias 必须放在前面，否则 'react-router' 会优先匹配 'react-router/dom'
      'react-router/dom': path.resolve(__dirname, './node_modules/react-router/dist/production/dom-export.mjs'),
      'react-router': path.resolve(__dirname, './node_modules/react-router/dist/production/index.mjs'),
    },
  },
  server: {
    port: 34115, // Wails 开发模式默认前端端口
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          // 框架核心
          vendor: ['react', 'react-dom', 'react-router-dom'],
          // Markdown 渲染引擎（仅消息气泡使用）
          markdown: ['react-markdown', 'remark-gfm', 'prismjs'],
          // 表单校验（仅设置页面使用）
          forms: ['react-hook-form', 'zod', '@hookform/resolvers'],
          // UI 工具库
          ui: ['lucide-react', 'tailwind-merge', 'zustand'],
        },
      },
    },
  },
})
