import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../pages/HomePage.vue'
import InsightPage from '../pages/InsightPage.vue'
import InsightsOverviewPage from '../pages/InsightsOverviewPage.vue'
import MessagesPage from '../pages/MessagesPage.vue'
import PostDetailPage from '../pages/PostDetailPage.vue'
import RequirementsPage from '../pages/RequirementsPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import AdviceOverviewPage from '../pages/AdviceOverviewPage.vue'
import TopicPage from '../pages/TopicPage.vue'
import TopicsOverviewPage from '../pages/TopicsOverviewPage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomePage },
    { path: '/posts/:id', name: 'post-detail', component: PostDetailPage },
    { path: '/topics', name: 'topics-overview', component: TopicsOverviewPage },
    { path: '/topics/:slug', name: 'topic-detail', component: TopicPage },
    { path: '/insights', name: 'insights-overview', component: InsightsOverviewPage },
    { path: '/insights/:id', name: 'insight-detail', component: InsightPage },
    { path: '/advice', name: 'advice-overview', component: AdviceOverviewPage },
    { path: '/settings', name: 'settings', component: SettingsPage },
    { path: '/messages', name: 'messages', component: MessagesPage },
    { path: '/requirements', name: 'requirements', component: RequirementsPage },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})
