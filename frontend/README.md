# 选科π Frontend

Vue 3 + TypeScript + Vite frontend for the public beta application.

## Verification

```bash
pnpm lint
pnpm test:unit
pnpm openapi:check
pnpm build
pnpm test:e2e
pnpm test:e2e:real-backend
```

`test:e2e` runs the fast mocked browser flows. `test:e2e:real-backend` starts a temporary Go backend with SQLite and runs a real browser registration-to-post smoke through `VITE_API_BASE_URL`; it uses the local debug verification code and does not touch production email.
