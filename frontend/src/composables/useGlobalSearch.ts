import { useRoute, useRouter } from 'vue-router'
import { useForumStore } from '../stores/forum'

function normalizeSearchKeyword(value: string) {
  return value.trim().replace(/\s+/g, ' ')
}

export function useGlobalSearch() {
  const forumStore = useForumStore()
  const route = useRoute()
  const router = useRouter()

  function applySearchKeyword(value: string) {
    forumStore.filter = {
      track: 'all',
      subjects: [],
      category: 'all',
      keyword: normalizeSearchKeyword(value),
      sort: 'recommended',
    }
    forumStore.page = 1
  }

  function searchQueryFromRoute() {
    const value = route.query.q
    return normalizeSearchKeyword(Array.isArray(value) ? value[0] ?? '' : value ?? '')
  }

  function syncSearchFromRoute() {
    if (route.path !== '/') return
    const keyword = searchQueryFromRoute()
    if (forumStore.filter.keyword !== keyword) {
      applySearchKeyword(keyword)
    }
  }

  function runSearch(rawKeyword = forumStore.filter.keyword) {
    const keyword = normalizeSearchKeyword(rawKeyword)
    applySearchKeyword(keyword)
    return router.push({
      path: '/',
      query: keyword ? { q: keyword } : {},
    })
  }

  return { runSearch, syncSearchFromRoute }
}
