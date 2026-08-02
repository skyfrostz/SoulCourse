import path from 'node:path'
import fs from 'node:fs'
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
  const devServerPort = Number.parseInt(env.FRONTEND_HOST_PORT || '5712', 10)
  const base = normalizeBasePath(env.VITE_APP_BASE_PATH || env.APP_BASE_PATH)
  const isAndroidTarget = env.VITE_APP_TARGET === 'android'
  const apiProxyPrefix = `${base}api`
  const uploadsProxyPrefix = `${base}uploads`

  return {
    base,
    build: {
      outDir: isAndroidTarget ? 'dist-android' : 'dist',
    },
    envDir: path.resolve(process.cwd(), '..'),
    plugins: [
      vue(),
      ...(isAndroidTarget ? [{
        name: 'exclude-admin-static-assets-from-android',
        closeBundle() {
          fs.rmSync(path.resolve('dist-android/admin'), { recursive: true, force: true })
        },
      }] : []),
    ],
    server: {
      host: '0.0.0.0',
      port: Number.isNaN(devServerPort) ? 5712 : devServerPort,
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
