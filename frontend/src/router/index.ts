import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // As telas do produto entram aqui. Por enquanto, a vitrine é a página inicial.
    { path: '/', redirect: '/design-system' },
    { path: '/design-system', name: 'design-system', component: () => import('@/views/DesignSystemView.vue') },
  ],
  scrollBehavior(to) {
    return to.hash ? { el: to.hash, top: 16 } : { top: 0 }
  },
})
