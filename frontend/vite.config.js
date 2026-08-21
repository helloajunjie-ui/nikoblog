import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 开发环境代理到后端 8080，生产环境由 Go embed 托管静态资源
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    // 输出到 web/dist，供 Go embed 打包进二进制
    outDir: '../web/dist',
    assetsDir: 'assets',
    emptyOutDir: true
  }
})
