// vite.config.ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path' // 导入 Node.js 的 path 模块

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  base: '/',
  // 添加 resolve.alias 配置
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // 你的Go后端地址
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/api/, '') // 如果后端没有/api前缀
      },
      '/ws': {
        target: 'ws://localhost:8080', // 你的Go后端地址
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/ws/, ''), // 如果后端没有/ws前缀
      },
      '/static': {
        target: 'http://localhost:8080', // 你的Go后端地址
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/static/, ''), // 如果后端没有/static前缀
      }
    }
  }
})
