import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import type { UserRole } from '../types/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/nodes' },
    { path: '/login', name: 'login', component: () => import('../pages/LoginPage.vue'), meta: { public: true } },
    { path: '/nodes', name: 'nodes', component: () => import('../pages/NodesPage.vue') },
    { path: '/deviations', name: 'deviations', component: () => import('../pages/DeviationsPage.vue') },
    { path: '/safeguards', name: 'safeguards', component: () => import('../pages/SafeguardsPage.vue') },
    { path: '/coverage', name: 'coverage', component: () => import('../pages/CoveragePage.vue') },
    { path: '/audit', name: 'audit', component: () => import('../pages/AuditPage.vue'), meta: { roles: ['admin', 'safety_reviewer', 'auditor'] } },
    { path: '/:pathMatch(.*)*', redirect: '/nodes' },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.authenticated) return { name: 'login', query: { redirect: to.fullPath } }
  const roles = to.meta.roles as UserRole[] | undefined
  if (roles && !auth.hasRole(...roles)) return { name: 'nodes' }
  if (to.name === 'login' && auth.authenticated) return { name: 'nodes' }
})

export default router
