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
        manualChunks: (id: string) => {
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom') || id.includes('node_modules/react-router-dom')) {
            return 'vendor';
          }
          if (id.includes('node_modules/react-markdown') || id.includes('node_modules/remark-gfm') || id.includes('node_modules/prismjs')) {
            return 'markdown';
          }
          if (id.includes('node_modules/react-hook-form') || id.includes('node_modules/zod') || id.includes('node_modules/@hookform/resolvers')) {
            return 'forms';
          }
          if (id.includes('node_modules/lucide-react') || id.includes('node_modules/tailwind-merge') || id.includes('node_modules/zustand')) {
            return 'ui';
          }
          return undefined;
        },
      },
    },
  },
})
