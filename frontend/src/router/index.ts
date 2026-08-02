import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../pages/HomePage.vue'
import { useForumStore } from '../stores/forum'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomePage, meta: { title: '选科π' } },
    { path: '/posts/:id', name: 'post-detail', component: () => import('../pages/PostDetailPage.vue'), meta: { title: '帖子详情 - 选科π' } },
    { path: '/users/:name', name: 'user-profile', component: () => import('../pages/UserProfilePage.vue'), meta: { title: '用户主页 - 选科π' } },
    { path: '/topics', name: 'topics-overview', component: () => import('../pages/TopicsOverviewPage.vue'), meta: { title: '话题 - 选科π' } },
    { path: '/topics/:slug', name: 'topic-detail', component: () => import('../pages/TopicPage.vue'), meta: { title: '话题详情 - 选科π' } },
    { path: '/insights', name: 'insights-overview', component: () => import('../pages/InsightsOverviewPage.vue'), meta: { title: '观察 - 选科π' } },
    { path: '/insights/:id', name: 'insight-detail', component: () => import('../pages/InsightPage.vue'), meta: { title: '观察详情 - 选科π' } },
    { path: '/advice', name: 'advice-overview', component: () => import('../pages/AdviceOverviewPage.vue'), meta: { title: '建议 - 选科π' } },
    { path: '/advice/:id', name: 'advice-detail', component: () => import('../pages/AdviceDetailPage.vue'), meta: { title: '建议详情 - 选科π' } },
    { path: '/admin', alias: '/admin/', name: 'admin-console', component: () => import('../pages/AdminConsolePage.vue'), meta: { layout: 'admin', title: '管理后台 - 选科π' } },
    { path: '/following', name: 'following', component: () => import('../pages/FollowingPage.vue'), meta: { requiresAuth: true, title: '关注 - 选科π' } },
    { path: '/settings', name: 'settings', component: () => import('../pages/SettingsPage.vue'), meta: { requiresAuth: true, title: '设置 - 选科π' } },
    { path: '/messages', name: 'messages', component: () => import('../pages/MessagesPage.vue'), meta: { requiresAuth: true, title: '私信 - 选科π' } },
    { path: '/notifications', name: 'notifications', component: () => import('../pages/NotificationsPage.vue'), meta: { requiresAuth: true, title: '通知 - 选科π' } },
    { path: '/observe', name: 'observation', component: () => import('../pages/ObservationPage.vue'), meta: { title: '观察站 - 选科π' } },
    { path: '/requirements', name: 'requirements', component: () => import('../pages/RequirementsPage.vue'), meta: { title: '选科要求 - 选科π' } },
    { path: '/requirements/:major', name: 'major-forum', component: () => import('../pages/MajorForumPage.vue'), meta: { title: '专业要求 - 选科π' } },
    { path: '/knowledge', name: 'knowledge-base', component: () => import('../pages/KnowledgeBasePage.vue'), meta: { title: '政策资料 - 选科π' } },
    { path: '/knowledge/:province/docs/:documentId', name: 'policy-document', component: () => import('../pages/PolicyDocumentPage.vue'), meta: { title: '政策文件 - 选科π' } },
    { path: '/knowledge/:province', name: 'province-detail', component: () => import('../pages/ProvinceDetailPage.vue'), meta: { title: '省份资料 - 选科π' } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../pages/NotFoundPage.vue'), meta: { layout: 'immersive', title: '页面不存在 - SoulCourse' } },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach((to, from) => {
  document.title = typeof to.meta.title === 'string' ? to.meta.title : '选科π'
  if (!to.meta.requiresAuth) return true
  const forumStore = useForumStore()
  if (forumStore.requireAuth(to.fullPath)) return true
  return from.matched.length ? false : '/'
})
