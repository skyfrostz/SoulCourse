import { describe, expect, it } from 'vitest'
import { ApiClientError, normalizeApiError } from './api-contract'

describe('API contract errors', () => {
  it('preserves structured error details and request id', () => {
    const error = normalizeApiError({
      response: {
        status: 422,
        headers: { 'x-request-id': 'req-header' },
        data: {
          error: {
            code: 'invalid_payload',
            message: '字段校验失败',
            fieldErrors: { email: '邮箱格式不正确' },
            requestId: 'req-body',
            retryAfterSeconds: 45,
          },
        },
      },
      isAxiosError: true,
    })

    expect(error).toBeInstanceOf(ApiClientError)
    expect(error.code).toBe('invalid_payload')
    expect(error.status).toBe(422)
    expect(error.requestId).toBe('req-body')
    expect(error.fieldErrors).toEqual({ email: '邮箱格式不正确' })
    expect(error.details).toEqual({ retryAfterSeconds: 45 })
  })

  it('falls back to the response request id for malformed error payloads', () => {
    const error = normalizeApiError({
      response: { status: 503, headers: { 'x-request-id': 'req-fallback' } },
      isAxiosError: true,
      message: '服务不可用',
    })

    expect(error.status).toBe(503)
    expect(error.requestId).toBe('req-fallback')
  })
})
