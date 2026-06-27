import { createRouter, createWebHistory } from 'vue-router'
import DashboardPage from '../pages/DashboardPage.vue'
import TodayPage from '../pages/TodayPage.vue'
import IssueListPage from '../pages/IssueListPage.vue'
import IssueDetailPage from '../pages/IssueDetailPage.vue'
import TempTaskListPage from '../pages/TempTaskListPage.vue'
import TempTaskDetailPage from '../pages/TempTaskDetailPage.vue'
import WeeklyViewPage from '../pages/WeeklyViewPage.vue'
import SearchPage from '../pages/SearchPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: DashboardPage },
    { path: '/today', component: TodayPage },
    { path: '/issues', component: IssueListPage },
    { path: '/issues/new', component: IssueDetailPage },
    { path: '/issues/:jiraKey', component: IssueDetailPage },
    { path: '/temp-tasks', component: TempTaskListPage },
    { path: '/temp-tasks/new', component: TempTaskDetailPage },
    { path: '/temp-tasks/:id', component: TempTaskDetailPage },
    { path: '/weeks/:week?', component: WeeklyViewPage },
    { path: '/search', component: SearchPage },
    { path: '/settings', component: SettingsPage }
  ]
})

export default router
