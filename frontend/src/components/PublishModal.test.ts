import { VueQueryPlugin } from '@tanstack/vue-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useForumStore } from '../stores/forum'
import PublishModal from './PublishModal.vue'

const uploadImage = vi.fn()
const createPost = vi.fn()

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    createPost: (...args: unknown[]) => createPost(...args),
    fetchTaxonomy: vi.fn().mockResolvedValue({ topicTags: [] }),
    uploadImage: (...args: unknown[]) => uploadImage(...args),
  }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

class LoadedImage {
  naturalWidth = 1200
  naturalHeight = 800
  onload: null | (() => void) = null
  onerror: null | (() => void) = null

  set src(_value: string) {
    queueMicrotask(() => this.onload?.())
  }
}

describe('PublishModal image uploads', () => {
  beforeEach(() => {
    uploadImage.mockReset()
    createPost.mockReset()
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.stubGlobal('Image', LoadedImage)
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn((file: File) => `blob:${file.name}`),
      revokeObjectURL: vi.fn(),
    })
  })

  afterEach(() => cleanup())

  it('keeps successful images and retries only failed images', async () => {
    uploadImage
      .mockResolvedValueOnce({ url: '/uploads/success.jpg' })
      .mockRejectedValueOnce(new Error('upload failed'))

    const { container } = render(PublishModal, {
      global: { plugins: [createPinia(), VueQueryPlugin] },
    })
    const input = container.querySelector('input[type="file"]') as HTMLInputElement
    const successFile = new File(['success'], 'success.jpg', { type: 'image/jpeg', lastModified: 1 })
    const failedFile = new File(['failed'], 'failed.jpg', { type: 'image/jpeg', lastModified: 2 })

    await fireEvent.change(input, { target: { files: [successFile, failedFile] } })

    expect(await screen.findByText('1 张图片待重试')).toBeInTheDocument()
    expect(screen.getByText(/1 张图片上传失败，已保留上传成功的图片/)).toBeInTheDocument()
    expect(screen.getAllByAltText('待发布图片预览')).toHaveLength(1)
    expect(uploadImage).toHaveBeenCalledTimes(2)

    uploadImage.mockResolvedValueOnce({ url: '/uploads/retried.jpg' })
    await fireEvent.click(screen.getByRole('button', { name: '重试失败图片' }))

    await waitFor(() => expect(screen.getAllByAltText('待发布图片预览')).toHaveLength(2))
    expect(screen.queryByText('1 张图片待重试')).not.toBeInTheDocument()
    expect(uploadImage).toHaveBeenCalledTimes(3)
    expect(uploadImage.mock.calls.filter(([file]) => file === successFile)).toHaveLength(1)
  })

  it('does not upload an already successful image when it is selected again', async () => {
    uploadImage.mockResolvedValue({ url: '/uploads/success.jpg' })
    const { container } = render(PublishModal, {
      global: { plugins: [createPinia(), VueQueryPlugin] },
    })
    const input = container.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['success'], 'success.jpg', { type: 'image/jpeg', lastModified: 1 })

    await fireEvent.change(input, { target: { files: [file] } })
    await waitFor(() => expect(screen.getAllByAltText('待发布图片预览')).toHaveLength(1))
    await fireEvent.change(input, { target: { files: [file] } })

    expect(uploadImage).toHaveBeenCalledTimes(1)
    expect(screen.getAllByAltText('待发布图片预览')).toHaveLength(1)
  })

  it('does not publish while failed images are still queued', async () => {
    uploadImage.mockRejectedValueOnce(new Error('complete failed'))
    const pinia = createPinia()
    setActivePinia(pinia)
    useForumStore().session = {
      user: { id: 7, publicId: 'u_7', email: 'student@example.com', nickname: '广东学生', role: 'student', province: '广东', grade: '高一', createdAt: '2026-07-31T00:00:00Z' },
      expiresAt: '2099-01-01T00:00:00Z',
    }
    const { container } = render(PublishModal, {
      global: { plugins: [pinia, VueQueryPlugin] },
    })
    const input = container.querySelector('input[type="file"]') as HTMLInputElement
    await fireEvent.change(input, { target: { files: [new File(['failed'], 'failed.jpg', { type: 'image/jpeg' })] } })
    await screen.findByText('1 张图片待重试')

    await fireEvent.update(screen.getByLabelText('标题'), '这是一个完整标题')
    await fireEvent.update(screen.getByLabelText('正文'), '这是足够长的帖子正文内容。')
    await fireEvent.click(screen.getByRole('button', { name: '发布' }))

    expect(createPost).not.toHaveBeenCalled()
    expect(screen.getByText('仍有图片上传失败，请先重试，或移除失败图片后再发布。')).toBeInTheDocument()
  })

  it('lets the user discard a failed image and continue', async () => {
    uploadImage.mockRejectedValueOnce(new Error('upload timeout'))
    const { container } = render(PublishModal, {
      global: { plugins: [createPinia(), VueQueryPlugin] },
    })
    const input = container.querySelector('input[type="file"]') as HTMLInputElement
    await fireEvent.change(input, { target: { files: [new File(['failed'], 'failed.jpg', { type: 'image/jpeg' })] } })

    await fireEvent.click(await screen.findByRole('button', { name: '移除失败图片' }))

    expect(screen.queryByText('1 张图片待重试')).not.toBeInTheDocument()
    expect(screen.queryByText(/图片上传失败/)).not.toBeInTheDocument()
  })
})
