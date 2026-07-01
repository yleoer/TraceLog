import { createRouter, createWebHistory } from 'vue-router'

const DashboardPage = () => import('../pages/DashboardPage.vue')
const TodayPage = () => import('../pages/TodayPage.vue')
const IssueListPage = () => import('../pages/IssueListPage.vue')
const IssueDetailPage = () => import('../pages/IssueDetailPage.vue')
const TempTaskListPage = () => import('../pages/TempTaskListPage.vue')
const TempTaskDetailPage = () => import('../pages/TempTaskDetailPage.vue')
const WeeklyViewPage = () => import('../pages/WeeklyViewPage.vue')
const SearchPage = () => import('../pages/SearchPage.vue')
const SettingsPage = () => import('../pages/SettingsPage.vue')

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
