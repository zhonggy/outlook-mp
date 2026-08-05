import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('../layout/Layout.vue'),
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '仪表盘' } },
        { path: 'accounts', name: 'accounts', component: () => import('../views/Accounts.vue'), meta: { title: '账号管理' } },
        { path: 'accounts/:id', name: 'account-detail', component: () => import('../views/AccountDetail.vue'), meta: { title: '账号详情' } },
        { path: 'tasks', name: 'tasks', component: () => import('../views/TaskLogs.vue'), meta: { title: '任务日志' } },
        { path: 'settings', name: 'settings', component: () => import('../views/Settings.vue'), meta: { title: '系统设置' } },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('om_token')
  if (!to.meta.public && !token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && token) {
    return { path: '/' }
  }
  return true
})

// 标签页标题随路由变化（登录页/无 title 路由用默认名）
router.afterEach((to) => {
  const t = to.meta.title as string | undefined
  document.title = t ? `${t} · Outlook Manager` : 'Outlook Manager'
})

export default router
