import { defineConfig, devices } from '@playwright/test'

const frontendPort = Number.parseInt(process.env.E2E_REAL_FRONTEND_PORT || '4210', 10)
const backendPort = Number.parseInt(process.env.E2E_REAL_BACKEND_PORT || '1319', 10)
const frontendOrigin = `http://127.0.0.1:${frontendPort}`
const backendOrigin = `http://127.0.0.1:${backendPort}`

export default defineConfig({
  testDir: './e2e-real',
  timeout: 45_000,
  expect: { timeout: 8_000 },
  reporter: [['list']],
  use: {
    baseURL: frontendOrigin,
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: [
        'mkdir -p ../.tmp/e2e-real ../.tmp/e2e-real-uploads',
        'rm -f ../.tmp/e2e-real/soulcourse.db ../.tmp/e2e-real/soulcourse.db-shm ../.tmp/e2e-real/soulcourse.db-wal',
        `cd ../backend && APP_ENV=local HTTP_HOST=127.0.0.1 HTTP_PORT=${backendPort} SQLITE_PATH=../.tmp/e2e-real/soulcourse.db MEDIA_UPLOAD_DIR=../.tmp/e2e-real-uploads CORS_ALLOWED_ORIGINS=${frontendOrigin} go run .`,
      ].join(' && '),
      url: `${backendOrigin}/readyz`,
      reuseExistingServer: false,
      timeout: 30_000,
    },
    {
      command: `VITE_API_BASE_URL=${backendOrigin}/api/v1 pnpm dev --host 127.0.0.1 --port ${frontendPort}`,
      url: frontendOrigin,
      reuseExistingServer: false,
      timeout: 30_000,
    },
  ],
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
})
