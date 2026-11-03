<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { TlAppHeader, type NavItem } from '@/design-system'
import EmailBanner from '@/components/EmailBanner.vue'
import { currentUser, session, signupOpen } from '@/data/store'

const route = useRoute()

const nav = computed<NavItem[]>(() => [
  { label: 'Explorar', to: '/', current: route.meta.nav === 'explorar' },
  { label: 'Comparar', to: '/comparar', current: route.meta.nav === 'comparar' },
  { label: 'Minhas timelines', to: '/perfil', current: route.meta.nav === 'minhas' },
])

const canSignUp = ref(false)
onMounted(async () => (canSignUp.value = await signupOpen()))
const isAuthPage = computed(() => route.meta.guest === true)
</script>

<template>
  <RouterView v-if="route.meta.bare" />
  <div v-else class="app">
    <TlAppHeader
      :nav="nav"
      :create-to="currentUser ? '/nova' : undefined"
      profile-to="/perfil"
      :user-name="currentUser?.name"
      :login-to="session.ready && route.name !== 'login' ? { name: 'login', query: { next: route.fullPath } } : undefined"
      :signup-to="session.ready && canSignUp && !isAuthPage ? { name: 'signup' } : undefined"
    />
    <EmailBanner v-if="currentUser && !currentUser.emailVerified && route.name !== 'verify-email'" :email="currentUser.email" />
    <div class="app__main"><RouterView /></div>
    <footer class="app-footer">
      <div class="tl-container app-footer__inner">
        <span>Timeliner</span>
        <nav aria-label="Rodapé" class="app-footer__links">
          <RouterLink to="/">Sobre</RouterLink>
          <RouterLink to="/">Regras de contribuição</RouterLink>
          <RouterLink to="/">Privacidade</RouterLink>
        </nav>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.app__main {
  flex: 1 0 auto;
}
.app-footer {
  border-top: 1px solid var(--tl-color-border);
  background: var(--tl-color-surface);
}
.app-footer__inner {
  padding-top: var(--tl-space-6);
  padding-bottom: var(--tl-space-6);
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: var(--tl-space-3);
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.app-footer__links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-5);
}
.app-footer__links a {
  color: var(--tl-color-text-muted);
}
</style>
