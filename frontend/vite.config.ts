import path from 'node:path'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

function normalizeBasePath(value: string | undefined) {
  const trimmed = (value || '/').trim()
  if (!trimmed || trimmed === '/') return '/'
  return `/${trimmed.replace(/^\/+|\/+$/g, '')}/`
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const workspaceEnv = loadEnv(mode, path.resolve(process.cwd(), '..'), '')
  const localEnv = loadEnv(mode, process.cwd(), '')
  const env = { ...workspaceEnv, ...localEnv }
  const devProxyTarget = env.VITE_DEV_PROXY_TARGET || 'http://localhost:1309'
  const base = normalizeBasePath(env.VITE_APP_BASE_PATH || env.APP_BASE_PATH)
  const apiProxyPrefix = `${base}api`
  const uploadsProxyPrefix = `${base}uploads`

  return {
    base,
    envDir: path.resolve(process.cwd(), '..'),
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      port: 5712,
      proxy: {
        [apiProxyPrefix]: {
          target: devProxyTarget,
          changeOrigin: true,
        },
        [uploadsProxyPrefix]: {
          target: devProxyTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
