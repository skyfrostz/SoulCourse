import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import NotFoundPage from './NotFoundPage.vue'

describe('NotFoundPage', () => {
  it('renders a usable recovery state instead of a blank route', () => {
    render(NotFoundPage, {
      global: {
        stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } },
      },
    })

    expect(screen.getByRole('heading')).toHaveTextContent('页面不存在')
    expect(screen.getByRole('link', { name: /回到首页/ })).toHaveAttribute('href', '/')
  })
})
