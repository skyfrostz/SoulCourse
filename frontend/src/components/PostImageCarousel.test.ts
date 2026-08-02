import { cleanup, fireEvent, render, screen } from '@testing-library/vue'
import { afterEach, describe, expect, it } from 'vitest'
import PostImageCarousel from './PostImageCarousel.vue'

const images = (count: number) => Array.from({ length: count }, (_, index) => `/image-${index + 1}.jpg`)

describe('PostImageCarousel', () => {
  afterEach(() => cleanup())

  it.each([1, 2, 5, 9])('supports %i images with stable count state', async (count) => {
    render(PostImageCarousel, { props: { images: images(count), title: '测试帖子' } })
    expect(screen.getByRole('group')).toHaveAttribute('aria-label', `第 1 张，共 ${count} 张`)
    if (count === 1) expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
    else expect(screen.getByRole('tablist')).toBeInTheDocument()

    if (count > 1) {
      await fireEvent.click(screen.getByRole('button', { name: '下一张图片' }))
      expect(screen.getByRole('group')).toHaveAttribute('aria-label', `第 2 张，共 ${count} 张`)
      await fireEvent.click(screen.getByRole('button', { name: '上一张图片' }))
      expect(screen.getByRole('group')).toHaveAttribute('aria-label', `第 1 张，共 ${count} 张`)
      await fireEvent.click(screen.getByRole('button', { name: '上一张图片' }))
      expect(screen.getByRole('group')).toHaveAttribute('aria-label', `第 ${count} 张，共 ${count} 张`)
    }
  })

  it('switches with dots, keyboard and touch gestures', async () => {
    render(PostImageCarousel, { props: { images: images(5), title: '测试帖子' } })
    await fireEvent.click(screen.getByRole('tab', { name: '查看第 4 张图片' }))
    expect(screen.getByRole('group')).toHaveAttribute('aria-label', '第 4 张，共 5 张')

    await fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(screen.getByRole('group')).toHaveAttribute('aria-label', '第 5 张，共 5 张')
    const stage = screen.getByRole('group')
    await fireEvent.touchStart(stage, { changedTouches: [{ clientX: 300 }] })
    await fireEvent.touchEnd(stage, { changedTouches: [{ clientX: 120 }] })
    expect(screen.getByRole('group')).toHaveAttribute('aria-label', '第 1 张，共 5 张')
  })

  it('opens preview, resets zoom, and closes with Escape', async () => {
    render(PostImageCarousel, { props: { images: images(2), title: '测试帖子' } })
    await fireEvent.click(screen.getByRole('button', { name: '点击放大当前图片' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: '放大图片' }))
    expect(screen.getByText('120%')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: '重置缩放' }))
    expect(screen.getByText('100%')).toBeInTheDocument()
    await fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows a local retry action when the current image fails', async () => {
    render(PostImageCarousel, { props: { images: images(1), title: '测试帖子' } })
    await fireEvent.error(screen.getByRole('img'))
    expect(screen.getByText('图片加载失败')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument()
  })
})
