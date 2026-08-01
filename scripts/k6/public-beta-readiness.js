import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate } from 'k6/metrics'

const baseUrl = (__ENV.BASE_URL || 'http://127.0.0.1:1309').replace(/\/+$/, '')
const smoke = __ENV.K6_SMOKE === '1'
const writeSmoke = __ENV.K6_WRITE_SMOKE === '1'

export const errorRate = new Rate('scf_errors')

const smokeThresholds = {
  scf_errors: ['rate<0.01'],
  http_req_failed: ['rate<0.01'],
}

const readinessThresholds = {
  scf_errors: ['rate<0.001'],
  http_req_failed: ['rate<0.001'],
  'http_req_duration{kind:read}': ['p(95)<300'],
  ...(writeSmoke ? { 'http_req_duration{kind:write}': ['p(95)<500'] } : {}),
}

export const options = smoke
  ? {
      scenarios: {
        read_smoke: {
          executor: 'shared-iterations',
          vus: 1,
          iterations: 1,
          exec: 'readJourney',
        },
        ...(writeSmoke
          ? {
              write_smoke: {
                executor: 'shared-iterations',
                vus: 1,
                iterations: 1,
                exec: 'writeJourney',
              },
            }
          : {}),
      },
      thresholds: smokeThresholds,
    }
  : {
      scenarios: {
        public_beta_readiness: {
          executor: 'constant-vus',
          vus: 50,
          duration: __ENV.PUBLIC_BETA_TEST_DURATION || '30m',
          exec: 'readJourney',
        },
        ...(writeSmoke
          ? {
              write_smoke: {
                executor: 'shared-iterations',
                vus: 1,
                iterations: 1,
                exec: 'writeJourney',
              },
            }
          : {}),
      },
      thresholds: readinessThresholds,
    }

const readEndpoints = [
  ['/healthz', 'health'],
  ['/api/v1/posts?sort=latest&limit=12', 'posts'],
  ['/api/v1/topics', 'topics'],
  ['/api/v1/insights', 'insights'],
  ['/api/v1/provinces', 'provinces'],
  ['/api/v1/policies', 'policies'],
  ['/api/v1/requirements', 'requirements'],
]

export default function () {
  readJourney()
}

export function readJourney() {
  const pacingSeconds = Number(__ENV.PUBLIC_BETA_PACING_SECONDS || '3.33')
  if (!smoke && __ITER === 0) sleep(Math.random() * pacingSeconds)
  const [path, name] = readEndpoints[Math.floor(Math.random() * readEndpoints.length)]
  const response = http.get(`${baseUrl}${path}`, { tags: { kind: 'read', endpoint: name } })
  const ok = check(response, {
    [`${name} returns 2xx`]: (res) => res.status >= 200 && res.status < 300,
    [`${name} has bounded body`]: (res) => res.body.length < 2_000_000,
  })
  errorRate.add(!ok)
  sleep(smoke ? 0 : pacingSeconds)
}

export function writeJourney() {
  const stamp = `${Date.now()}-${__VU}-${__ITER}`
  const email = `k6-write-${stamp}@example.com`
  const password = 'k6-password-123'
  const nickname = `k6写入${String(Date.now()).slice(-6)}`

  const codeResponse = postJSON('/api/v1/auth/email-verification-code', { email }, undefined, { endpoint: 'email_code' })
  const debugCode = jsonData(codeResponse)?.debugCode
  const codeOK = check(codeResponse, {
    'email code returns debug code in isolated write smoke': (res) => res.status >= 200 && res.status < 300 && Boolean(jsonData(res)?.debugCode),
  })
  errorRate.add(!codeOK)
  if (!debugCode) return

  const registerResponse = postJSON('/api/v1/auth/register', {
    email,
    password,
    verificationCode: debugCode,
    nickname,
    role: 'student',
    province: '广东',
    grade: '高一',
  }, undefined, { endpoint: 'register' })
  const registerOK = check(registerResponse, {
    'register returns user': (res) => res.status >= 200 && res.status < 300 && Boolean(jsonData(res)?.user?.id),
  })
  errorRate.add(!registerOK)
  if (!registerOK) return

  const csrf = csrfToken(registerResponse)
  const postResponse = postJSON('/api/v1/posts', {
    title: `k6 公测写入 smoke ${stamp}`,
    content: '这是一条 k6 写入 smoke 内容，用于覆盖注册、发帖、评论、点赞和收藏链路。',
    imageUrls: [],
    tags: ['k6-smoke'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'question',
    grade: '高一',
    province: '广东',
  }, csrf, { endpoint: 'create_post' })
  const postID = jsonData(postResponse)?.id
  const postOK = check(postResponse, {
    'create post returns id': (res) => res.status >= 200 && res.status < 300 && Boolean(jsonData(res)?.id),
  })
  errorRate.add(!postOK)
  if (!postID) return

  const commentResponse = postJSON(`/api/v1/posts/${postID}/comments`, {
    content: 'k6 写入 smoke 评论。',
  }, csrf, { endpoint: 'create_comment' })
  errorRate.add(!check(commentResponse, {
    'create comment returns id': (res) => res.status >= 200 && res.status < 300 && Boolean(jsonData(res)?.id),
  }))

  const likeResponse = postJSON(`/api/v1/posts/${postID}/like`, undefined, csrf, { endpoint: 'like_post' })
  errorRate.add(!check(likeResponse, {
    'like returns active true': (res) => res.status >= 200 && res.status < 300 && jsonData(res)?.active === true,
  }))

  const favoriteResponse = postJSON(`/api/v1/posts/${postID}/favorite`, undefined, csrf, { endpoint: 'favorite_post' })
  errorRate.add(!check(favoriteResponse, {
    'favorite returns active true': (res) => res.status >= 200 && res.status < 300 && jsonData(res)?.active === true,
  }))
}

function postJSON(path, payload, csrf, extraTags = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (csrf) headers['X-CSRF-Token'] = csrf
  return http.post(`${baseUrl}${path}`, payload === undefined ? null : JSON.stringify(payload), {
    headers,
    tags: { kind: 'write', ...extraTags },
  })
}

function jsonData(response) {
  try {
    return JSON.parse(response.body).data
  } catch {
    return undefined
  }
}

function csrfToken(response) {
  const responseCookie = response.cookies?.scf_csrf?.[0]?.value
  if (responseCookie) return responseCookie
  const jarCookie = http.cookieJar().cookiesForURL(baseUrl).scf_csrf
  if (Array.isArray(jarCookie)) return jarCookie[0]?.value || jarCookie[0] || ''
  return jarCookie?.value || jarCookie || ''
}
