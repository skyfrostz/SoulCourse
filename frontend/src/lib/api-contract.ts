import axios from 'axios'
import type {
  Error as OpenAPIErrorPayload,
  ErrorEnvelope as OpenAPIErrorEnvelope,
  Meta as OpenAPIMeta,
} from './generated/openapi-types'

export type ApiMeta = OpenAPIMeta

export interface ApiEnvelope<T> {
  data: T
  meta?: ApiMeta
}

export type ApiErrorPayload = OpenAPIErrorPayload
export type ApiErrorEnvelope = OpenAPIErrorEnvelope

export class ApiClientError extends Error {
  status?: number
  code?: string
  requestId?: string
  fieldErrors?: Record<string, string>
  details?: Record<string, unknown>

  constructor(message: string, options: { status?: number; code?: string; requestId?: string; fieldErrors?: Record<string, string>; details?: Record<string, unknown> } = {}) {
    super(message)
    this.name = 'ApiClientError'
    this.status = options.status
    this.code = options.code
    this.requestId = options.requestId
    this.fieldErrors = options.fieldErrors
    this.details = options.details
  }
}

export function isApiErrorEnvelope(value: unknown): value is ApiErrorEnvelope {
  return Boolean(
    value &&
      typeof value === 'object' &&
      'error' in value &&
      value.error &&
      typeof value.error === 'object' &&
      'message' in value.error &&
      'code' in value.error,
  )
}

export function normalizeApiError(error: unknown): ApiClientError {
  if (axios.isAxiosError(error)) {
    const requestIdHeader = error.response?.headers?.['x-request-id']
    const payload = error.response?.data
    if (isApiErrorEnvelope(payload)) {
      const details = { ...(payload.error as unknown as Record<string, unknown>) }
      delete details.code
      delete details.message
      delete details.requestId
      delete details.fieldErrors
      return new ApiClientError(payload.error.message, {
        status: error.response?.status,
        code: payload.error.code,
        requestId: payload.error.requestId || requestIdHeader,
        fieldErrors: payload.error.fieldErrors,
        details,
      })
    }
    return new ApiClientError(error.message || '请求失败', {
      status: error.response?.status,
      requestId: requestIdHeader,
    })
  }
  if (error instanceof Error) return new ApiClientError(error.message)
  return new ApiClientError('请求失败')
}
