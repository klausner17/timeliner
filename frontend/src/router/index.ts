import { createRouter, createWebHistory } from 'vue-router'
import { initSession, session } from '@/data/store'

declare module 'vue-router' {
  interface RouteMeta {
    /** Item do menu principal que fica marcado. */
    nav?: 'explorar' | 'comparar' | 'minhas'
    /** Tela sem o cabeçalho do app (a vitrine tem o próprio). */
    bare?: boolean
    /** Só com login: sem sessão, vai para /entrar e volta depois. */
    auth?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue'), meta: { nav: 'explorar' } },
    { path: '/t/:slug', name: 'timeline', component: () => import('@/views/TimelineView.vue'), meta: { nav: 'explorar' } },
    { path: '/t/:slug/contribuir', name: 'contribute', component: () => import('@/views/ContributeView.vue'), meta: { nav: 'explorar' } },
    { path: '/t/:slug/editar', name: 'editor', component: () => import('@/views/EditorView.vue'), meta: { nav: 'minhas', auth: true } },
    { path: '/nova', name: 'new', component: () => import('@/views/NewTimelineView.vue'), meta: { nav: 'minhas', auth: true } },
    { path: '/calendarios/:id', name: 'calendar', component: () => import('@/views/CalendarView.vue'), meta: { nav: 'minhas' } },
    { path: '/comparar', name: 'compare', component: () => import('@/views/CompareView.vue'), meta: { nav: 'comparar' } },
    { path: '/perfil', name: 'profile', component: () => import('@/views/ProfileView.vue'), meta: { nav: 'minhas', auth: true } },
    { path: '/entrar', name: 'login', component: () => import('@/views/LoginView.vue') },
    { path: '/design-system', name: 'design-system', component: () => import('@/views/DesignSystemView.vue'), meta: { bare: true } },
    { path: '/:rest(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
  ],
  scrollBehavior(to, from, saved) {
    if (saved) return saved
    if (to.hash) return { el: to.hash, top: 16 }
    // Trocar filtros na mesma tela (query) não volta para o topo.
    if (to.path === from.path) return false
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  await initSession()
  if (to.meta.auth && !session.user) return { name: 'login', query: { next: to.fullPath } }
})
