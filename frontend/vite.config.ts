import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const devProxyTarget = env.VITE_DEV_PROXY_TARGET || 'http://localhost:8081'

  return {
    plugins: [vue()],
    server: {
      proxy: {
        '/api': {
          target: devProxyTarget,
          changeOrigin: true,
        },
        '/uploads': {
          target: devProxyTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
