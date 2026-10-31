<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import TlAvatar from './TlAvatar.vue'
import TlButton from './TlButton.vue'
import TlLogo from './TlLogo.vue'

export type NavItem = { label: string; to: RouteLocationRaw; current?: boolean }

withDefaults(
  defineProps<{
    nav: NavItem[]
    home?: RouteLocationRaw
    createTo?: RouteLocationRaw
    profileTo?: RouteLocationRaw
    userName?: string
    /** Sem usuário (userName vazio), mostra "Entrar" apontando para cá. */
    loginTo?: RouteLocationRaw
  }>(),
  { home: '/' },
)
</script>

<template>
  <header class="tl-header">
    <div class="tl-header__inner tl-container">
      <RouterLink :to="home" class="tl-header__brand"><TlLogo /></RouterLink>
      <nav aria-label="Principal" class="tl-header__nav">
        <RouterLink
          v-for="item in nav"
          :key="item.label"
          :to="item.to"
          class="tl-header__link"
          :class="{ 'tl-header__link--on': item.current }"
          :aria-current="item.current ? 'page' : undefined"
          >{{ item.label }}</RouterLink
        >
      </nav>
      <TlButton v-if="createTo" :to="createTo" variant="dark" icon="plus" class="tl-header__create" aria-label="Criar timeline"
        >Criar<span class="tl-header__create-more"> timeline</span></TlButton
      >
      <RouterLink v-if="profileTo && userName" :to="profileTo" class="tl-header__profile" aria-label="Seu perfil">
        <TlAvatar :name="userName" tint="a" decorative />
      </RouterLink>
      <TlButton v-else-if="loginTo" :to="loginTo">Entrar</TlButton>
    </div>
  </header>
</template>

<style scoped>
.tl-header {
  background: var(--tl-color-surface);
  border-bottom: 1px solid var(--tl-color-border);
}
.tl-header__inner {
  padding-top: var(--tl-space-3);
  padding-bottom: var(--tl-space-3);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-3) var(--tl-space-7);
}
.tl-header__brand {
  display: flex;
  align-items: center;
  min-height: 44px;
  text-decoration: none;
}
.tl-header__nav {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-1);
  flex: 1 1 auto;
}
.tl-header__link {
  padding: var(--tl-space-3);
  text-decoration: none;
  color: var(--tl-color-text-muted);
  font-weight: var(--tl-weight-medium);
  font-size: var(--tl-text-base);
  border-bottom: 2px solid transparent;
}
.tl-header__link:hover {
  color: var(--tl-color-text);
}
.tl-header__link--on {
  color: var(--tl-color-text);
  font-weight: var(--tl-weight-semibold);
  border-bottom-color: var(--tl-color-text);
}
.tl-header__profile {
  display: inline-flex;
  text-decoration: none;
  border-radius: var(--tl-radius-pill);
  box-shadow: 0 0 0 1px var(--tl-color-border);
}

/* Em telas estreitas: logo, criar e perfil na primeira linha; o menu embaixo, rolando de lado se faltar espaço. */
@media (max-width: 860px) {
  .tl-header__inner {
    padding-bottom: 0;
    gap: 0 var(--tl-space-3);
  }
  .tl-header__brand {
    flex: 1 1 auto;
  }
  .tl-header__nav {
    order: 1;
    flex: 1 1 100%;
    flex-wrap: nowrap;
    gap: 0;
    margin: 0 calc(var(--tl-gutter) * -1);
    padding: 0 calc(var(--tl-gutter) - var(--tl-space-3));
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tl-header__link {
    flex: none;
    white-space: nowrap;
  }
}
@media (max-width: 400px) {
  .tl-header__create-more {
    display: none;
  }
}
/* Em telas bem estreitas o botão fica só com o "+". */
@media (max-width: 359px) {
  .tl-header__create {
    font-size: 0;
    gap: 0;
    padding: 0 12px;
  }
}
</style>
