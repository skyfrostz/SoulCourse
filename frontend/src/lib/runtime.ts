const absoluteUrlPattern = /^(https?:|data:|blob:)/i

function normalizeBaseUrl(value: string | undefined) {
  const raw = (value || '/').trim()
  if (!raw || raw === '/') return '/'
  return `/${raw.replace(/^\/+|\/+$/g, '')}/`
}

export const appBaseUrl = normalizeBaseUrl(import.meta.env.BASE_URL)
export const appBasePath = appBaseUrl === '/' ? '' : appBaseUrl.slice(0, -1)

export function withAppBase(path: string) {
  if (!path || absoluteUrlPattern.test(path)) return path
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return appBasePath ? `${appBasePath}${normalizedPath}` : normalizedPath
}

export function buildAppUrl(path: string) {
  return new URL(withAppBase(path), window.location.origin).toString()
}

export function defaultApiBasePath() {
  return withAppBase('/api/v1').replace(/\/$/, '')
}

export function appAssetUrl(path: string) {
  return withAppBase(path)
}
