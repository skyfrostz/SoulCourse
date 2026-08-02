import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import NotFoundPage from './NotFoundPage.vue'

describe('NotFoundPage', () => {
  it('renders the immersive branded 404 state', () => {
    const { container } = render(NotFoundPage)

    expect(screen.getByRole('heading', { name: '404' })).toBeInTheDocument()
    expect(screen.getByText('这条选科路径暂时没有内容，回到首页继续探索。')).toBeInTheDocument()
    expect(screen.getByLabelText('SoulCourse')).toBeInTheDocument()
    expect(container.querySelector('video')).toHaveAttribute('autoplay')
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
