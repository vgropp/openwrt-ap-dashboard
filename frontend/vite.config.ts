import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  // API_ prefix keeps these values out of the client bundle (only VITE_ is exposed).
  const env = loadEnv(mode, '.', 'API_')

  return {
    plugins: [react()],
    base: "./",
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: env.API_PROXY_TARGET || 'http://localhost:8189',
          changeOrigin: true,
        },
      },
    },
  }
})
