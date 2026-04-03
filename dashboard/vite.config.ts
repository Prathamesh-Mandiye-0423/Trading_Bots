import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      // 1. Standard API (Go Backend)
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      // 2. WebSockets (Go Backend)
      // Switch this to use the proxy port in your hook!
      '/ws': {
        target: 'http://127.0.0.1:8080', // Vite handles the upgrade
        ws: true
      },
      // 3. ML Service (Python Backend)
      '/ml-api': {
        target: 'http://127.0.0.1:8001', // Updated from 8081 to 8001
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/ml-api/, '') 
        // Note: If your Python routes already start with /api/v1, 
        // just replace /ml-api with an empty string.
      }
    }
  }
})