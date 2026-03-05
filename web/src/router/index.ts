import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/DashboardView.vue'),
    meta: { title: '仪表盘', icon: 'dashboard' },
  },
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('../views/SetupView.vue'),
    meta: { title: '任务配置', icon: 'settings' },
  },
  {
    path: '/restore',
    name: 'Restore',
    component: () => import('../views/RestoreView.vue'),
    meta: { title: '数据恢复', icon: 'restore' },
  },
]
