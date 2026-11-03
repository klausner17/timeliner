<script setup lang="ts">
import { TlCard } from '@/design-system'

/** Cartão centralizado das telas de conta: entrar, criar conta, senha, e-mail. */
defineProps<{ title: string; lead?: string; form?: boolean }>()
defineEmits<{ submit: [] }>()
</script>

<template>
  <main class="tl-container auth">
    <TlCard :as="form ? 'form' : 'div'" padding="lg" class="auth__card" :novalidate="form || undefined" @submit.prevent="$emit('submit')">
      <h1 class="auth__title">{{ title }}</h1>
      <p v-if="lead" class="auth__lead">{{ lead }}</p>
      <slot />
    </TlCard>
    <div v-if="$slots.after" class="auth__after"><slot name="after" /></div>
  </main>
</template>

<style scoped>
.auth {
  padding-top: var(--tl-space-14);
  padding-bottom: var(--tl-space-14);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--tl-space-5);
}
.auth__card {
  width: 100%;
  max-width: 420px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.auth__title {
  font-size: var(--tl-text-3xl);
}
.auth__lead {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
  margin-top: calc(var(--tl-space-2) * -1);
}
.auth__after {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
  text-align: center;
}
/* Estilos usados pelos formulários dentro do cartão. */
.auth :deep(.auth__error) {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
.auth :deep(.auth__note) {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.auth :deep(.auth__ok) {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text);
}
.auth :deep(.auth__row) {
  display: flex;
  justify-content: flex-end;
  margin-top: calc(var(--tl-space-2) * -1);
  font-size: var(--tl-text-sm);
}
/* Campo-isca contra robôs: fora da tela, sem foco nem leitor de tela. */
.auth :deep(.auth__trap) {
  position: absolute;
  left: -10000px;
  width: 1px;
  height: 1px;
  overflow: hidden;
}
@media (max-width: 480px) {
  .auth {
    padding-top: var(--tl-space-6);
    padding-bottom: var(--tl-space-8);
  }
}
</style>
