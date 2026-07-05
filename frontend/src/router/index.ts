import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../pages/HomePage.vue'
import KnowledgeBasePage from '../pages/KnowledgeBasePage.vue'
import ProvinceDetailPage from '../pages/ProvinceDetailPage.vue'
import InsightPage from '../pages/InsightPage.vue'
import InsightsOverviewPage from '../pages/InsightsOverviewPage.vue'
import MessagesPage from '../pages/MessagesPage.vue'
import NotificationsPage from '../pages/NotificationsPage.vue'
import MajorForumPage from '../pages/MajorForumPage.vue'
import PolicyDocumentPage from '../pages/PolicyDocumentPage.vue'
import PostDetailPage from '../pages/PostDetailPage.vue'
import RequirementsPage from '../pages/RequirementsPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import AdviceDetailPage from '../pages/AdviceDetailPage.vue'
import AdviceOverviewPage from '../pages/AdviceOverviewPage.vue'
import FollowingPage from '../pages/FollowingPage.vue'
import TopicPage from '../pages/TopicPage.vue'
import TopicsOverviewPage from '../pages/TopicsOverviewPage.vue'
import UserProfilePage from '../pages/UserProfilePage.vue'
import AdminConsolePage from '../pages/AdminConsolePage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomePage },
    { path: '/posts/:id', name: 'post-detail', component: PostDetailPage },
    { path: '/users/:name', name: 'user-profile', component: UserProfilePage },
    { path: '/topics', name: 'topics-overview', component: TopicsOverviewPage },
    { path: '/topics/:slug', name: 'topic-detail', component: TopicPage },
    { path: '/insights', name: 'insights-overview', component: InsightsOverviewPage },
    { path: '/insights/:id', name: 'insight-detail', component: InsightPage },
    { path: '/advice', name: 'advice-overview', component: AdviceOverviewPage },
    { path: '/advice/:id', name: 'advice-detail', component: AdviceDetailPage },
    { path: '/admin', alias: '/admin/', name: 'admin-console', component: AdminConsolePage, meta: { layout: 'admin' } },
    { path: '/following', name: 'following', component: FollowingPage },
    { path: '/settings', name: 'settings', component: SettingsPage },
    { path: '/messages', name: 'messages', component: MessagesPage },
    { path: '/notifications', name: 'notifications', component: NotificationsPage },
    { path: '/requirements', name: 'requirements', component: RequirementsPage },
    { path: '/requirements/:major', name: 'major-forum', component: MajorForumPage },
    { path: '/knowledge', name: 'knowledge-base', component: KnowledgeBasePage },
    { path: '/knowledge/:province/docs/:documentId', name: 'policy-document', component: PolicyDocumentPage },
    { path: '/knowledge/:province', name: 'province-detail', component: ProvinceDetailPage },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})
