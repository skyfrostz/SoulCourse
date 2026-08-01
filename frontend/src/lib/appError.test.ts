import { afterEach, describe, expect, it } from 'vitest'
import { appError, clearAppError, isChunkLoadError, showAppError } from './appError'

describe('app error boundary state', () => {
  afterEach(() => clearAppError())

  it('recognizes lazy route chunk failures', () => {
    expect(isChunkLoadError(new Error('Failed to fetch dynamically imported module'))).toBe(true)
    expect(isChunkLoadError(new Error('ordinary component failure'))).toBe(false)
  })

  it('shows a recoverable chunk load message', () => {
    showAppError(new Error('Loading chunk 12 failed'), 'runtime')

    expect(appError.value).toMatchObject({
      kind: 'chunk',
      title: '页面资源加载失败',
    })
  })
})
